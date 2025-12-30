package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NewObjectID generates a new MongoDB-compatible ObjectID string.
// Use this when creating Render documents to ensure the ID is valid for MongoDB operations.
func NewObjectID() string {
	return primitive.NewObjectID().Hex()
}

// ToJSONSafe converts BSON-specific types (primitive.D, primitive.A, etc.) to
// standard Go types (map[string]interface{}, []interface{}) that serialize
// correctly to JSON. This is necessary because MongoDB's primitive.D serializes
// to JSON as an array of key-value pairs, not as a JSON object.
func ToJSONSafe(v interface{}) interface{} {
	switch val := v.(type) {
	case primitive.D:
		// Convert ordered document to map
		m := make(map[string]interface{})
		for _, elem := range val {
			m[elem.Key] = ToJSONSafe(elem.Value)
		}
		return m
	case primitive.M:
		// Already a map, but recursively convert values
		m := make(map[string]interface{})
		for k, v := range val {
			m[k] = ToJSONSafe(v)
		}
		return m
	case primitive.A:
		// Convert BSON array to slice
		arr := make([]interface{}, len(val))
		for i, elem := range val {
			arr[i] = ToJSONSafe(elem)
		}
		return arr
	case []interface{}:
		// Recursively convert array elements
		arr := make([]interface{}, len(val))
		for i, elem := range val {
			arr[i] = ToJSONSafe(elem)
		}
		return arr
	case map[string]interface{}:
		// Recursively convert map values
		m := make(map[string]interface{})
		for k, v := range val {
			m[k] = ToJSONSafe(v)
		}
		return m
	default:
		// Return as-is (primitive types like string, int, bool, etc.)
		return v
	}
}

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
	MaxDepth  int  `json:"max_depth" bson:"max_depth"`
	MaxNodes  int  `json:"max_nodes" bson:"max_nodes"`
	Normalize bool `json:"normalize" bson:"normalize"`
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
	Data        interface{}  `json:"data" bson:"data"`                 // DAG as BSON document (queryable)
	CreatedAt   time.Time    `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" bson:"updated_at"`
}

// =============================================================================
// Render Document (Tier 2 - stored in DocumentStore/MongoDB)
// =============================================================================

// LayoutOptions defines parameters for layout computation.
type LayoutOptions struct {
	VizType   string  `json:"viz_type" bson:"viz_type"`
	Width     float64 `json:"width" bson:"width"`
	Height    float64 `json:"height" bson:"height"`
	Ordering  string  `json:"ordering,omitempty" bson:"ordering,omitempty"`
	Merge     bool    `json:"merge,omitempty" bson:"merge,omitempty"`
	Randomize bool    `json:"randomize,omitempty" bson:"randomize,omitempty"`
	Seed      uint64  `json:"seed,omitempty" bson:"seed,omitempty"`
}

// RenderOptions defines parameters for rendering output.
type RenderOptions struct {
	Formats   []string `json:"formats" bson:"formats"`
	Style     string   `json:"style,omitempty" bson:"style,omitempty"`
	ShowEdges bool     `json:"show_edges,omitempty" bson:"show_edges,omitempty"`
	Nebraska  bool     `json:"nebraska,omitempty" bson:"nebraska,omitempty"`
	Popups    bool     `json:"popups,omitempty" bson:"popups,omitempty"`
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
	SVG  string `json:"svg,omitempty" bson:"svg,omitempty"`
	PNG  string `json:"png,omitempty" bson:"png,omitempty"`
	PDF  string `json:"pdf,omitempty" bson:"pdf,omitempty"`
	JSON string `json:"json,omitempty" bson:"json,omitempty"`
}

// NebraskaRanking represents a maintainer's influence score.
type NebraskaRanking struct {
	Maintainer string            `json:"maintainer" bson:"maintainer"`
	Score      float64           `json:"score" bson:"score"`
	Packages   []NebraskaPackage `json:"packages" bson:"packages"`
}

// NebraskaPackage represents a package maintained by someone.
type NebraskaPackage struct {
	Package string `json:"package" bson:"package"`
	Role    string `json:"role" bson:"role"` // "owner", "lead", or "maintainer"
	URL     string `json:"url,omitempty" bson:"url,omitempty"`
	Depth   int    `json:"depth,omitempty" bson:"depth,omitempty"`
}

// Render represents a user's visualization stored in the DocumentStore.
type Render struct {
	ID                string          `json:"id" bson:"_id,omitempty"`
	UserID            string          `json:"user_id" bson:"user_id"`
	Source            RenderSource    `json:"source" bson:"source"`
	GraphID           string          `json:"graph_id" bson:"graph_id"`
	GraphHash         string          `json:"graph_hash" bson:"graph_hash"`
	LayoutOptions     LayoutOptions   `json:"layout_options" bson:"layout_options"`
	RenderOptions     RenderOptions   `json:"render_options" bson:"render_options"`
	Layout            interface{}     `json:"layout" bson:"layout"` // Layout as BSON document (includes nebraska rankings, queryable)
	Artifacts         RenderArtifacts `json:"artifacts" bson:"artifacts"`
	NodeCount         int             `json:"node_count" bson:"node_count"`
	EdgeCount         int             `json:"edge_count" bson:"edge_count"`
	CreatedAt         time.Time       `json:"created_at" bson:"created_at"`
	AccessedAt        time.Time       `json:"accessed_at" bson:"accessed_at"`
	AvailableVizTypes []string        `json:"available_viz_types,omitempty" bson:"available_viz_types,omitempty"`
}

// Nebraska extracts Nebraska rankings from the Layout document.
// Returns nil if Layout is empty or doesn't contain nebraska data.
func (r *Render) Nebraska() []NebraskaRanking {
	return ParseNebraskaFromLayout(r.Layout)
}

// ParseNebraskaFromLayout extracts Nebraska rankings from layout data.
// Handles both interface{} (BSON document) and []byte (legacy) formats.
func ParseNebraskaFromLayout(layout interface{}) []NebraskaRanking {
	if layout == nil {
		return nil
	}

	// Handle legacy []byte format
	if layoutBytes, ok := layout.([]byte); ok {
		if len(layoutBytes) == 0 {
			return nil
		}
		var data struct {
			Nebraska []NebraskaRanking `json:"nebraska"`
		}
		if err := json.Unmarshal(layoutBytes, &data); err != nil {
			return nil
		}
		return data.Nebraska
	}

	// Convert BSON types (primitive.D, primitive.M) to standard Go types
	// before trying to access the nebraska field
	safeLayout := ToJSONSafe(layout)

	// Handle map format (after conversion from BSON)
	if layoutMap, ok := safeLayout.(map[string]interface{}); ok {
		if nebraskaData, exists := layoutMap["nebraska"]; exists {
			// Convert to JSON and back to properly typed structs
			nebraskaBytes, err := json.Marshal(nebraskaData)
			if err != nil {
				return nil
			}
			var rankings []NebraskaRanking
			if err := json.Unmarshal(nebraskaBytes, &rankings); err != nil {
				return nil
			}
			return rankings
		}
	}

	return nil
}

// =============================================================================
// Operation Types (for rate limiting)
// =============================================================================

// OperationType identifies what kind of pipeline operation was performed.
// Used by rate limiting to track operations per type.
type OperationType string

const (
	OpTypeParse  OperationType = "parse"
	OpTypeLayout OperationType = "layout"
	OpTypeRender OperationType = "render"
)

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

// =============================================================================
// Package Suggestions (for autocomplete)
// =============================================================================

// PackageSuggestion represents a package that can be suggested in autocomplete.
type PackageSuggestion struct {
	Package    string `json:"package" bson:"package"`
	Language   string `json:"language" bson:"language"`
	Popularity int    `json:"popularity" bson:"popularity"` // Number of users with this in their library
}

// =============================================================================
// Explore
// =============================================================================

// ExploreVizType represents a single viz type within an explore entry.
type ExploreVizType struct {
	VizType     string `json:"viz_type" bson:"viz_type"`
	RenderID    string `json:"render_id" bson:"render_id"`
	GraphID     string `json:"graph_id,omitempty" bson:"graph_id,omitempty"`
	ArtifactSVG string `json:"artifact_svg,omitempty" bson:"artifact_svg,omitempty"`
	ArtifactPNG string `json:"artifact_png,omitempty" bson:"artifact_png,omitempty"`
	ArtifactPDF string `json:"artifact_pdf,omitempty" bson:"artifact_pdf,omitempty"`
}

// ExploreEntry represents a grouped package in the explore view.
// Multiple viz types (tower, nodelink) for the same package are grouped together.
type ExploreEntry struct {
	Source          RenderSource     `json:"source" bson:"source"`
	NodeCount       int              `json:"node_count" bson:"node_count"`
	EdgeCount       int              `json:"edge_count" bson:"edge_count"`
	CreatedAt       time.Time        `json:"created_at" bson:"created_at"`       // Most recent
	VizTypes        []ExploreVizType `json:"viz_types" bson:"viz_types"`         // Available visualizations
	PopularityCount int              `json:"popularity_count" bson:"popularity"` // Users with this in collection
}

// =============================================================================
// User Library (saved towers)
// =============================================================================

// LibraryEntry represents a package saved in a user's library.
// When a user renders a public package or saves one from explore, it's added here.
type LibraryEntry struct {
	ID       string    `json:"id" bson:"_id,omitempty"`
	UserID   string    `json:"user_id" bson:"user_id"`
	Language string    `json:"language" bson:"language"`
	Package  string    `json:"package" bson:"package"`
	SavedAt  time.Time `json:"saved_at" bson:"saved_at"`
}
