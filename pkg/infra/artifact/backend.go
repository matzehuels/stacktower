// Package artifact provides the caching abstraction used by pipeline.Service.
//
// This package defines the Backend interface that pipeline.Service uses to
// cache parsed graphs, computed layouts, and rendered artifacts (SVG, PNG, PDF).
// The interface abstracts the underlying storage, allowing the same pipeline
// code to work with different backends.
//
// IMPORTANT: This is different from pkg/infra/cache, which is the lower-level
// two-tier caching system. artifact.Backend wraps cache.Cache in production
// (via ProdBackend) but uses local files in CLI (via LocalBackend).
//
// # Implementations
//
//   - LocalBackend: File-based storage for CLI (uses local filesystem)
//   - ProdBackend: Production storage for API/Worker (wraps cache.Cache)
//   - NullBackend: No-op backend for testing or when caching is disabled
//
// # Usage
//
// CLI:
//
//	backend, _ := artifact.NewLocalBackend(artifact.LocalBackendConfig{
//	    CacheDir: "~/.stacktower/cache",
//	})
//	defer backend.Close()
//	svc := pipeline.NewService(backend)
//
// API/Worker:
//
//	cache := cache.NewCombinedCache(redis.Cache(), mongo.Store())
//	backend := artifact.NewProdBackend(cache)
//	svc := pipeline.NewService(backend)
//
// Testing:
//
//	svc := pipeline.NewService(nil)  // uses NullBackend

package artifact

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/core/dag"
)

// Backend is the unified caching interface for Stacktower.
//
// This interface serves two purposes:
//
//  1. Pipeline artifact caching (graphs, layouts, renders)
//  2. HTTP response caching for package registry APIs (PyPI, npm, etc.)
//
// By consolidating caching into a single interface, the entire application
// (CLI, API, Worker) can use one caching abstraction configured appropriately
// for each deployment context.
//
// # Pipeline Artifacts
//
// pipeline.Service uses this interface to cache:
//   - Parsed dependency graphs (GetGraph/PutGraph)
//   - Computed layouts (GetLayout/PutLayout)
//   - Rendered artifacts like SVG, PNG, PDF (GetRender/PutRender)
//
// The hash parameter is computed from the inputs that produce the artifact,
// ensuring content-addressable caching.
//
// # HTTP Response Caching
//
// pkg/integrations uses this interface to cache HTTP responses from
// package registries (GetHTTP/SetHTTP). The namespace parameter allows
// different registries to have isolated cache spaces (e.g., "pypi:", "npm:").
//
// # Implementations
//
//   - LocalBackend: File-based (CLI) - stores in ~/.stacktower/cache/
//   - ProdBackend: Redis+MongoDB (API/Worker) - wraps cache.Cache
//   - NullBackend: No-op (testing)
type Backend interface {
	// GetGraph retrieves a cached dependency graph by its input hash.
	// Returns (graph, true, nil) on cache hit.
	// Returns (nil, false, nil) on cache miss or expiration.
	// Returns (nil, false, err) on error.
	GetGraph(ctx context.Context, hash string) (*dag.DAG, bool, error)

	// PutGraph stores a dependency graph with its input hash and TTL.
	// The hash should be computed from the parse inputs (language, package, options).
	PutGraph(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration) error

	// GetLayout retrieves cached layout data by its input hash.
	// Layout data is JSON-encoded positioning information.
	GetLayout(ctx context.Context, hash string) ([]byte, bool, error)

	// PutLayout stores layout data with its input hash and TTL.
	// The hash should be computed from the layout inputs (graph hash, viz type, dimensions).
	PutLayout(ctx context.Context, hash string, data []byte, ttl time.Duration) error

	// GetRender retrieves a cached render artifact by hash and format.
	// format is typically "svg", "png", or "pdf".
	GetRender(ctx context.Context, hash, format string) ([]byte, bool, error)

	// PutRender stores a render artifact with its input hash, format, and TTL.
	// The hash should be computed from the render inputs (layout hash, style options).
	PutRender(ctx context.Context, hash, format string, data []byte, ttl time.Duration) error

	// GetHTTP retrieves a cached HTTP response by namespace and key.
	// namespace isolates different registries (e.g., "pypi:", "npm:").
	// key is typically a package name or URL path.
	// Returns (data, true, nil) on cache hit.
	// Returns (nil, false, nil) on cache miss or expiration.
	GetHTTP(ctx context.Context, namespace, key string) ([]byte, bool, error)

	// SetHTTP stores an HTTP response with namespace, key, and TTL.
	// namespace isolates different registries (e.g., "pypi:", "npm:").
	// key is typically a package name or URL path.
	SetHTTP(ctx context.Context, namespace, key string, data []byte, ttl time.Duration) error

	// DeleteHTTP removes a cached HTTP response.
	DeleteHTTP(ctx context.Context, namespace, key string) error

	// Close releases any resources held by the backend.
	Close() error
}

// NullBackend is a no-op backend that never caches anything.
// Useful for testing or when caching should be disabled.
type NullBackend struct{}

func (NullBackend) GetGraph(ctx context.Context, hash string) (*dag.DAG, bool, error) {
	return nil, false, nil
}

func (NullBackend) PutGraph(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration) error {
	return nil
}

func (NullBackend) GetLayout(ctx context.Context, hash string) ([]byte, bool, error) {
	return nil, false, nil
}

func (NullBackend) PutLayout(ctx context.Context, hash string, data []byte, ttl time.Duration) error {
	return nil
}

func (NullBackend) GetRender(ctx context.Context, hash, format string) ([]byte, bool, error) {
	return nil, false, nil
}

func (NullBackend) PutRender(ctx context.Context, hash, format string, data []byte, ttl time.Duration) error {
	return nil
}

func (NullBackend) GetHTTP(ctx context.Context, namespace, key string) ([]byte, bool, error) {
	return nil, false, nil
}

func (NullBackend) SetHTTP(ctx context.Context, namespace, key string, data []byte, ttl time.Duration) error {
	return nil
}

func (NullBackend) DeleteHTTP(ctx context.Context, namespace, key string) error {
	return nil
}

func (NullBackend) Close() error {
	return nil
}

// Ensure NullBackend implements Backend
var _ Backend = NullBackend{}
