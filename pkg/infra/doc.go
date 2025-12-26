// Package infra provides production infrastructure components for Stacktower.
//
// See pkg/infra/README.md for detailed architecture documentation.
//
// # Design Philosophy
//
// This package consolidates database connections into unified clients
// that implement multiple interfaces from other packages. This:
//
//   - Reduces connection overhead (one Redis client, multiple uses)
//   - Centralizes configuration (all config from environment)
//   - Simplifies dependency injection (pass one client, get many interfaces)
//
// # Package Structure
//
// The infra package is organized into sub-packages:
//
//   - cache: Two-tier caching for graphs, renders, and HTTP responses (Redis+MongoDB)
//   - artifact: Unified caching backend for pipeline artifacts AND HTTP responses
//     (LocalBackend for CLI, ProdBackend for API/Worker)
//   - session: User session management
//   - queue: Job queue abstraction
//   - common: Shared constants, errors, hash utilities, retry logic
//
// # Redis Client (redis.go)
//
// The Redis struct wraps a single redis.Client and provides:
//
//   - Queue() → queue.Queue (job queue via Redis Streams)
//   - Sessions() → session.Store (session storage)
//   - OAuthStates() → session.StateStore (OAuth state tokens)
//   - Cache() → cache.LookupCache (fast TTL-based lookups, Tier 1)
//   - Raw() → *redis.Client (for advanced operations)
//
// # MongoDB Client (mongo.go)
//
// The Mongo struct wraps a single mongo.Client and provides:
//
//   - Store() → cache.Store (graph and render storage, Tier 2)
//   - Database() → *mongo.Database (for GridFS and custom queries)
//
// # Usage Example
//
//	cfg := infra.Load()
//
//	redis, err := infra.NewRedis(ctx, cfg.Redis)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer redis.Close()
//
//	mongo, err := infra.NewMongo(ctx, cfg.Mongo)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer mongo.Close()
//
//	// Use unified clients
//	cache := cache.NewCombinedCache(redis.Cache(), mongo.Store())
//	server := api.New(
//	    redis.Queue(),
//	    cache,
//	    api.WithSessions(redis.Sessions()),
//	    api.WithStates(redis.OAuthStates()),
//	)
package infra
