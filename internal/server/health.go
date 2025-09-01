package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gravitational/trace"
)

// HealthStatus represents the health status of the service
type HealthStatus struct {
	Status               string    `json:"status"`
	TeleportConnected    bool      `json:"teleport_connected"`
	IdentityValid        bool      `json:"identity_valid"`
	LastRequestProcessed time.Time `json:"last_request_processed"`
	LastIdentityRefresh  time.Time `json:"last_identity_refresh"`
	Uptime               string    `json:"uptime"`
}

// HealthServer provides HTTP health check endpoints
type HealthServer struct {
	port              int
	path              string
	server            *http.Server
	logger            *log.Logger
	mu                sync.RWMutex
	teleportConnected bool
	identityValid     bool
	lastRequestTime   time.Time
	lastIdentityTime  time.Time
	startTime         time.Time
}

// NewHealthServer creates a new health check server
func NewHealthServer(port int, path string, logger *log.Logger) *HealthServer {
	return &HealthServer{
		port:      port,
		path:      path,
		logger:    logger,
		startTime: time.Now(),
	}
}

// Start starts the health check HTTP server
func (h *HealthServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc(h.path, h.healthHandler)

	h.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", h.port),
		Handler: mux,
	}

	h.logger.Printf("Health check server starting on port %d, path %s", h.port, h.path)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.server.Shutdown(shutdownCtx); err != nil {
			h.logger.Printf("Health server shutdown error: %v", err)
		}
	}()

	if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return trace.Wrap(err, "health server failed")
	}

	return nil
}

// Stop stops the health check server
func (h *HealthServer) Stop() error {
	if h.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return h.server.Shutdown(ctx)
	}
	return nil
}

// SetTeleportStatus updates the Teleport connection status
func (h *HealthServer) SetTeleportStatus(connected bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.teleportConnected = connected
}

// SetIdentityStatus updates the identity validity status
func (h *HealthServer) SetIdentityStatus(valid bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.identityValid = valid
	if valid {
		h.lastIdentityTime = time.Now()
	}
}

// UpdateLastRequestTime updates the last request processing time
func (h *HealthServer) UpdateLastRequestTime() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastRequestTime = time.Now()
}

// healthHandler handles health check requests
func (h *HealthServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.mu.RLock()
	status := HealthStatus{
		TeleportConnected:    h.teleportConnected,
		IdentityValid:        h.identityValid,
		LastRequestProcessed: h.lastRequestTime,
		LastIdentityRefresh:  h.lastIdentityTime,
		Uptime:               time.Since(h.startTime).String(),
	}
	h.mu.RUnlock()

	// Determine overall status
	if status.TeleportConnected && status.IdentityValid {
		status.Status = "healthy"
		w.WriteHeader(http.StatusOK)
	} else {
		status.Status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		h.logger.Printf("Failed to encode health status: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
