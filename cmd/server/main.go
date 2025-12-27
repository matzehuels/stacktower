package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/matzehuels/stacktower/pkg/buildinfo"
	"github.com/matzehuels/stacktower/pkg/infra"
)

func main() {
	if err := execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// serverConfig holds the CLI configuration for the server.
type serverConfig struct {
	port         int
	host         string
	workerMode   bool
	localMode    bool
	concurrency  int
	pollInterval time.Duration
}

// execute runs the API server CLI.
func execute() error {
	var cfg serverConfig

	rootCmd := &cobra.Command{
		Use:     "stacktowerd",
		Short:   "Stacktower daemon (API + Worker)",
		Version: buildinfo.Version,
		Long: `Stacktower daemon provides the HTTP API and background worker
for the Stacktower dependency visualization platform.

Modes:
  - Default: Run HTTP API server only
  - --worker: Run background worker only (job consumer)
  - --local:  Run both API and worker in a single process (development)

Environment variables (required):
  STACKTOWER_REDIS_ADDR=host:port     Redis server address
  STACKTOWER_MONGODB_URI=mongodb://   MongoDB connection string

Environment variables (optional):
  STACKTOWER_REDIS_PASSWORD=secret    Redis password
  STACKTOWER_MONGODB_DATABASE=name    MongoDB database (default: stacktower)
  STACKTOWER_SESSION_KEY=base64key    Session encryption key (32 bytes, base64-encoded)
                                      Generate with: openssl rand -base64 32`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cmd.Context(), cfg)
		},
	}

	rootCmd.SetVersionTemplate(buildinfo.Template())

	rootCmd.Flags().IntVarP(&cfg.port, "port", "p", 8080, "HTTP port")
	rootCmd.Flags().StringVar(&cfg.host, "host", "0.0.0.0", "HTTP host")
	rootCmd.Flags().BoolVar(&cfg.workerMode, "worker", false, "Run in worker-only mode (no HTTP server)")
	rootCmd.Flags().BoolVar(&cfg.localMode, "local", false, "Run in local mode (API + worker in same process)")
	rootCmd.Flags().IntVar(&cfg.concurrency, "concurrency", 2, "Number of concurrent workers")
	rootCmd.Flags().DurationVar(&cfg.pollInterval, "poll-interval", 1*time.Second, "Worker poll interval")

	return rootCmd.ExecuteContext(context.Background())
}

// runServer initializes infrastructure and starts the appropriate mode.
func runServer(ctx context.Context, cfg serverConfig) error {
	log := infra.DefaultLogger()

	// Create production infrastructure (Redis + MongoDB required).
	clients, err := createInfra(ctx)
	if err != nil {
		return err
	}
	defer clients.redis.Close()
	defer clients.mongo.Close()

	// Create server components from infrastructure.
	comp := createComponents(clients)
	defer comp.backend.Close()

	// Set up graceful shutdown.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Info("received shutdown signal")
		cancel()
	}()

	// Determine mode and run.
	switch {
	case cfg.workerMode:
		return runWorkerOnly(ctx, comp, cfg)
	case cfg.localMode:
		return runLocal(ctx, comp, cfg)
	default:
		return runAPIOnly(ctx, comp, cfg)
	}
}
