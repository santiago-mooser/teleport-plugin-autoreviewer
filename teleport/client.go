package teleport

import (
	"context"
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/gravitational/teleport/api/client"
	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/trace"

	"teleport-autoreviewer/config"
)

// Client is a Teleport client with auto-rejection capabilities
type Client struct {
	*client.Client
	config          *config.Config
	logger          *log.Logger
	mu              sync.RWMutex
	compiledRules   []*CompiledRule
	healthStatus    *HealthStatus
	lastRequestTime time.Time
}

// CompiledRule contains a compiled regex rule for efficient matching
type CompiledRule struct {
	Name        string
	ReasonRegex *regexp.Regexp
	RolesRegex  *regexp.Regexp
	Message     string
}

// HealthStatus tracks the health of the teleport client
type HealthStatus struct {
	TeleportConnected bool
	IdentityValid     bool
	LastRefresh       time.Time
}

// New creates a new Teleport client
func New(ctx context.Context, cfg *config.Config, logger *log.Logger) (*Client, error) {
	creds := client.LoadIdentityFile(cfg.Teleport.Identity)

	c, err := client.New(ctx, client.Config{
		Addrs:       []string{cfg.Teleport.Addr},
		Credentials: []client.Credentials{creds},
	})
	if err != nil {
		return nil, trace.Wrap(err)
	}

	client := &Client{
		Client: c,
		config: cfg,
		logger: logger,
		healthStatus: &HealthStatus{
			TeleportConnected: true,
			IdentityValid:     true,
			LastRefresh:       time.Now(),
		},
	}

	// Compile regex rules
	if err := client.compileRules(); err != nil {
		return nil, trace.Wrap(err, "failed to compile rejection rules")
	}

	return client, nil
}

// GetHealthStatus returns the current health status
func (c *Client) GetHealthStatus() *HealthStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return &HealthStatus{
		TeleportConnected: c.healthStatus.TeleportConnected,
		IdentityValid:     c.healthStatus.IdentityValid,
		LastRefresh:       c.healthStatus.LastRefresh,
	}
}

// GetLastRequestTime returns the last request processing time
func (c *Client) GetLastRequestTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastRequestTime
}

// RefreshIdentity refreshes the identity file and reconnects
func (c *Client) RefreshIdentity(ctx context.Context) error {
	c.logger.Printf("Refreshing identity from %s", c.config.Teleport.Identity)

	creds := client.LoadIdentityFile(c.config.Teleport.Identity)

	newClient, err := client.New(ctx, client.Config{
		Addrs:       []string{c.config.Teleport.Addr},
		Credentials: []client.Credentials{creds},
	})
	if err != nil {
		c.mu.Lock()
		c.healthStatus.TeleportConnected = false
		c.healthStatus.IdentityValid = false
		c.mu.Unlock()
		return trace.Wrap(err)
	}

	// Replace the underlying client
	c.mu.Lock()
	if c.Client != nil {
		c.Client.Close()
	}
	c.Client = newClient
	c.healthStatus.TeleportConnected = true
	c.healthStatus.IdentityValid = true
	c.healthStatus.LastRefresh = time.Now()
	c.mu.Unlock()

	c.logger.Printf("Successfully refreshed identity and reconnected")
	return nil
}

// compileRules compiles all regex patterns for efficient matching
func (c *Client) compileRules() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.compiledRules = make([]*CompiledRule, 0, len(c.config.Rejection.Rules))

	for _, rule := range c.config.Rejection.Rules {
		compiledRule := &CompiledRule{
			Name:    rule.Name,
			Message: rule.Message,
		}

		// Compile reason regex if provided
		if rule.ReasonRegex != "" {
			reasonRegex, err := regexp.Compile(rule.ReasonRegex)
			if err != nil {
				return trace.Wrap(err, "failed to compile reason regex for rule %s", rule.Name)
			}
			compiledRule.ReasonRegex = reasonRegex
		}

		// Compile roles regex if provided
		if rule.RolesRegex != "" {
			rolesRegex, err := regexp.Compile(rule.RolesRegex)
			if err != nil {
				return trace.Wrap(err, "failed to compile roles regex for rule %s", rule.Name)
			}
			compiledRule.RolesRegex = rolesRegex
		}

		c.compiledRules = append(c.compiledRules, compiledRule)
	}

	c.logger.Printf("Compiled %d rejection rules", len(c.compiledRules))
	return nil
}

// WatchAccessRequests watches for access requests and rejects them based on configured rules
func (c *Client) WatchAccessRequests(ctx context.Context) error {
	watcher, err := c.NewWatcher(ctx, types.Watch{
		Kinds: []types.WatchKind{
			{
				Kind: types.KindAccessRequest,
			},
		},
	})
	if err != nil {
		c.mu.Lock()
		c.healthStatus.TeleportConnected = false
		c.mu.Unlock()
		return trace.Wrap(err)
	}
	defer watcher.Close()

	c.logger.Printf("Started watching access requests")

	// First, let's check for existing pending requests
	c.logger.Printf("Checking for existing pending access requests...")
	if err := c.processExistingRequests(ctx); err != nil {
		c.logger.Printf("Error processing existing requests: %v", err)
	}

	for {
		select {
		case event := <-watcher.Events():
			c.logger.Printf("Received event: Type=%s, Kind=%s", event.Type, event.Resource.GetKind())

			if event.Type != types.OpPut {
				c.logger.Printf("Ignoring event type: %s", event.Type)
				continue
			}

			req, ok := event.Resource.(types.AccessRequest)
			if !ok {
				c.logger.Printf("Event resource is not an AccessRequest: %T", event.Resource)
				continue
			}

			c.logger.Printf("Processing access request %s, state: %s, reason: %s", req.GetName(), req.GetState(), req.GetRequestReason())

			if req.GetState() != types.RequestState_PENDING {
				c.logger.Printf("Request %s is not pending (state: %s), ignoring", req.GetName(), req.GetState())
				continue
			}

			// Update last request time
			c.mu.Lock()
			c.lastRequestTime = time.Now()
			c.mu.Unlock()

			// Check if request should be rejected
			if rule := c.shouldReject(req); rule != nil {
				if err := c.rejectRequest(ctx, req, rule); err != nil {
					c.logger.Printf("Failed to reject request %s: %v", req.GetName(), err)
				} else {
					c.logger.Printf("Rejected request %s using rule '%s': %s", req.GetName(), rule.Name, rule.Message)
				}
			} else {
				c.logger.Printf("Request %s does not match any rejection rules, allowing to proceed", req.GetName())
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// processExistingRequests checks for existing pending requests and processes them
func (c *Client) processExistingRequests(ctx context.Context) error {
	requests, err := c.GetAccessRequests(ctx, types.AccessRequestFilter{
		State: types.RequestState_PENDING,
	})
	if err != nil {
		return trace.Wrap(err, "failed to get existing access requests")
	}

	c.logger.Printf("Found %d existing pending requests", len(requests))

	for _, req := range requests {
		c.logger.Printf("Processing existing request %s, reason: %s", req.GetName(), req.GetRequestReason())

		// Update last request time
		c.mu.Lock()
		c.lastRequestTime = time.Now()
		c.mu.Unlock()

		// Check if request should be rejected
		if rule := c.shouldReject(req); rule != nil {
			if err := c.rejectRequest(ctx, req, rule); err != nil {
				c.logger.Printf("Failed to reject existing request %s: %v", req.GetName(), err)
			} else {
				c.logger.Printf("Rejected existing request %s using rule '%s': %s", req.GetName(), rule.Name, rule.Message)
			}
		} else {
			c.logger.Printf("Existing request %s does not match any rejection rules, allowing to proceed", req.GetName())
		}
	}

	return nil
}

// shouldReject checks if a request should be rejected based on configured rules
// Uses two-stage filtering: 1) Role filter (does rule apply?), 2) Reason check (should reject?)
func (c *Client) shouldReject(req types.AccessRequest) *CompiledRule {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, rule := range c.compiledRules {
		// Stage 1: Role Filter - Does this rule apply to this request?
		ruleApplies := false
		if rule.RolesRegex != nil {
			// Rule has role filter - check if any requested role matches
			for _, role := range req.GetRoles() {
				if rule.RolesRegex.MatchString(role) {
					ruleApplies = true
					c.logger.Printf("Rule '%s' applies to request %s - role '%s' matches pattern '%s'",
						rule.Name, req.GetName(), role, rule.RolesRegex.String())
					break
				}
			}
			if !ruleApplies {
				c.logger.Printf("Rule '%s' does not apply to request %s - no roles %v match pattern '%s'",
					rule.Name, req.GetName(), req.GetRoles(), rule.RolesRegex.String())
				continue // Skip this rule, doesn't apply to these roles
			}
		} else {
			// No role filter - rule applies to all requests
			ruleApplies = true
			c.logger.Printf("Rule '%s' applies to request %s - no role filter specified",
				rule.Name, req.GetName())
		}

		// Stage 2: Reason Check - Should we reject based on reason?
		if rule.ReasonRegex != nil {
			if !rule.ReasonRegex.MatchString(req.GetRequestReason()) {
				c.logger.Printf("Request %s reason '%s' does NOT match required pattern '%s' in rule '%s' - rejecting",
					req.GetName(), req.GetRequestReason(), rule.ReasonRegex.String(), rule.Name)
				return rule // Reject: reason doesn't match required pattern
			} else {
				c.logger.Printf("Request %s reason '%s' matches required pattern '%s' in rule '%s' - allowing",
					req.GetName(), req.GetRequestReason(), rule.ReasonRegex.String(), rule.Name)
			}
		} else {
			c.logger.Printf("Rule '%s' has no reason filter - allowing request %s",
				rule.Name, req.GetName())
		}
	}

	return nil // Allow: no rules triggered rejection
}

// rejectRequest rejects an access request with the specified rule's message
func (c *Client) rejectRequest(ctx context.Context, req types.AccessRequest, rule *CompiledRule) error {
	message := rule.Message
	if message == "" {
		message = c.config.Rejection.DefaultMessage
	}

	return c.SetAccessRequestState(ctx, types.AccessRequestUpdate{
		RequestID: req.GetName(),
		State:     types.RequestState_DENIED,
		Reason:    message,
	})
}
