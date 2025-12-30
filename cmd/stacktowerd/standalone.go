package main

import (
	"context"
	"fmt"
	"time"

	"github.com/matzehuels/stacktower/internal/api"
	"github.com/matzehuels/stacktower/internal/worker"
	"github.com/matzehuels/stacktower/pkg/core/deps"
	"github.com/matzehuels/stacktower/pkg/core/deps/languages"
	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/session"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

// runStandalone starts the daemon in standalone mode.
// This is an all-in-one mode with no external dependencies:
// - In-memory job queue
// - In-memory caching and storage
// - In-memory session/state stores
// - Optional authentication bypass with --no-auth
// - API + Worker run together in single process
func runStandalone(ctx context.Context, cfg serverConfig) error {
	log := infra.DefaultLogger()
	log.Info("running in STANDALONE mode (no external dependencies)")
	log.Info("note: data will not persist across restarts")

	if cfg.noAuth {
		log.Warn("authentication DISABLED (--no-auth flag)")
		log.Warn("all requests will be treated as coming from 'local' user")
	}

	// Create in-memory components
	memQueue := queue.NewMemoryQueue()
	memBackend := storage.NewMemoryBackend()
	memHTTPCache := storage.NewMemoryHTTPCache()

	// Wrap MemoryBackend in DistributedBackend interface for API/Worker compatibility
	// MemoryBackend implements Index, DocumentStore, RateLimiter
	// MemoryHTTPCache implements HTTPCache
	backend := storage.NewDistributedBackend(
		memBackend,   // Index
		memBackend,   // DocumentStore
		memHTTPCache, // HTTPCache
		memBackend,   // RateLimiter
	)

	// Create API server with relaxed security
	opts := []api.Option{
		api.WithHost(cfg.host),
		api.WithPort(cfg.port),
		api.WithSessions(session.NewMemoryStore()),
		api.WithStates(session.NewMemoryStateStore()),
		api.WithManifestPatterns(deps.SupportedManifests(languages.All)),
		api.WithLogger(log),
		api.WithFrontendURL("http://localhost:5173"),
		api.WithSecureCookies(false),
		api.WithAllowOrigins([]string{"*"}), // Allow all origins in standalone
		// GitHub OAuth is optional in standalone mode
		// If GITHUB_CLIENT_ID/SECRET env vars are set, OAuth will work
	}

	// Enable no-auth mode if requested
	if cfg.noAuth {
		opts = append(opts, api.WithNoAuth(true))
	}

	server := api.New(memQueue, backend, opts...)

	// Create worker
	w := worker.New(memQueue, backend, worker.Config{
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

	log.Info("standalone server ready",
		"url", fmt.Sprintf("http://%s:%d", cfg.host, cfg.port),
		"workers", cfg.concurrency,
	)

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

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	log.Info("shutting down worker")
	if err := w.Shutdown(shutdownCtx); err != nil {
		log.Warn("worker shutdown error", "error", err)
	}

	log.Info("shutting down server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Warn("server shutdown error", "error", err)
	}

	return shutdownErr
}
