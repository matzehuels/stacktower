package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/internal/api"
	"github.com/matzehuels/stacktower/pkg/core/deps"
	"github.com/matzehuels/stacktower/pkg/core/deps/languages"
	"github.com/matzehuels/stacktower/pkg/infra"
)

// runAPIOnly starts only the HTTP API server.
func runAPIOnly(ctx context.Context, comp *serverComponents, cfg serverConfig) error {
	log := infra.DefaultLogger()
	log.Info("starting server (API-only mode)", "host", cfg.host, "port", cfg.port)
	log.Info("note: workers must be started separately to process jobs")

	server := newAPIServer(comp, cfg)

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

	w := api.NewWorker(comp.queue, comp.backend, api.WorkerConfig{
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

// runLocal starts both API server and worker in the same process (development mode).
func runLocal(ctx context.Context, comp *serverComponents, cfg serverConfig) error {
	log := infra.DefaultLogger()
	log.Info("starting in LOCAL mode", "host", cfg.host, "port", cfg.port, "concurrency", cfg.concurrency)

	server := newAPIServer(comp, cfg)

	w := api.NewWorker(comp.queue, comp.backend, api.WorkerConfig{
		Concurrency:  cfg.concurrency,
		PollInterval: cfg.pollInterval,
		Logger:       log,
	})

	// Start server and worker concurrently.
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Start()
	}()

	workerErrCh := make(chan error, 1)
	go func() {
		workerErrCh <- w.Start(ctx)
	}()

	// Wait for shutdown signal or error.
	var shutdownErr error
	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-serverErrCh:
		shutdownErr = fmt.Errorf("server error: %w", err)
	case err := <-workerErrCh:
		shutdownErr = fmt.Errorf("worker error: %w", err)
	}

	// Graceful shutdown with timeout.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Shutdown worker first (drain in-flight jobs).
	log.Info("shutting down worker")
	if err := w.Shutdown(shutdownCtx); err != nil {
		log.Warn("worker shutdown error", "error", err)
	}

	// Then shutdown server.
	log.Info("shutting down server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Warn("server shutdown error", "error", err)
	}

	return shutdownErr
}

// newAPIServer creates a configured API server from components and config.
func newAPIServer(comp *serverComponents, cfg serverConfig) *api.Server {
	log := infra.DefaultLogger()

	// Enable secure cookies in production (when not using localhost frontend).
	secureCookies := !strings.Contains(comp.frontendURL, "localhost")

	return api.New(comp.queue, comp.backend,
		api.WithHost(cfg.host),
		api.WithPort(cfg.port),
		api.WithSessions(comp.sessionStore),
		api.WithStates(comp.stateStore),
		api.WithManifestPatterns(deps.SupportedManifests(languages.All)),
		api.WithLogger(log),
		api.WithGitHubOAuth(comp.githubOAuth),
		api.WithFrontendURL(comp.frontendURL),
		api.WithSecureCookies(secureCookies),
	)
}
