// Package cache provides a two-tier caching system for Stacktower.
//
// # Architecture
//
// Tier 1 - Redis/Memory (fast TTL-based lookups):
//   - Stores cache keys with TTL and pointers to MongoDB document IDs
//   - Fast "do we have this?" check without hitting MongoDB
//   - Expires entries automatically based on TTL
//
// Tier 2 - MongoDB (durable storage):
//   - Stores actual graph data, layouts, and artifacts
//   - Data persists indefinitely (can be cleaned up periodically)
//   - Enables querying, history, analytics
//
// # Flow
//
//  1. Check Redis for cache key
//  2. If HIT and within TTL → fetch from MongoDB using stored ID
//  3. If MISS or expired → compute, store in MongoDB, update Redis
//
// # Privacy Model
//
// Public packages (global scope):
//   - Cache key: "graph:python:flask:opts_hash"
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

	"github.com/matzehuels/stacktower/pkg/config"
	pkgerr "github.com/matzehuels/stacktower/pkg/errors"
	"github.com/matzehuels/stacktower/pkg/hash"
)

// Sentinel errors for cache operations.
var (
	// ErrNotFound is returned when a requested item does not exist.
	ErrNotFound = pkgerr.ErrNotFound

	// ErrExpired is returned when a cache entry has exceeded its TTL.
	ErrExpired = pkgerr.ErrExpired
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

// LookupCache is the fast tier (Redis/memory) for TTL-based lookups.
type LookupCache interface {
	// Graph lookups
	GetGraphEntry(ctx context.Context, key string) (*CacheEntry, error)
	SetGraphEntry(ctx context.Context, key string, entry *CacheEntry) error
	DeleteGraphEntry(ctx context.Context, key string) error

	// Render lookups
	GetRenderEntry(ctx context.Context, key string) (*CacheEntry, error)
	SetRenderEntry(ctx context.Context, key string, entry *CacheEntry) error
	DeleteRenderEntry(ctx context.Context, key string) error

	Close() error
}

// Store is the durable tier (MongoDB) for actual data.
type Store interface {
	// Graph operations
	GetGraph(ctx context.Context, id string) (*Graph, error)
	StoreGraph(ctx context.Context, graph *Graph) error
	DeleteGraph(ctx context.Context, id string) error

	// Render operations
	GetRender(ctx context.Context, id string) (*Render, error)
	GetRenderByGraphAndOptions(ctx context.Context, userID, graphHash string, layoutOpts LayoutOptions) (*Render, error)
	StoreRender(ctx context.Context, render *Render) error
	DeleteRender(ctx context.Context, id string) error
	ListRenders(ctx context.Context, userID string, limit, offset int) ([]*Render, int64, error)

	// Artifact operations
	StoreArtifact(ctx context.Context, renderID, filename string, data []byte) (string, error)
	GetArtifact(ctx context.Context, artifactID string) ([]byte, error)

	Close() error
}

// Cache combines both tiers for unified access.
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
	return hash.Bytes(data)
}

// OptionsHash computes a short hash of options for cache keys.
func OptionsHash(opts interface{}) string {
	return hash.JSON(opts)[:16] // hex hash, 8 bytes = 16 chars
}

// =============================================================================
// TTL Configuration
// =============================================================================

// Default TTLs for cache entries
const (
	// GraphTTL is how long a graph cache entry is valid (7 days).
	// After this, we'll re-resolve to pick up new versions.
	GraphTTL = config.GraphTTL

	// RenderTTL is how long a render cache entry is valid (90 days).
	// Renders are deterministic given layout hash.
	RenderTTL = config.RenderTTL
)
