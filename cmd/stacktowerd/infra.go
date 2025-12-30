package main

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/session"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
)

// infraClients holds the distributed infrastructure clients.
type infraClients struct {
	redis *infra.Redis
	mongo *infra.Mongo
}

// serverComponents holds the initialized server components derived from infrastructure.
type serverComponents struct {
	queue        queue.Queue
	backend      *storage.DistributedBackend
	sessionStore session.Store
	stateStore   session.StateStore
	githubOAuth  github.OAuthConfig
	frontendURL  string
}

// createInfra creates distributed infrastructure clients from environment config.
// In production mode, stricter validation is applied.
func createInfra(ctx context.Context, production bool) (*infraClients, error) {
	log := infra.DefaultLogger()
	cfg := infra.Load()

	// Validate configuration based on mode
	if production {
		if err := cfg.ValidateForProduction(); err != nil {
			return nil, fmt.Errorf("production validation failed: %w", err)
		}
	} else {
		if err := cfg.ValidateForAPI(); err != nil {
			return nil, fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	// Connect to Redis
	redis, err := infra.NewRedis(ctx, cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}

	// Configure session encryption if key is provided
	if cfg.Session.EncryptionKey != "" {
		if err := redis.WithSessionEncryption(cfg.Session.EncryptionKey); err != nil {
			redis.Close()
			return nil, fmt.Errorf("configure session encryption: %w", err)
		}
		log.Info("session encryption enabled")
	} else if production {
		// This shouldn't happen due to ValidateForProduction, but be defensive
		redis.Close()
		return nil, fmt.Errorf("session encryption required in production mode")
	} else {
		log.Warn("session encryption disabled (set STACKTOWER_SESSION_KEY to enable)")
	}

	// Connect to MongoDB
	mongo, err := infra.NewMongo(ctx, cfg.Mongo)
	if err != nil {
		redis.Close()
		return nil, fmt.Errorf("connect to mongodb: %w", err)
	}

	log.Info("infrastructure connected", "redis", redis.Info(), "mongo", mongo.Info())

	return &infraClients{redis: redis, mongo: mongo}, nil
}

// createComponents creates server components from infrastructure clients.
// In production mode, stricter URL validation is applied.
func createComponents(clients *infraClients, production bool) (*serverComponents, error) {
	// Load GitHub OAuth config once
	githubCfg := infra.LoadGitHubConfig()

	// Validate frontend URL
	if err := validateFrontendURL(githubCfg.FrontendURL, production); err != nil {
		return nil, fmt.Errorf("invalid frontend URL: %w", err)
	}

	return &serverComponents{
		queue: clients.redis.Queue(),
		backend: storage.NewDistributedBackend(
			clients.redis.Index(),
			clients.mongo.DocumentStore(),
			clients.redis.HTTPCache(),
			clients.redis.RateLimiter(),
		),
		sessionStore: clients.redis.Sessions(),
		stateStore:   clients.redis.OAuthStates(),
		githubOAuth: github.OAuthConfig{
			ClientID:     githubCfg.ClientID,
			ClientSecret: githubCfg.ClientSecret,
			RedirectURI:  githubCfg.RedirectURI,
		},
		frontendURL: githubCfg.FrontendURL,
	}, nil
}

// validateFrontendURL validates that the frontend URL is well-formed and safe.
// In production mode, HTTPS is required for non-localhost URLs.
func validateFrontendURL(frontendURL string, production bool) error {
	if frontendURL == "" {
		return nil // Empty is allowed (defaults will be used)
	}

	parsed, err := url.Parse(frontendURL)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https, got %q", parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("host is required")
	}

	// In production, require HTTPS for non-localhost
	isLocalhost := strings.HasPrefix(parsed.Host, "localhost") ||
		strings.HasPrefix(parsed.Host, "127.0.0.1")

	if production && !isLocalhost && parsed.Scheme != "https" {
		return fmt.Errorf("HTTPS required for non-localhost URLs in production mode")
	}

	return nil
}
