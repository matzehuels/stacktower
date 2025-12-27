package storage

import (
	"context"
	"errors"
	"time"
)

// =============================================================================
// Sentinel Errors (canonical definitions)
// =============================================================================

var (
	// ErrNotFound is returned when a requested item does not exist.
	ErrNotFound = errors.New("not found")

	// ErrExpired is returned when a cache entry has exceeded its TTL.
	ErrExpired = errors.New("expired")

	// ErrCacheMiss is returned when an item is not found in cache.
	ErrCacheMiss = errors.New("cache miss")

	// ErrNetwork is returned for HTTP failures (timeouts, connection errors, 5xx responses).
	ErrNetwork = errors.New("network error")

	// ErrInvalid is returned for invalid input or state.
	ErrInvalid = errors.New("invalid")

	// ErrAccessDenied is returned when a user tries to access a resource they don't own.
	ErrAccessDenied = errors.New("access denied")

	// ErrRateLimited is returned when a user exceeds their rate limit.
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrQuotaExceeded is returned when a user exceeds their storage quota.
	ErrQuotaExceeded = errors.New("storage quota exceeded")
)

// =============================================================================
// Retry Utilities
// =============================================================================

// RetryableError wraps an error to indicate it should trigger a retry.
type RetryableError struct{ Err error }

// Retryable wraps an error as a [RetryableError].
func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}

// Error returns the error message of the wrapped error.
func (e *RetryableError) Error() string { return e.Err.Error() }

// Unwrap returns the wrapped error.
func (e *RetryableError) Unwrap() error { return e.Err }

// RetryWithBackoff retries fn up to 3 times with exponential backoff.
// Only errors wrapped with [Retryable] will trigger retries.
func RetryWithBackoff(ctx context.Context, fn func() error) error {
	const attempts = 3
	delay := time.Second
	var lastErr error

	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else if lastErr = err; !IsRetryable(err) {
			return err
		}

		if i < attempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				delay *= 2
			}
		}
	}
	return lastErr
}

// IsRetryable checks if an error is wrapped with [RetryableError].
func IsRetryable(err error) bool {
	var re *RetryableError
	return errors.As(err, &re)
}

// =============================================================================
// TTL Configuration (canonical definitions)
// =============================================================================

const (
	// GraphTTL is how long resolved dependency graphs are cached (7 days).
	GraphTTL = 7 * 24 * time.Hour

	// LayoutTTL is how long computed layouts are cached (30 days).
	LayoutTTL = 30 * 24 * time.Hour

	// RenderTTL is how long rendered artifacts (SVG/PNG/PDF) are cached (90 days).
	RenderTTL = 90 * 24 * time.Hour

	// HTTPTTL is the default TTL for HTTP response caching (24 hours).
	HTTPTTL = 24 * time.Hour

	// HTTPCacheTTL is an alias for HTTPTTL for consistency.
	HTTPCacheTTL = HTTPTTL

	// SessionTTL is the default session duration.
	SessionTTL = 24 * time.Hour

	// OAuthStateTTL is the default OAuth state token duration.
	OAuthStateTTL = 10 * time.Minute
)

// =============================================================================
// Scope (for user-scoped vs global caching)
// =============================================================================

// Scope indicates whether data is globally shared or user-private.
type Scope string

const (
	// ScopeGlobal means data is shared across all users (public packages).
	ScopeGlobal Scope = "global"

	// ScopeUser means data is private to a specific user (private repos).
	ScopeUser Scope = "user"
)

// =============================================================================
// Cache Entry (Tier 1 - stored in Index/Redis)
// =============================================================================

// CacheEntry is stored in the Index (Redis) - a pointer to DocumentStore (MongoDB).
type CacheEntry struct {
	DocumentID string    `json:"document_id"` // MongoDB document ID
	ExpiresAt  time.Time `json:"expires_at"`
}

// IsExpired returns true if the cache entry has exceeded its TTL.
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// =============================================================================
// Graph Document (Tier 2 - stored in DocumentStore/MongoDB)
// =============================================================================

// GraphOptions defines parameters that affect graph resolution.
// Different options produce different graphs, so they're part of the cache key.
type GraphOptions struct {
	MaxDepth  int  `json:"max_depth"`
	MaxNodes  int  `json:"max_nodes"`
	Normalize bool `json:"normalize"`
}

// Graph represents a stored dependency graph in the DocumentStore.
type Graph struct {
	ID          string       `json:"id" bson:"_id,omitempty"`
	Scope       Scope        `json:"scope" bson:"scope"`
	UserID      string       `json:"user_id,omitempty" bson:"user_id,omitempty"` // Only for ScopeUser
	Language    string       `json:"language" bson:"language"`
	Package     string       `json:"package,omitempty" bson:"package,omitempty"`
	Repo        string       `json:"repo,omitempty" bson:"repo,omitempty"` // For manifest sources
	Options     GraphOptions `json:"options" bson:"options"`
	NodeCount   int          `json:"node_count" bson:"node_count"`
	EdgeCount   int          `json:"edge_count" bson:"edge_count"`
	ContentHash string       `json:"content_hash" bson:"content_hash"` // SHA256 of graph data
	Data        []byte       `json:"data" bson:"data"`                 // JSON-encoded DAG
	CreatedAt   time.Time    `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" bson:"updated_at"`
}

// =============================================================================
// Render Document (Tier 2 - stored in DocumentStore/MongoDB)
// =============================================================================

// LayoutOptions defines parameters for layout computation.
type LayoutOptions struct {
	VizType   string  `json:"viz_type"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Ordering  string  `json:"ordering,omitempty"`
	Merge     bool    `json:"merge,omitempty"`
	Randomize bool    `json:"randomize,omitempty"`
	Seed      uint64  `json:"seed,omitempty"`
}

// RenderOptions defines parameters for rendering output.
type RenderOptions struct {
	Formats   []string `json:"formats"`
	Style     string   `json:"style,omitempty"`
	ShowEdges bool     `json:"show_edges,omitempty"`
	Nebraska  bool     `json:"nebraska,omitempty"`
	Popups    bool     `json:"popups,omitempty"`
}

// RenderSource identifies what was rendered.
type RenderSource struct {
	Type             string `json:"type" bson:"type"` // "package" or "manifest"
	Language         string `json:"language" bson:"language"`
	Package          string `json:"package,omitempty" bson:"package,omitempty"`
	Repo             string `json:"repo,omitempty" bson:"repo,omitempty"`
	ManifestFilename string `json:"manifest_filename,omitempty" bson:"manifest_filename,omitempty"`
	ManifestHash     string `json:"manifest_hash,omitempty" bson:"manifest_hash,omitempty"`
}

// RenderArtifacts holds references to rendered output files (GridFS IDs).
type RenderArtifacts struct {
	SVG string `json:"svg,omitempty" bson:"svg,omitempty"`
	PNG string `json:"png,omitempty" bson:"png,omitempty"`
	PDF string `json:"pdf,omitempty" bson:"pdf,omitempty"`
}

// Render represents a user's visualization stored in the DocumentStore.
type Render struct {
	ID            string          `json:"id" bson:"_id,omitempty"`
	UserID        string          `json:"user_id" bson:"user_id"`
	Source        RenderSource    `json:"source" bson:"source"`
	GraphID       string          `json:"graph_id" bson:"graph_id"`
	GraphHash     string          `json:"graph_hash" bson:"graph_hash"`
	LayoutOptions LayoutOptions   `json:"layout_options" bson:"layout_options"`
	RenderOptions RenderOptions   `json:"render_options" bson:"render_options"`
	Layout        []byte          `json:"layout" bson:"layout"` // JSON-encoded layout
	Artifacts     RenderArtifacts `json:"artifacts" bson:"artifacts"`
	NodeCount     int             `json:"node_count" bson:"node_count"`
	EdgeCount     int             `json:"edge_count" bson:"edge_count"`
	CreatedAt     time.Time       `json:"created_at" bson:"created_at"`
	AccessedAt    time.Time       `json:"accessed_at" bson:"accessed_at"`
}

// =============================================================================
// Operation Log (User History for all pipeline stages)
// =============================================================================

// OperationType identifies what kind of pipeline operation was performed.
type OperationType string

const (
	OpTypeParse  OperationType = "parse"
	OpTypeLayout OperationType = "layout"
	OpTypeRender OperationType = "render"
)

// Operation represents any pipeline operation by a user.
// This enables tracking "which user triggered how many layouts" etc.
type Operation struct {
	ID        string           `json:"id" bson:"_id,omitempty"`
	UserID    string           `json:"user_id" bson:"user_id"`
	Type      OperationType    `json:"type" bson:"type"`
	Scope     Scope            `json:"scope" bson:"scope"`
	Source    OperationSource  `json:"source" bson:"source"`
	GraphID   string           `json:"graph_id,omitempty" bson:"graph_id,omitempty"`
	RenderID  string           `json:"render_id,omitempty" bson:"render_id,omitempty"`
	Options   OperationOptions `json:"options" bson:"options"`
	Stats     OperationStats   `json:"stats" bson:"stats"`
	CreatedAt time.Time        `json:"created_at" bson:"created_at"`
}

// OperationSource identifies what was processed.
type OperationSource struct {
	Type             string `json:"type" bson:"type"` // "package", "manifest", or "repo"
	Registry         string `json:"registry,omitempty" bson:"registry,omitempty"`
	Language         string `json:"language,omitempty" bson:"language,omitempty"`
	Package          string `json:"package,omitempty" bson:"package,omitempty"`
	Repo             string `json:"repo,omitempty" bson:"repo,omitempty"`
	ManifestFilename string `json:"manifest_filename,omitempty" bson:"manifest_filename,omitempty"`
	ManifestHash     string `json:"manifest_hash,omitempty" bson:"manifest_hash,omitempty"`
}

// OperationOptions captures the options used for the operation.
type OperationOptions struct {
	// Parse options
	MaxDepth  int  `json:"max_depth,omitempty" bson:"max_depth,omitempty"`
	MaxNodes  int  `json:"max_nodes,omitempty" bson:"max_nodes,omitempty"`
	Normalize bool `json:"normalize,omitempty" bson:"normalize,omitempty"`
	Enrich    bool `json:"enrich,omitempty" bson:"enrich,omitempty"`

	// Layout options
	VizType   string  `json:"viz_type,omitempty" bson:"viz_type,omitempty"`
	Width     float64 `json:"width,omitempty" bson:"width,omitempty"`
	Height    float64 `json:"height,omitempty" bson:"height,omitempty"`
	Ordering  string  `json:"ordering,omitempty" bson:"ordering,omitempty"`
	Merge     bool    `json:"merge,omitempty" bson:"merge,omitempty"`
	Randomize bool    `json:"randomize,omitempty" bson:"randomize,omitempty"`

	// Render options
	Formats   []string `json:"formats,omitempty" bson:"formats,omitempty"`
	Style     string   `json:"style,omitempty" bson:"style,omitempty"`
	ShowEdges bool     `json:"show_edges,omitempty" bson:"show_edges,omitempty"`
}

// OperationStats captures metrics about the operation.
type OperationStats struct {
	NodeCount  int   `json:"node_count" bson:"node_count"`
	EdgeCount  int   `json:"edge_count" bson:"edge_count"`
	DurationMs int64 `json:"duration_ms" bson:"duration_ms"`
	CacheHit   bool  `json:"cache_hit" bson:"cache_hit"`
}

// =============================================================================
// Rate Limiting / Quotas
// =============================================================================

// QuotaConfig defines rate limits and storage quotas per user.
type QuotaConfig struct {
	// Rate limits (operations per hour)
	MaxParsesPerHour  int `json:"max_parses_per_hour"`
	MaxLayoutsPerHour int `json:"max_layouts_per_hour"`
	MaxRendersPerHour int `json:"max_renders_per_hour"`

	// Storage limits
	MaxStorageBytes  int64 `json:"max_storage_bytes"`
	MaxRendersStored int   `json:"max_renders_stored"`
}

// DefaultQuotaConfig returns sensible defaults for rate limiting.
func DefaultQuotaConfig() QuotaConfig {
	return QuotaConfig{
		MaxParsesPerHour:  100,
		MaxLayoutsPerHour: 200,
		MaxRendersPerHour: 100,
		MaxStorageBytes:   500 * 1024 * 1024, // 500 MB
		MaxRendersStored:  1000,
	}
}

// QuotaUsage tracks current usage against quotas.
type QuotaUsage struct {
	ParsesThisHour   int   `json:"parses_this_hour"`
	LayoutsThisHour  int   `json:"layouts_this_hour"`
	RendersThisHour  int   `json:"renders_this_hour"`
	StorageBytesUsed int64 `json:"storage_bytes_used"`
	RendersStored    int   `json:"renders_stored"`
}
