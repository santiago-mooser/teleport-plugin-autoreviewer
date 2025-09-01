package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"teleport-autoreviewer/teleport"
)

// HealthStatus represents the overall health status
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
	port      int
	path      string
	server    *http.Server
	logger    *log.Logger
	client    *teleport.Client
	startTime time.Time
	mu        sync.RWMutex
}

// NewHealthServer creates a new health check server
func NewHealthServer(port int, path string, client *teleport.Client, logger *log.Logger) *HealthServer {
	return &HealthServer{
		port:      port,
		path:      path,
		client:    client,
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
		return fmt.Errorf("health server failed: %w", err)
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

// healthHandler handles health check requests
func (h *HealthServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.mu.RLock()
	teleportHealth := h.client.GetHealthStatus()
	lastRequestTime := h.client.GetLastRequestTime()
	h.mu.RUnlock()

	status := HealthStatus{
		TeleportConnected:    teleportHealth.TeleportConnected,
		IdentityValid:        teleportHealth.IdentityValid,
		LastRequestProcessed: lastRequestTime,
		LastIdentityRefresh:  teleportHealth.LastRefresh,
		Uptime:               time.Since(h.startTime).String(),
	}

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
