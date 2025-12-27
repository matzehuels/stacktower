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
//   - storage: Unified storage backends for caching and persistence
//     (FileBackend for CLI, DistributedBackend for API/Worker, MemoryBackend for testing)
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
//   - Index() → storage.Index (fast TTL-based lookups, Tier 1)
//   - HTTPCache() → storage.HTTPCache (HTTP response caching)
//   - Raw() → *redis.Client (for advanced operations)
//
// # MongoDB Client (mongo.go)
//
// The Mongo struct wraps a single mongo.Client and provides:
//
//   - DocumentStore() → storage.DocumentStore (graph and render storage, Tier 2)
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
//	// Create distributed backend from Redis + MongoDB
//	backend := storage.NewDistributedBackend(
//	    redis.Index(),
//	    mongo.DocumentStore(),
//	    redis.HTTPCache(),
//	)
//
//	// Use in API server
//	server := api.New(
//	    redis.Queue(),
//	    backend,
//	    api.WithSessions(redis.Sessions()),
//	    api.WithStates(redis.OAuthStates()),
//	)
package infra
