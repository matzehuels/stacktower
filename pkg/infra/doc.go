// Package infra provides production infrastructure clients.
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
// # Redis Client
//
// The Redis struct wraps a single redis.Client and provides:
//
//   - Queue() → queue.Queue (job queue via Redis Streams)
//   - Sessions() → session.Store (session storage)
//   - OAuthStates() → session.StateStore (OAuth state tokens)
//   - Cache() → cache.LookupCache (fast TTL-based lookups)
//   - Raw() → *redis.Client (for advanced operations)
//
// # MongoDB Client
//
// The Mongo struct wraps a single mongo.Client and provides:
//
//   - Store() → cache.Store (graph and render storage)
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
//	server := api.New(
//	    redis.Queue(),
//	    cache.NewCombinedCache(redis.Cache(), mongo.Store()),
//	    api.WithSessions(redis.Sessions()),
//	    api.WithStates(redis.OAuthStates()),
//	    api.WithStorage(storage.NewGridFSStorage(mongo.Database())),
//	)
package infra
