package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/internal/api"
	"github.com/matzehuels/stacktower/internal/worker"
	"github.com/matzehuels/stacktower/pkg/core/deps"
	"github.com/matzehuels/stacktower/pkg/core/deps/languages"
	"github.com/matzehuels/stacktower/pkg/infra"
)

// runDistributed starts the daemon in distributed mode (compose or production).
// This mode requires Redis + MongoDB infrastructure.
func runDistributed(ctx context.Context, cfg serverConfig, production bool) error {
	log := infra.DefaultLogger()

	modeName := "compose"
	if production {
		modeName = "production"
	}
	log.Info(fmt.Sprintf("running in %s mode (distributed)", strings.ToUpper(modeName)))

	// Create distributed infrastructure (Redis + MongoDB)
	clients, err := createInfra(ctx, production)
	if err != nil {
		return err
	}
	defer clients.redis.Close()
	defer clients.mongo.Close()

	// Create server components from infrastructure
	comp, err := createComponents(clients, production)
	if err != nil {
		return err
	}
	defer comp.backend.Close()

	// Run the appropriate role
	switch cfg.role {
	case RoleAPI:
		return runAPIOnly(ctx, comp, cfg, production)
	case RoleWorker:
		return runWorkerOnly(ctx, comp, cfg)
	case RoleAll:
		return runCombined(ctx, comp, cfg, production)
	default:
		return fmt.Errorf("unknown role: %s", cfg.role)
	}
}

// runAPIOnly starts only the HTTP API server.
func runAPIOnly(ctx context.Context, comp *serverComponents, cfg serverConfig, production bool) error {
	log := infra.DefaultLogger()
	log.Info("starting API server (API-only mode)", "host", cfg.host, "port", cfg.port)
	log.Info("note: workers must be started separately to process jobs")

	server := newDistributedAPIServer(comp, cfg, production)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start()
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// runWorkerOnly starts only the background worker.
func runWorkerOnly(ctx context.Context, comp *serverComponents, cfg serverConfig) error {
	log := infra.DefaultLogger()
	log.Info("starting worker (worker-only mode)", "concurrency", cfg.concurrency)

	w := worker.New(comp.queue, comp.backend, worker.Config{
		Concurrency:  cfg.concurrency,
		PollInterval: cfg.pollInterval,
		Logger:       log,
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Start(ctx)
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return w.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// runCombined starts both API server and worker in the same process.
func runCombined(ctx context.Context, comp *serverComponents, cfg serverConfig, production bool) error {
	log := infra.DefaultLogger()
	log.Info("starting combined API + Worker",
		"host", cfg.host,
		"port", cfg.port,
		"concurrency", cfg.concurrency,
	)

	server := newDistributedAPIServer(comp, cfg, production)

	w := worker.New(comp.queue, comp.backend, worker.Config{
		Concurrency:  cfg.concurrency,
		PollInterval: cfg.pollInterval,
		Logger:       log,
	})

	// Start server and worker concurrently
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Start()
	}()

	workerErrCh := make(chan error, 1)
	go func() {
		workerErrCh <- w.Start(ctx)
	}()

	// Wait for shutdown signal or error
	var shutdownErr error
	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-serverErrCh:
		shutdownErr = fmt.Errorf("server error: %w", err)
	case err := <-workerErrCh:
		shutdownErr = fmt.Errorf("worker error: %w", err)
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Shutdown worker first (drain in-flight jobs)
	log.Info("shutting down worker")
	if err := w.Shutdown(shutdownCtx); err != nil {
		log.Warn("worker shutdown error", "error", err)
	}

	// Then shutdown server
	log.Info("shutting down server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Warn("server shutdown error", "error", err)
	}

	return shutdownErr
}

// newDistributedAPIServer creates a configured API server for distributed modes.
func newDistributedAPIServer(comp *serverComponents, cfg serverConfig, production bool) *api.Server {
	log := infra.DefaultLogger()

	// Enable secure cookies in production (when not using localhost frontend)
	isLocalhost := strings.Contains(comp.frontendURL, "localhost") ||
		strings.Contains(comp.frontendURL, "127.0.0.1")
	secureCookies := production && !isLocalhost

	// Warn about trust-proxy configuration
	if cfg.trustProxyHeaders {
		if isLocalhost {
			log.Warn("--trust-proxy enabled in development mode; this is safe for local testing")
		} else if !secureCookies {
			log.Warn("--trust-proxy enabled without HTTPS; ensure you're behind a trusted proxy")
		}
	}

	opts := []api.Option{
		api.WithHost(cfg.host),
		api.WithPort(cfg.port),
		api.WithSessions(comp.sessionStore),
		api.WithStates(comp.stateStore),
		api.WithManifestPatterns(deps.SupportedManifests(languages.All)),
		api.WithLogger(log),
		api.WithGitHubOAuth(comp.githubOAuth),
		api.WithFrontendURL(comp.frontendURL),
		api.WithSecureCookies(secureCookies),
		api.WithTrustProxyHeaders(cfg.trustProxyHeaders),
	}

	// In development mode, allow all origins for easier testing
	if !production {
		opts = append(opts, api.WithAllowOrigins([]string{"*"}))
	}

	return api.New(comp.queue, comp.backend, opts...)
}
