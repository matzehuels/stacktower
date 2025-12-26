// Package cache provides a two-tier caching system for graphs, renders, and artifacts.
//
// This package stores dependency graphs and render results. It also provides
// HTTP response caching for package registry APIs (PyPI, npm, etc.) via the
// LookupCache interface, which is used by artifact.Backend.
//
// # Architecture
//
// Tier 1 - LookupCache (Redis/Memory):
//   - Fast TTL-based index: "Do we have this content?"
//   - Stores cache keys with TTL and pointers to MongoDB document IDs
//   - Expires entries automatically based on TTL
//   - Implemented by: infra.Redis.Cache(), MemoryCache
//
// Tier 2 - Store (MongoDB/Memory):
//   - Durable storage for actual data
//   - Stores Graph documents, Render documents, and binary Artifacts (via GridFS)
//   - Enables querying, history, analytics
//   - Implemented by: infra.Mongo.Store(), MemoryCache
//
// Cache = LookupCache + Store combined (use CombinedCache for production).
//
// # Flow
//
//  1. Check LookupCache for cache key → get MongoDB document ID
//  2. If HIT and within TTL → fetch from Store using stored ID
//  3. If MISS or expired → compute, store in Store, update LookupCache
//
// # Usage
//
// For local development (CLI):
//
//	cache := cache.NewMemoryCache()  // Both tiers in memory
//
// For production (API/Worker):
//
//	redis, _ := infra.NewRedis(ctx, cfg.Redis)
//	mongo, _ := infra.NewMongo(ctx, cfg.Mongo)
//	cache := cache.NewCombinedCache(redis.Cache(), mongo.Store())
//
// # Privacy Model
//
// Public packages (global scope):
//   - Cache key: "graph:global:python:flask:opts_hash"
//   - Shared by all users
//
// Private manifests (user scope):
//   - Cache key: "graph:user:123:manifest_hash:opts_hash"
//   - Only accessible to that user
//
// Renders are always user-scoped (users don't share layouts).
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/matzehuels/stacktower/pkg/infra/common"
)

// Sentinel errors for cache operations.
var (
	// ErrNotFound is returned when a requested item does not exist.
	ErrNotFound = common.ErrNotFound

	// ErrExpired is returned when a cache entry has exceeded its TTL.
	ErrExpired = common.ErrExpired
)

// Scope indicates whether data is globally shared or user-private.
type Scope string

const (
	// ScopeGlobal means data is shared across all users (public packages).
	ScopeGlobal Scope = "global"

	// ScopeUser means data is private to a specific user (private repos).
	ScopeUser Scope = "user"
)

// GraphOptions defines parameters that affect graph resolution.
// Different options produce different graphs, so they're part of the cache key.
type GraphOptions struct {
	MaxDepth  int  `json:"max_depth"`
	MaxNodes  int  `json:"max_nodes"`
	Normalize bool `json:"normalize"`
}

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

// CacheEntry is stored in Redis/memory tier - just a pointer to MongoDB.
type CacheEntry struct {
	MongoID   string    `json:"mongo_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IsExpired returns true if the cache entry has exceeded its TTL.
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// Graph represents a stored dependency graph in MongoDB.
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

// RenderSource identifies what was rendered.
type RenderSource struct {
	Type             string `json:"type" bson:"type"` // "package" or "manifest"
	Language         string `json:"language" bson:"language"`
	Package          string `json:"package,omitempty" bson:"package,omitempty"`
	Repo             string `json:"repo,omitempty" bson:"repo,omitempty"`
	ManifestFilename string `json:"manifest_filename,omitempty" bson:"manifest_filename,omitempty"`
	ManifestHash     string `json:"manifest_hash,omitempty" bson:"manifest_hash,omitempty"`
}

// Render represents a user's visualization stored in MongoDB.
type Render struct {
	ID            string        `json:"id" bson:"_id,omitempty"`
	UserID        string        `json:"user_id" bson:"user_id"`
	Source        RenderSource  `json:"source" bson:"source"`
	GraphID       string        `json:"graph_id" bson:"graph_id"`
	GraphHash     string        `json:"graph_hash" bson:"graph_hash"`
	LayoutOptions LayoutOptions `json:"layout_options" bson:"layout_options"`
	RenderOptions RenderOptions `json:"render_options" bson:"render_options"`
	Layout        []byte        `json:"layout" bson:"layout"` // JSON-encoded layout
	Artifacts     Artifacts     `json:"artifacts" bson:"artifacts"`
	NodeCount     int           `json:"node_count" bson:"node_count"`
	EdgeCount     int           `json:"edge_count" bson:"edge_count"`
	CreatedAt     time.Time     `json:"created_at" bson:"created_at"`
	AccessedAt    time.Time     `json:"accessed_at" bson:"accessed_at"`
}

// Artifacts holds references to rendered output files in GridFS.
type Artifacts struct {
	SVG string `json:"svg,omitempty" bson:"svg,omitempty"` // GridFS ID
	PNG string `json:"png,omitempty" bson:"png,omitempty"`
	PDF string `json:"pdf,omitempty" bson:"pdf,omitempty"`
}

// LookupCache is Tier 1: fast TTL-based index (Redis or in-memory).
//
// It answers the question: "Do we have this content cached?"
// If yes, it returns a CacheEntry with the MongoDB document ID.
// If no (or expired), the caller should compute the value, store it
// in the Store (Tier 2), and then call Set*Entry to update the index.
//
// Additionally, LookupCache provides direct byte storage for HTTP response
// caching (used by package registry integrations). HTTP responses are small
// and transient, so they live entirely in Tier 1 without MongoDB storage.
//
// Implementations:
//   - Production: infra.Redis.Cache() returns a Redis-backed LookupCache
//   - Development: MemoryCache implements LookupCache (and Store)
type LookupCache interface {
	// GetGraphEntry checks if a graph is cached.
	// key is typically a content hash or input hash.
	// Returns (entry, nil) on hit, (nil, nil) on miss, (nil, err) on error.
	GetGraphEntry(ctx context.Context, key string) (*CacheEntry, error)

	// SetGraphEntry stores a graph cache entry.
	// entry.MongoID should be the ID returned by Store.StoreGraph().
	SetGraphEntry(ctx context.Context, key string, entry *CacheEntry) error

	// GetRenderEntry checks if a render is cached.
	GetRenderEntry(ctx context.Context, key string) (*CacheEntry, error)

	// SetRenderEntry stores a render cache entry.
	SetRenderEntry(ctx context.Context, key string, entry *CacheEntry) error

	// GetHTTP retrieves a cached HTTP response.
	// Used for caching package registry API responses.
	// Returns (data, true, nil) on hit, (nil, false, nil) on miss.
	GetHTTP(ctx context.Context, key string) ([]byte, bool, error)

	// SetHTTP stores an HTTP response with TTL.
	SetHTTP(ctx context.Context, key string, data []byte, ttl time.Duration) error

	// DeleteHTTP removes a cached HTTP response.
	DeleteHTTP(ctx context.Context, key string) error

	Close() error
}

// Store is Tier 2: durable document storage (MongoDB or in-memory).
//
// It stores the actual data: Graph documents, Render documents, and
// binary Artifacts (SVG, PNG, PDF via GridFS).
//
// The IDs returned by Store methods are what go into CacheEntry.MongoID
// for the LookupCache.
//
// Implementations:
//   - Production: infra.Mongo.Store() returns a MongoDB-backed Store
//   - Development: MemoryCache implements Store (and LookupCache)
type Store interface {
	// GetGraph retrieves a graph by its MongoDB ObjectID.
	// Returns (graph, nil) on success, (nil, nil) if not found.
	GetGraph(ctx context.Context, id string) (*Graph, error)

	// StoreGraph saves a graph. Sets graph.ID if empty (new graph).
	// Returns the stored graph ID via graph.ID.
	StoreGraph(ctx context.Context, graph *Graph) error

	// GetRender retrieves a render by its MongoDB ObjectID.
	GetRender(ctx context.Context, id string) (*Render, error)

	// StoreRender saves a render. Sets render.ID if empty.
	StoreRender(ctx context.Context, render *Render) error

	// DeleteRender removes a render and its associated artifacts.
	DeleteRender(ctx context.Context, id string) error

	// ListRenders returns a user's render history (paginated).
	ListRenders(ctx context.Context, userID string, limit, offset int) ([]*Render, int64, error)

	// StoreArtifact saves a binary file (SVG, PNG, PDF) to GridFS.
	// Returns the artifact ID for later retrieval.
	StoreArtifact(ctx context.Context, renderID, filename string, data []byte) (string, error)

	// GetArtifact retrieves a binary file from GridFS by ID.
	GetArtifact(ctx context.Context, artifactID string) ([]byte, error)

	Close() error
}

// Cache combines both tiers (LookupCache + Store) for unified access.
//
// Use this interface when you need access to both tiers, such as in
// artifact.ProdBackend or the Worker.
//
// Implementations:
//   - Production: cache.NewCombinedCache(redis.Cache(), mongo.Store())
//   - Development: cache.NewMemoryCache() implements Cache directly
type Cache interface {
	LookupCache
	Store
}

// =============================================================================
// Key Generation
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

// ContentHash computes a SHA256 hash of content.
func ContentHash(data []byte) string {
	return common.HashBytes(data)
}

// OptionsHash computes a short hash of options for cache keys.
func OptionsHash(opts interface{}) string {
	return common.HashJSON(opts)[:16] // hex hash, 8 bytes = 16 chars
}

// =============================================================================
// TTL Configuration
// =============================================================================

// Default TTLs for cache entries
const (
	// GraphTTL is how long a graph cache entry is valid (7 days).
	// After this, we'll re-resolve to pick up new versions.
	GraphTTL = common.GraphTTL

	// RenderTTL is how long a render cache entry is valid (90 days).
	// Renders are deterministic given layout hash.
	RenderTTL = common.RenderTTL
)
