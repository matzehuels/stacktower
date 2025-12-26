package api

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/internal/worker"
	"github.com/matzehuels/stacktower/pkg/core/deps"
	"github.com/matzehuels/stacktower/pkg/core/deps/languages"
	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/infra/cache"
	"github.com/matzehuels/stacktower/pkg/infra/common"
	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/session"
)

// Execute runs the API server CLI.
func Execute() error {
	var (
		port         int
		host         string
		workerMode   bool
		localMode    bool
		concurrency  int
		pollInterval time.Duration
	)

	rootCmd := &cobra.Command{
		Use:   "stacktower-api",
		Short: "Stacktower API server for async dependency visualization",
		Long: `Stacktower API server provides a REST API for asynchronous dependency
resolution and visualization. Jobs are queued and processed by workers,
enabling horizontal scaling and long-running operations.

The API server requires Redis and MongoDB for production operation:
  - Queue: Redis Streams (async job processing)
  - Sessions: Redis (user sessions with TTL)
  - Cache: Redis (lookup) + MongoDB (storage)

Modes:
  - API-only: Run just the HTTP API (default)
  - Worker-only: Run just the worker (--worker flag)
  - Local: Run both API and worker in same process (--local flag)

Environment variables (required):
  STACKTOWER_REDIS_ADDR=host:port     Redis server address
  STACKTOWER_MONGODB_URI=mongodb://   MongoDB connection string

Environment variables (optional):
  STACKTOWER_REDIS_PASSWORD=secret    Redis password
  STACKTOWER_MONGODB_DATABASE=name    MongoDB database (default: stacktower)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cmd.Context(), serverConfig{
				port:         port,
				host:         host,
				workerMode:   workerMode,
				localMode:    localMode,
				concurrency:  concurrency,
				pollInterval: pollInterval,
			})
		},
	}

	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "HTTP port")
	rootCmd.Flags().StringVar(&host, "host", "0.0.0.0", "HTTP host")
	rootCmd.Flags().BoolVar(&workerMode, "worker", false, "Run in worker-only mode (no HTTP server)")
	rootCmd.Flags().BoolVar(&localMode, "local", false, "Run in local mode (API + worker in same process)")
	rootCmd.Flags().IntVar(&concurrency, "concurrency", 2, "Number of concurrent workers")
	rootCmd.Flags().DurationVar(&pollInterval, "poll-interval", 1*time.Second, "Worker poll interval")

	return rootCmd.ExecuteContext(context.Background())
}

type serverConfig struct {
	port         int
	host         string
	workerMode   bool
	localMode    bool
	concurrency  int
	pollInterval time.Duration
}

func runServer(ctx context.Context, cfg serverConfig) error {
	// Create production infrastructure (Redis + MongoDB required)
	clients, err := createInfra(ctx)
	if err != nil {
		return err
	}
	defer clients.redis.Close()
	defer clients.mongo.Close()

	// Create components from infrastructure
	q := clients.redis.Queue()
	sessionStore := clients.redis.Sessions()
	stateStore := clients.redis.OAuthStates()
	c := cache.NewCombinedCache(clients.redis.Cache(), clients.mongo.Store())
	defer c.Close()

	// Log configuration
	fmt.Printf("Redis: %s\n", clients.redis.Info())
	fmt.Printf("MongoDB: %s\n", clients.mongo.Info())

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down gracefully...")
		cancel()
	}()

	// Determine mode
	if cfg.workerMode {
		return runWorkerOnly(ctx, q, c, cfg)
	}

	if cfg.localMode {
		return runLocal(ctx, q, c, sessionStore, stateStore, cfg)
	}

	return runAPIOnly(ctx, q, c, sessionStore, stateStore, cfg)
}

// infraClients holds the production infrastructure clients.
type infraClients struct {
	redis *infra.Redis
	mongo *infra.Mongo
}

// createInfra creates production infrastructure clients from environment config.
func createInfra(ctx context.Context) (*infraClients, error) {
	cfg := infra.Load()

	if err := cfg.ValidateForAPI(); err != nil {
		return nil, err
	}

	redis, err := infra.NewRedis(ctx, cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}

	mongo, err := infra.NewMongo(ctx, cfg.Mongo)
	if err != nil {
		redis.Close()
		return nil, fmt.Errorf("connect to mongodb: %w", err)
	}

	return &infraClients{redis: redis, mongo: mongo}, nil
}

func runAPIOnly(ctx context.Context, q queue.Queue, c cache.Cache, sessionStore session.Store, stateStore session.StateStore, cfg serverConfig) error {
	fmt.Printf("Starting API server (API-only mode) on %s:%d\n", cfg.host, cfg.port)
	fmt.Println("Note: Workers must be started separately to process jobs")

	server := New(q, c,
		WithHost(cfg.host),
		WithPort(cfg.port),
		WithSessions(sessionStore),
		WithStates(stateStore),
		WithManifestPatterns(deps.SupportedManifests(languages.All)),
		WithLogger(common.DefaultLogger()),
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func runWorkerOnly(ctx context.Context, q queue.Queue, c cache.Cache, cfg serverConfig) error {
	fmt.Printf("Starting worker (worker-only mode) with concurrency=%d\n", cfg.concurrency)

	w := worker.New(q, c, worker.Config{
		Concurrency:  cfg.concurrency,
		PollInterval: cfg.pollInterval,
	})

	return w.Start(ctx)
}

func runLocal(ctx context.Context, q queue.Queue, c cache.Cache, sessionStore session.Store, stateStore session.StateStore, cfg serverConfig) error {
	fmt.Printf("Starting in LOCAL mode (API + Worker) on %s:%d\n", cfg.host, cfg.port)
	fmt.Printf("Worker concurrency: %d\n", cfg.concurrency)

	workerCtx, cancelWorker := context.WithCancel(ctx)
	defer cancelWorker()

	server := New(q, c,
		WithHost(cfg.host),
		WithPort(cfg.port),
		WithSessions(sessionStore),
		WithStates(stateStore),
		WithManifestPatterns(deps.SupportedManifests(languages.All)),
		WithLogger(common.DefaultLogger()),
	)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Start()
	}()

	w := worker.New(q, c, worker.Config{
		Concurrency:  cfg.concurrency,
		PollInterval: cfg.pollInterval,
	})

	workerErrCh := make(chan error, 1)
	go func() {
		workerErrCh <- w.Start(workerCtx)
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Shutting down API server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("Error shutting down server: %v\n", err)
		}
		return nil
	case err := <-serverErrCh:
		cancelWorker()
		return fmt.Errorf("server error: %w", err)
	case err := <-workerErrCh:
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		server.Shutdown(shutdownCtx)
		return fmt.Errorf("worker error: %w", err)
	}
}
