package storage

import (
	"context"
	"fmt"
	"time"
)

// Note: ErrRateLimited and ErrQuotaExceeded are defined in types.go (re-exported from infra)

// Index is Tier 1 of the two-tier cache: fast TTL-based lookups.
//
// In production, this is backed by Redis. It answers the question:
// "Do we have this content cached?" If yes, it returns a CacheEntry
// with the DocumentStore ID. If no (or expired), the caller should
// compute the value and store it.
//
// Note: HTTP response caching is handled by Backend.GetHTTP/SetHTTP,
// not by Index. The Backend implementations manage HTTP caching internally.
//
// # Implementations
//
//   - Production: infra.Redis.Index() returns a Redis-backed Index
//   - Development: MemoryBackend implements Index (and DocumentStore)
type Index interface {
	// GetGraphEntry checks if a graph is cached.
	// key is typically a content hash or input hash.
	// Returns (entry, nil) on hit, (nil, nil) on miss, (nil, err) on error.
	GetGraphEntry(ctx context.Context, key string) (*CacheEntry, error)

	// SetGraphEntry stores a graph cache entry.
	// entry.DocumentID should be the ID returned by DocumentStore.StoreGraph().
	SetGraphEntry(ctx context.Context, key string, entry *CacheEntry) error

	// GetRenderEntry checks if a render is cached.
	GetRenderEntry(ctx context.Context, key string) (*CacheEntry, error)

	// SetRenderEntry stores a render cache entry.
	SetRenderEntry(ctx context.Context, key string, entry *CacheEntry) error

	// Ping checks if the index backend is reachable.
	Ping(ctx context.Context) error

	Close() error
}

// HTTPCache provides HTTP response caching, typically backed by Redis.
// This is used internally by Backend implementations for registry API caching.
type HTTPCache interface {
	// GetHTTP retrieves a cached HTTP response.
	// Returns (data, true, nil) on hit, (nil, false, nil) on miss.
	GetHTTP(ctx context.Context, key string) ([]byte, bool, error)

	// SetHTTP stores an HTTP response with TTL.
	SetHTTP(ctx context.Context, key string, data []byte, ttl time.Duration) error

	// DeleteHTTP removes a cached HTTP response.
	DeleteHTTP(ctx context.Context, key string) error
}

// RateLimiter provides rate limiting and quota management for users.
// Uses a sliding window approach for rate limiting.
//
// # Implementations
//
//   - Production: infra.Redis.RateLimiter() returns a Redis-backed RateLimiter
//   - Development: MemoryBackend implements RateLimiter
type RateLimiter interface {
	// CheckRateLimit checks if a user can perform an operation.
	// Returns nil if allowed, ErrRateLimited if over limit.
	// Does NOT increment the counter - call IncrementRateLimit after successful operation.
	CheckRateLimit(ctx context.Context, userID string, opType OperationType, quota QuotaConfig) error

	// IncrementRateLimit increments the operation counter for a user.
	// Should be called after a successful operation.
	IncrementRateLimit(ctx context.Context, userID string, opType OperationType) error

	// GetRateLimitStatus returns current rate limit status for a user.
	GetRateLimitStatus(ctx context.Context, userID string, quota QuotaConfig) (*RateLimitStatus, error)

	// CheckStorageQuota checks if user can store more data.
	// Returns nil if allowed, ErrQuotaExceeded if over limit.
	CheckStorageQuota(ctx context.Context, userID string, bytesToAdd int64, quota QuotaConfig) error

	// UpdateStorageUsage updates the storage usage counter for a user.
	// Can be positive (add) or negative (delete).
	UpdateStorageUsage(ctx context.Context, userID string, byteDelta int64) error
}

// RateLimitStatus represents current rate limit usage for a user.
type RateLimitStatus struct {
	ParsesUsed        int   `json:"parses_used"`
	ParsesLimit       int   `json:"parses_limit"`
	LayoutsUsed       int   `json:"layouts_used"`
	LayoutsLimit      int   `json:"layouts_limit"`
	RendersUsed       int   `json:"renders_used"`
	RendersLimit      int   `json:"renders_limit"`
	StorageBytesUsed  int64 `json:"storage_bytes_used"`
	StorageBytesLimit int64 `json:"storage_bytes_limit"`
	WindowResetAt     int64 `json:"window_reset_at"` // Unix timestamp
}

// =============================================================================
// Key Generation Utilities
// =============================================================================

// GraphCacheKey generates a cache key for a dependency graph.
func GraphCacheKey(scope Scope, userID, language, packageOrManifest string, opts GraphOptions) string {
	optsHash := OptionsHash(opts)
	if scope == ScopeGlobal {
		return fmt.Sprintf("graph:global:%s:%s:%s", language, packageOrManifest, optsHash)
	}
	return fmt.Sprintf("graph:user:%s:%s:%s:%s", userID, language, packageOrManifest, optsHash)
}

// RenderCacheKey generates a cache key for a user's render.
func RenderCacheKey(userID, graphHash string, layoutOpts LayoutOptions) string {
	optsHash := OptionsHash(layoutOpts)
	return fmt.Sprintf("render:user:%s:%s:%s", userID, graphHash[:16], optsHash)
}

// OptionsHash computes a short hash of options for cache keys.
func OptionsHash(opts interface{}) string {
	return HashJSON(opts)[:16] // hex hash, 8 bytes = 16 chars
}
