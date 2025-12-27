package main

import (
	"context"
	"fmt"

	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/session"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/integrations/github"
)

// infraClients holds the production infrastructure clients.
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

// createInfra creates production infrastructure clients from environment config.
func createInfra(ctx context.Context) (*infraClients, error) {
	log := infra.DefaultLogger()
	cfg := infra.Load()

	if err := cfg.ValidateForAPI(); err != nil {
		return nil, err
	}

	redis, err := infra.NewRedis(ctx, cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}

	// Configure session encryption if key is provided.
	if cfg.Session.EncryptionKey != "" {
		if err := redis.WithSessionEncryption(cfg.Session.EncryptionKey); err != nil {
			redis.Close()
			return nil, fmt.Errorf("configure session encryption: %w", err)
		}
		log.Info("session encryption enabled")
	} else {
		log.Warn("session encryption disabled (set STACKTOWER_SESSION_KEY to enable)")
	}

	mongo, err := infra.NewMongo(ctx, cfg.Mongo)
	if err != nil {
		redis.Close()
		return nil, fmt.Errorf("connect to mongodb: %w", err)
	}

	log.Info("infrastructure connected", "redis", redis.Info(), "mongo", mongo.Info())

	return &infraClients{redis: redis, mongo: mongo}, nil
}

// createComponents creates server components from infrastructure clients.
func createComponents(clients *infraClients) *serverComponents {
	// Load GitHub OAuth config once.
	githubCfg := infra.LoadGitHubConfig()

	return &serverComponents{
		queue: clients.redis.Queue(),
		backend: storage.NewDistributedBackend(
			clients.redis.Index(),
			clients.mongo.DocumentStore(),
			clients.redis.HTTPCache(),
			clients.redis.RateLimiter(),
			clients.mongo.OperationStore(),
		),
		sessionStore: clients.redis.Sessions(),
		stateStore:   clients.redis.OAuthStates(),
		githubOAuth: github.OAuthConfig{
			ClientID:     githubCfg.ClientID,
			ClientSecret: githubCfg.ClientSecret,
			RedirectURI:  githubCfg.RedirectURI,
		},
		frontendURL: githubCfg.FrontendURL,
	}
}
