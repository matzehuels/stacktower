package storage

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/core/dag"
)

// Backend is the primary storage interface for Stacktower.
//
// This interface is used by:
//   - pipeline.Service: For caching parsed graphs, computed layouts, and rendered artifacts
//   - pkg/integrations: For caching HTTP responses from package registries
//
// All methods use content-addressable hashing. The same inputs produce the same
// hash, enabling efficient cache lookups without user-specific keys.
//
// # Scoping
//
// Public packages (from registries) use ScopeGlobal - cached content is shared across all users.
// Private manifests use ScopeUser - cached content is isolated to the user who created it.
// HTTP responses are always shared (registry data is public).
//
// # Implementations
//
//   - FileBackend: File-based storage for CLI (uses local filesystem)
//   - DistributedBackend: Production storage for API/Worker (Redis + MongoDB)
//   - MemoryBackend: In-memory storage for testing
//   - NullBackend: No-op backend that never caches
//
// # Thread Safety
//
// All implementations must be safe for concurrent use by multiple goroutines.
type Backend interface {
	// GetGraph retrieves a cached dependency graph by its input hash.
	// Returns (graph, true, nil) on cache hit.
	// Returns (nil, false, nil) on cache miss or expiration.
	// Returns (nil, false, err) on error.
	GetGraph(ctx context.Context, hash string) (*dag.DAG, bool, error)

	// PutGraph stores a dependency graph with its input hash and TTL.
	// The hash should be computed from the parse inputs (language, package, options).
	PutGraph(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration) error

	// GetGraphScoped retrieves a cached dependency graph with scope enforcement.
	// For ScopeUser graphs, verifies the requesting user matches the owner.
	// userID should be the authenticated user making the request.
	// Returns (nil, false, nil) if graph exists but user doesn't have access.
	GetGraphScoped(ctx context.Context, hash string, userID string) (*dag.DAG, bool, error)

	// PutGraphScoped stores a dependency graph with explicit scope and user association.
	// For ScopeGlobal: userID is ignored, content is shared across all users.
	// For ScopeUser: userID is required, content is private to that user.
	PutGraphScoped(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration, scope Scope, userID string, meta GraphMeta) error

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
	// HTTP responses are always shared (no user scoping).
	GetHTTP(ctx context.Context, namespace, key string) ([]byte, bool, error)

	// SetHTTP stores an HTTP response with namespace, key, and TTL.
	// namespace isolates different registries (e.g., "pypi:", "npm:").
	// key is typically a package name or URL path.
	// HTTP responses are always shared (no user scoping).
	SetHTTP(ctx context.Context, namespace, key string, data []byte, ttl time.Duration) error

	// DeleteHTTP removes a cached HTTP response.
	DeleteHTTP(ctx context.Context, namespace, key string) error

	// Close releases any resources held by the backend.
	Close() error
}

// GraphMeta contains metadata for storing a graph (used by PutGraphScoped).
type GraphMeta struct {
	Language string
	Package  string
	Repo     string
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

func (NullBackend) GetGraphScoped(ctx context.Context, hash string, userID string) (*dag.DAG, bool, error) {
	return nil, false, nil
}

func (NullBackend) PutGraphScoped(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration, scope Scope, userID string, meta GraphMeta) error {
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
