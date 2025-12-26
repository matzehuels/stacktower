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
	"github.com/matzehuels/stacktower/pkg/config"
	pkgerr "github.com/matzehuels/stacktower/pkg/errors"
	"github.com/matzehuels/stacktower/pkg/hash"
)

// Artifact types stored in the cache.
const (
	TypeGraph  = "graph"
	TypeLayout = "layout"
	TypeRender = "render"
)

// Default TTLs for different artifact types.
const (
	// GraphTTL is how long a parsed graph is cached (7 days).
	GraphTTL = config.GraphTTL

	// LayoutTTL is how long a computed layout is cached (30 days).
	LayoutTTL = config.LayoutTTL

	// RenderTTL is how long rendered artifacts are cached (90 days).
	RenderTTL = config.RenderTTL
)

// ErrCacheMiss is returned when an artifact is not found in cache.
var ErrCacheMiss = pkgerr.ErrCacheMiss

// =============================================================================
// Hash utilities (exported for use by pipeline)
// =============================================================================

// Hash computes a SHA256 hash of the given data and returns it as a hex string.
func Hash(data []byte) string {
	return hash.Bytes(data)
}

// HashJSON computes a hash of a JSON-serializable value.
func HashJSON(v interface{}) string {
	return hash.JSON(v)
}
