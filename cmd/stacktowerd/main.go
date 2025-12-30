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

// DeployMode represents the deployment environment.
type DeployMode string

const (
	// ModeStandalone runs everything in-memory with no external dependencies.
	// API + Worker run together in a single process.
	//
	// Security: None (for local dev only)
	//   - Session: in-memory, unencrypted
	//   - CORS: allow all origins
	//   - Auth: optional (--no-auth bypasses entirely)
	//   - Cookies: not secure (HTTP)
	ModeStandalone DeployMode = "standalone"

	// ModeCompose uses Redis + MongoDB with relaxed security.
	// Designed for docker-compose local development with the full distributed stack.
	// Supports separate API and Worker processes via --worker and --all flags.
	//
	// Security: Relaxed (for local docker-compose)
	//   - Session: Redis-backed, encryption optional (warns if disabled)
	//   - CORS: allow all origins
	//   - Auth: required
	//   - Cookies: not secure (HTTP)
	ModeCompose DeployMode = "compose"

	// ModeProduction uses Redis + MongoDB with strict security requirements.
	// For Kubernetes or production deployments with proper TLS termination.
	//
	// Security: Strict
	//   - Session: Redis-backed, encryption REQUIRED
	//   - CORS: frontend URL only
	//   - Auth: required
	//   - Cookies: secure (HTTPS required for non-localhost)
	ModeProduction DeployMode = "production"
)

// Role represents what the daemon runs.
type Role string

const (
	RoleAPI    Role = "api"    // HTTP server only
	RoleWorker Role = "worker" // Background worker only
	RoleAll    Role = "all"    // Both API and Worker
)

// serverConfig holds the CLI configuration for the server.
type serverConfig struct {
	mode              DeployMode
	role              Role
	port              int
	host              string
	concurrency       int
	pollInterval      time.Duration
	trustProxyHeaders bool
	noAuth            bool // Bypass authentication (standalone mode only)
}

// execute runs the API server CLI.
func execute() error {
	var cfg serverConfig
	var modeStr string
	var workerFlag, allFlag bool

	rootCmd := &cobra.Command{
		Use:     "stacktowerd [standalone]",
		Short:   "Stacktower daemon (API + Worker)",
		Version: buildinfo.Version,
		Long: `Stacktower daemon provides the HTTP API and background worker
for the Stacktower dependency visualization platform.

DEPLOYMENT MODES:

  standalone    All-in-one mode with no external dependencies.
                Uses in-memory caching, storage, and queues.
                API and Worker run together in one process.
                Perfect for local development without Docker.
                Use --no-auth to bypass authentication entirely.

  compose       Distributed mode with Redis + MongoDB.
                Relaxed security (HTTP allowed, session key optional).
                Run API, Worker, or both via --worker / --all flags.
                Designed for docker-compose local development.

  production    Same as compose but with strict security.
                HTTPS required, session encryption required.
                Auto-detected when STACKTOWER_MODE=production.

SECURITY SUMMARY:

  Mode        Sessions    Encryption  CORS      Cookies   Auth
  ----------  ----------  ----------  --------  --------  --------
  standalone  in-memory   none        *         insecure  optional
  compose     redis       optional    *         insecure  required
  production  redis       REQUIRED    frontend  secure    required

PROCESS ROLES (for compose/production modes):

  (default)     Run HTTP API server only
  --worker      Run background worker only (job consumer)
  --all         Run both API and Worker in a single process

EXAMPLES:

  # Local development with no Docker (simplest)
  stacktowerd standalone
  stacktowerd standalone --no-auth    # Skip auth for quick testing

  # Docker-compose development
  stacktowerd                # API only (workers separate)
  stacktowerd --worker       # Worker only
  stacktowerd --all          # Both in same process

  # Production (Kubernetes, etc.)
  STACKTOWER_MODE=production stacktowerd
  STACKTOWER_MODE=production stacktowerd --worker

ENVIRONMENT VARIABLES:

  Required for compose/production modes:
    STACKTOWER_REDIS_ADDR=host:port     Redis server address
    STACKTOWER_MONGODB_URI=mongodb://   MongoDB connection string

  Optional:
    STACKTOWER_MODE=production          Force production mode
    STACKTOWER_SESSION_KEY=base64key    Session encryption (required in production)
    GITHUB_CLIENT_ID                    GitHub OAuth client ID
    GITHUB_CLIENT_SECRET                GitHub OAuth client secret
    FRONTEND_URL                        Frontend URL for OAuth redirects`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine mode
			cfg.mode = determineMode(modeStr, args)

			// Determine role
			cfg.role = determineRole(cfg.mode, workerFlag, allFlag)

			// Validate --no-auth usage
			if cfg.noAuth && cfg.mode != ModeStandalone {
				return fmt.Errorf("--no-auth is only allowed in standalone mode")
			}

			return runDaemon(cmd.Context(), cfg)
		},
	}

	rootCmd.SetVersionTemplate(buildinfo.Template())

	// Mode flag (can also be positional arg for "standalone")
	rootCmd.Flags().StringVar(&modeStr, "mode", "", "Deployment mode: standalone, compose, production")

	// Role flags
	rootCmd.Flags().BoolVar(&workerFlag, "worker", false, "Run as worker only (no API server)")
	rootCmd.Flags().BoolVar(&allFlag, "all", false, "Run both API and Worker in same process")

	// Common flags
	rootCmd.Flags().IntVarP(&cfg.port, "port", "p", 8080, "HTTP port")
	rootCmd.Flags().StringVar(&cfg.host, "host", "0.0.0.0", "HTTP host")
	rootCmd.Flags().IntVar(&cfg.concurrency, "concurrency", 2, "Number of concurrent workers")
	rootCmd.Flags().DurationVar(&cfg.pollInterval, "poll-interval", 1*time.Second, "Worker poll interval")
	rootCmd.Flags().BoolVar(&cfg.trustProxyHeaders, "trust-proxy", false, "Trust X-Forwarded-For headers (only enable behind trusted proxy)")

	// Standalone-only flags
	rootCmd.Flags().BoolVar(&cfg.noAuth, "no-auth", false, "Bypass authentication (standalone mode only)")

	return rootCmd.ExecuteContext(context.Background())
}

// determineMode resolves the deployment mode from flags, args, and environment.
func determineMode(modeStr string, args []string) DeployMode {
	// Check positional arg first (e.g., "stacktowerd standalone")
	if len(args) > 0 && args[0] == "standalone" {
		return ModeStandalone
	}

	// Check --mode flag
	if modeStr != "" {
		switch DeployMode(modeStr) {
		case ModeStandalone:
			return ModeStandalone
		case ModeCompose, "development": // Accept "development" as alias for backwards compat
			return ModeCompose
		case ModeProduction:
			return ModeProduction
		}
	}

	// Check environment variable
	if envMode := os.Getenv("STACKTOWER_MODE"); envMode != "" {
		switch DeployMode(envMode) {
		case ModeStandalone:
			return ModeStandalone
		case ModeCompose, "development": // Accept "development" as alias
			return ModeCompose
		case ModeProduction:
			return ModeProduction
		}
	}

	// Auto-detect: if Redis/MongoDB aren't configured, assume standalone
	redisCfg := os.Getenv("STACKTOWER_REDIS_ADDR")
	mongoCfg := os.Getenv("STACKTOWER_MONGODB_URI")
	if redisCfg == "" && mongoCfg == "" {
		return ModeStandalone
	}

	// Default to compose (has Redis/MongoDB but not explicitly production)
	return ModeCompose
}

// determineRole resolves which components to run.
func determineRole(mode DeployMode, workerFlag, allFlag bool) Role {
	// Standalone always runs everything
	if mode == ModeStandalone {
		return RoleAll
	}

	// Check flags
	if workerFlag {
		return RoleWorker
	}
	if allFlag {
		return RoleAll
	}

	// Default: API only
	return RoleAPI
}

// runDaemon initializes infrastructure and starts the appropriate components.
func runDaemon(ctx context.Context, cfg serverConfig) error {
	log := infra.DefaultLogger()

	// Log the mode we're running in
	logFields := []any{
		"mode", cfg.mode,
		"role", cfg.role,
		"host", cfg.host,
		"port", cfg.port,
	}
	if cfg.noAuth {
		logFields = append(logFields, "auth", "disabled")
	}
	log.Info("starting stacktowerd", logFields...)

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Info("received shutdown signal")
		cancel()
	}()

	// Branch based on mode
	switch cfg.mode {
	case ModeStandalone:
		return runStandalone(ctx, cfg)
	case ModeCompose:
		return runDistributed(ctx, cfg, false)
	case ModeProduction:
		return runDistributed(ctx, cfg, true)
	default:
		return fmt.Errorf("unknown mode: %s", cfg.mode)
	}
}
