// Package artifact provides unified artifact caching and storage for the pipeline.
//
// # Architecture
//
// This package provides a Backend interface that abstracts different storage
// implementations:
//
//   - FileBackend: File-based storage for CLI (uses local filesystem)
//   - ProdBackend: Production storage for API (uses Redis for lookup + MongoDB for storage)
//   - NullBackend: No-op backend for testing or when caching is disabled
//
// The pipeline.Service uses a Backend to cache pipeline artifacts (graphs,
// layouts, renders) with automatic TTL-based expiration.
//
// # Usage
//
// For CLI (local caching):
//
//	backend, _ := artifact.NewLocalBackend(artifact.LocalBackendConfig{
//	    CacheDir: "~/.stacktower/cache",
//	})
//	defer backend.Close()
//	svc := pipeline.NewService(backend)
//
// For API (Redis + MongoDB):
//
//	backend := artifact.NewProdBackend(cache)
//	svc := pipeline.NewService(backend)
//
// For testing (no caching):
//
//	svc := pipeline.NewService(nil) // uses NullBackend
package artifact

import (
	"github.com/matzehuels/stacktower/pkg/infra/common"
)

// Artifact types stored in the cache.
const (
	TypeGraph  = "graph"
	TypeLayout = "layout"
	TypeRender = "render"
	TypeHTTP   = "http" // HTTP response cache for registry APIs
)

// Default TTLs for different artifact types.
const (
	// GraphTTL is how long a parsed graph is cached (7 days).
	GraphTTL = common.GraphTTL

	// LayoutTTL is how long a computed layout is cached (30 days).
	LayoutTTL = common.LayoutTTL

	// RenderTTL is how long rendered artifacts are cached (90 days).
	RenderTTL = common.RenderTTL

	// HTTPTTL is how long HTTP responses are cached (24 hours).
	HTTPTTL = common.HTTPCacheTTL
)

// ErrCacheMiss is returned when an artifact is not found in cache.
var ErrCacheMiss = common.ErrCacheMiss

// =============================================================================
// Hash utilities (exported for use by pipeline)
// =============================================================================

// Hash computes a SHA256 hash of the given data and returns it as a hex string.
func Hash(data []byte) string {
	return common.HashBytes(data)
}

// HashJSON computes a hash of a JSON-serializable value.
func HashJSON(v interface{}) string {
	return common.HashJSON(v)
}

// HashKey computes a hash of a cache key string.
// Used for HTTP cache keys to normalize key length.
func HashKey(key string) string {
	return common.HashBytes([]byte(key))
}
