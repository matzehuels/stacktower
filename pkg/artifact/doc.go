// Package artifact provides caching for pipeline artifacts.
//
// # Backend Implementations
//
// Two backends are provided for different deployment scenarios:
//
// ## LocalBackend (CLI)
//
// For CLI usage. Uses pkg/storage for file storage and a local JSON
// index for TTL tracking. Artifacts are stored in ~/.stacktower/cache/artifacts/.
//
// Example:
//
//	backend, _ := artifact.NewLocalBackend(artifact.LocalBackendConfig{
//	    CacheDir: "~/.stacktower/cache",
//	})
//	defer backend.Close()
//	svc := pipeline.NewService(backend)
//
// ## ProdBackend (API)
//
// For API usage. Uses pkg/cache (Redis + MongoDB) for distributed
// caching with automatic TTL expiration handled by Redis.
//
// Example:
//
//	backend := artifact.NewProdBackend(cache)
//	svc := pipeline.NewService(backend)
//
// # Interface
//
// Both implement the Backend interface and are interchangeable
// from the pipeline.Service perspective.
package artifact
