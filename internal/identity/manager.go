package identity

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gravitational/teleport/api/client"
	"github.com/gravitational/trace"
)

// Manager handles automatic refresh of Teleport identity files
type Manager struct {
	identityPath    string
	refreshInterval time.Duration
	currentCreds    client.Credentials
	mu              sync.RWMutex
	stopCh          chan struct{}
	credUpdateCh    chan client.Credentials
	logger          *log.Logger
}

// NewManager creates a new identity manager
func NewManager(identityPath string, refreshInterval time.Duration, logger *log.Logger) *Manager {
	return &Manager{
		identityPath:    identityPath,
		refreshInterval: refreshInterval,
		stopCh:          make(chan struct{}),
		credUpdateCh:    make(chan client.Credentials, 1),
		logger:          logger,
	}
}

// Start begins the identity refresh process
func (m *Manager) Start(ctx context.Context) error {
	// Load initial credentials
	creds, err := m.loadCredentials()
	if err != nil {
		return trace.Wrap(err, "failed to load initial credentials")
	}

	m.mu.Lock()
	m.currentCreds = creds
	m.mu.Unlock()

	// Send initial credentials
	select {
	case m.credUpdateCh <- creds:
	default:
	}

	// Start refresh ticker
	ticker := time.NewTicker(m.refreshInterval)
	defer ticker.Stop()

	m.logger.Printf("Identity manager started, refresh interval: %v", m.refreshInterval)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.stopCh:
			return nil
		case <-ticker.C:
			if err := m.refreshCredentials(); err != nil {
				m.logger.Printf("Failed to refresh credentials: %v", err)
			}
		}
	}
}

// Stop stops the identity refresh process
func (m *Manager) Stop() {
	close(m.stopCh)
}

// GetCredentials returns the current credentials
func (m *Manager) GetCredentials() client.Credentials {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentCreds
}

// CredentialUpdates returns a channel that receives credential updates
func (m *Manager) CredentialUpdates() <-chan client.Credentials {
	return m.credUpdateCh
}

// refreshCredentials loads and updates credentials if the file has changed
func (m *Manager) refreshCredentials() error {
	// Check if file has been modified
	stat, err := os.Stat(m.identityPath)
	if err != nil {
		return trace.Wrap(err, "failed to stat identity file")
	}

	// For simplicity, we'll always try to reload the credentials
	// In a production system, you might want to check modification time
	newCreds, err := m.loadCredentials()
	if err != nil {
		return trace.Wrap(err, "failed to load credentials")
	}

	// Update credentials
	m.mu.Lock()
	m.currentCreds = newCreds
	m.mu.Unlock()

	// Notify about credential update
	select {
	case m.credUpdateCh <- newCreds:
		m.logger.Printf("Credentials refreshed from %s (mod time: %v)", m.identityPath, stat.ModTime())
	default:
		// Channel is full, skip this update
	}

	return nil
}

// loadCredentials loads credentials from the identity file
func (m *Manager) loadCredentials() (client.Credentials, error) {
	creds := client.LoadIdentityFile(m.identityPath)
	return creds, nil
}
