package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"teleport-autoreviewer/config"
	"teleport-autoreviewer/server"
	"teleport-autoreviewer/teleport"

	"github.com/gravitational/trace"
	"gopkg.in/yaml.v2"
)

func main() {
	logger := log.New(os.Stdout, "[teleport-autoreviewer] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Starting Teleport Auto-reviewer")

	if err := run(logger); err != nil {
		logger.Printf("error: %v", err)
		os.Exit(1)
	}
}

func run(logger *log.Logger) error {
	// Load configuration
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		return trace.Wrap(err)
	}

	logger.Printf("Loaded configuration with %d rejection rules", len(cfg.Rejection.Rules))

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Teleport client
	client, err := teleport.New(ctx, cfg, logger)
	if err != nil {
		return trace.Wrap(err, "failed to create Teleport client")
	}

	// Create health server
	healthServer := server.NewHealthServer(
		cfg.Server.HealthPort,
		cfg.Server.HealthPath,
		client,
		logger,
	)

	// Create a wait group for goroutines
	var wg sync.WaitGroup

	// Start health server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := healthServer.Start(ctx); err != nil {
			logger.Printf("Health server error: %v", err)
		}
	}()

	// Start identity refresh routine if configured
	if cfg.Teleport.IdentityRefreshInterval > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runIdentityRefresh(ctx, client, cfg.Teleport.IdentityRefreshInterval, logger)
		}()
		logger.Printf("Identity refresh enabled with interval: %v", cfg.Teleport.IdentityRefreshInterval)
	}

	// Start access request watcher
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := client.WatchAccessRequests(ctx); err != nil {
			logger.Printf("Access request watcher error: %v", err)
		}
	}()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	logger.Println("Teleport Auto-reviewer started successfully")
	logger.Printf("Health endpoint available at http://localhost:%d%s", cfg.Server.HealthPort, cfg.Server.HealthPath)

	// Wait for shutdown signal
	<-sigCh
	logger.Println("Received shutdown signal")

	// Cancel context to signal all goroutines to stop
	cancel()

	// Create shutdown timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Println("All services stopped gracefully")
	case <-shutdownCtx.Done():
		logger.Println("Shutdown timeout reached, forcing exit")
	}

	return nil
}

// runIdentityRefresh runs the identity refresh routine
func runIdentityRefresh(ctx context.Context, client *teleport.Client, interval time.Duration, logger *log.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := client.RefreshIdentity(ctx); err != nil {
				logger.Printf("Failed to refresh identity: %v", err)
			} else {
				logger.Printf("Identity refreshed successfully")
			}
		}
	}
}

// loadConfig loads the configuration from the specified file
func loadConfig(path string) (*config.Config, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, trace.Wrap(err, "failed to read config file")
	}

	var cfg config.Config
	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return nil, trace.Wrap(err, "failed to parse config file")
	}

	// Set defaults if not specified
	if cfg.Server.HealthPort == 0 {
		cfg.Server.HealthPort = 8080
	}
	if cfg.Server.HealthPath == "" {
		cfg.Server.HealthPath = "/health"
	}
	if cfg.Rejection.DefaultMessage == "" {
		cfg.Rejection.DefaultMessage = "Access request rejected due to policy violation"
	}
	if cfg.Teleport.IdentityRefreshInterval == 0 {
		cfg.Teleport.IdentityRefreshInterval = time.Hour // Default to 1 hour
	}

	return &cfg, nil
}
