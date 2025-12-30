package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
)

// DistributedBackend implements Backend using a two-tier distributed cache:
// Redis (Index) for fast TTL-based lookups + MongoDB (DocumentStore) for durable storage.
//
// This is the recommended backend for API and Worker deployments.
//
// Usage:
//
//	redis, _ := infra.NewRedis(ctx, cfg.Redis)
//	mongo, _ := infra.NewMongo(ctx, cfg.Mongo)
//	backend := storage.NewDistributedBackend(redis.Index(), mongo.DocumentStore(), redis.HTTPCache(), redis.RateLimiter())
type DistributedBackend struct {
	index     Index
	docstore  DocumentStore
	httpCache HTTPCache
	limiter   RateLimiter
}

// NewDistributedBackend creates a production backend with Redis + MongoDB.
// The httpCache parameter provides HTTP response caching (typically backed by Redis).
// The limiter parameter provides rate limiting (typically backed by Redis).
func NewDistributedBackend(index Index, docstore DocumentStore, httpCache HTTPCache, limiter RateLimiter) *DistributedBackend {
	return &DistributedBackend{
		index:     index,
		docstore:  docstore,
		httpCache: httpCache,
		limiter:   limiter,
	}
}

// =============================================================================
// Backend interface implementation
// =============================================================================

func (b *DistributedBackend) GetGraph(ctx context.Context, hash string) (*dag.DAG, bool, error) {
	// Check index for entry
	entry, err := b.index.GetGraphEntry(ctx, hash)
	if err != nil || entry == nil || entry.IsExpired() {
		return nil, false, nil
	}

	// Fetch from DocumentStore using the stored ID
	stored, err := b.docstore.GetGraphDoc(ctx, entry.DocumentID)
	if err != nil || stored == nil {
		return nil, false, nil
	}

	// Deserialize graph - convert BSON to JSON-safe format, then to DAG
	// (primitive.D -> map, primitive.A -> slice, etc.)
	jsonSafe := ToJSONSafe(stored.Data)
	jsonData, err := json.Marshal(jsonSafe)
	if err != nil {
		return nil, false, nil
	}
	g, err := pkgio.ReadJSON(bytes.NewReader(jsonData))
	if err != nil {
		return nil, false, nil
	}

	return g, true, nil
}

func (b *DistributedBackend) PutGraph(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration) error {
	// Use PutGraphScoped with global scope (legacy compatibility)
	return b.PutGraphScoped(ctx, hash, g, ttl, ScopeGlobal, "", GraphMeta{})
}

func (b *DistributedBackend) GetGraphScoped(ctx context.Context, hash string, userID string) (*dag.DAG, bool, error) {
	// Check index for entry
	entry, err := b.index.GetGraphEntry(ctx, hash)
	if err != nil || entry == nil || entry.IsExpired() {
		return nil, false, nil
	}

	// Fetch from DocumentStore using scoped method (enforces authorization)
	stored, err := b.docstore.GetGraphDocScoped(ctx, entry.DocumentID, userID)
	if err != nil {
		if err == ErrAccessDenied {
			return nil, false, nil // Treat as cache miss for unauthorized access
		}
		return nil, false, nil
	}
	if stored == nil {
		return nil, false, nil
	}

	// Deserialize graph - convert BSON to JSON-safe format, then to DAG
	// (primitive.D -> map, primitive.A -> slice, etc.)
	jsonSafe := ToJSONSafe(stored.Data)
	jsonData, err := json.Marshal(jsonSafe)
	if err != nil {
		return nil, false, nil
	}
	g, err := pkgio.ReadJSON(bytes.NewReader(jsonData))
	if err != nil {
		return nil, false, nil
	}

	return g, true, nil
}

func (b *DistributedBackend) PutGraphScoped(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration, scope Scope, userID string, meta GraphMeta) error {
	// Serialize graph to JSON
	var buf bytes.Buffer
	if err := pkgio.WriteJSON(g, &buf); err != nil {
		return err
	}
	jsonData := buf.Bytes()

	// Unmarshal JSON to interface{} for BSON document storage
	var dataDoc interface{}
	if err := json.Unmarshal(jsonData, &dataDoc); err != nil {
		return err
	}

	// Create graph record with full metadata
	stored := &Graph{
		Scope:       scope,
		UserID:      userID, // Empty for ScopeGlobal
		Language:    meta.Language,
		Package:     meta.Package,
		Repo:        meta.Repo,
		Options:     meta.Options, // Store options for cache key reconstruction
		ContentHash: Hash(jsonData),
		NodeCount:   g.NodeCount(),
		EdgeCount:   g.EdgeCount(),
		Data:        dataDoc, // Store as BSON document
	}

	// For ScopeUser, userID is required
	if scope == ScopeUser && userID == "" {
		return ErrAccessDenied
	}

	// Store in DocumentStore (MongoDB)
	if err := b.docstore.StoreGraphDoc(ctx, stored); err != nil {
		return err
	}

	// Update Index (Redis)
	return b.index.SetGraphEntry(ctx, hash, &CacheEntry{
		DocumentID: stored.ID,
		ExpiresAt:  time.Now().Add(ttl),
	})
}

func (b *DistributedBackend) GetLayout(ctx context.Context, hash string) ([]byte, bool, error) {
	// hash already includes scope prefix from Keys.LayoutCacheKey (e.g., "layout:global:..." or "layout:user:...")
	entry, err := b.index.GetRenderEntry(ctx, hash)
	if err != nil || entry == nil || entry.IsExpired() {
		return nil, false, nil
	}

	// Get from artifact storage
	data, err := b.docstore.GetArtifact(ctx, entry.DocumentID)
	if err != nil {
		return nil, false, nil
	}

	return data, true, nil
}

func (b *DistributedBackend) PutLayout(ctx context.Context, hash string, data []byte, ttl time.Duration, scope Scope, userID string) error {
	// hash already includes scope prefix from Keys.LayoutCacheKey

	// Store as artifact
	// Note: We use "layout" as renderID and hash as filename to namespace it,
	// but we pass userID to enforce ownership/scoping in the underlying store.
	artifactID, err := b.docstore.StoreArtifact(ctx, "layout", hash+".json", data, userID)
	if err != nil {
		return err
	}

	// Update index
	return b.index.SetRenderEntry(ctx, hash, &CacheEntry{
		DocumentID: artifactID,
		ExpiresAt:  time.Now().Add(ttl),
	})
}

func (b *DistributedBackend) GetRender(ctx context.Context, hash, format string) ([]byte, bool, error) {
	// hash already includes scope and format prefix from Keys.ArtifactCacheKey
	// (e.g., "artifact:global:..." or "artifact:user:...")
	entry, err := b.index.GetRenderEntry(ctx, hash)
	if err != nil || entry == nil || entry.IsExpired() {
		return nil, false, nil
	}

	data, err := b.docstore.GetArtifact(ctx, entry.DocumentID)
	if err != nil {
		return nil, false, nil
	}

	return data, true, nil
}

func (b *DistributedBackend) PutRender(ctx context.Context, hash, format string, data []byte, ttl time.Duration, scope Scope, userID string) error {
	// hash already includes scope and format prefix from Keys.ArtifactCacheKey

	// Store as artifact
	// We use "render" as renderID prefix for cache entries
	artifactID, err := b.docstore.StoreArtifact(ctx, "render", hash+"."+format, data, userID)
	if err != nil {
		return err
	}

	// Update index
	return b.index.SetRenderEntry(ctx, hash, &CacheEntry{
		DocumentID: artifactID,
		ExpiresAt:  time.Now().Add(ttl),
	})
}

func (b *DistributedBackend) GetHTTP(ctx context.Context, namespace, key string) ([]byte, bool, error) {
	cacheKey := namespace + ":" + HashKey(key)
	return b.httpCache.GetHTTP(ctx, cacheKey)
}

func (b *DistributedBackend) SetHTTP(ctx context.Context, namespace, key string, data []byte, ttl time.Duration) error {
	cacheKey := namespace + ":" + HashKey(key)
	return b.httpCache.SetHTTP(ctx, cacheKey, data, ttl)
}

func (b *DistributedBackend) DeleteHTTP(ctx context.Context, namespace, key string) error {
	cacheKey := namespace + ":" + HashKey(key)
	return b.httpCache.DeleteHTTP(ctx, cacheKey)
}

func (b *DistributedBackend) Close() error {
	// Close index and docstore - ignore errors from first to ensure both are closed
	indexErr := b.index.Close()
	docstoreErr := b.docstore.Close()
	if indexErr != nil {
		return indexErr
	}
	return docstoreErr
}

// =============================================================================
// Cache interface (Index + DocumentStore) for direct low-level access
// =============================================================================

// Index returns the underlying Index for direct access.
// This is useful for API/Worker when they need to set cache entries directly.
func (b *DistributedBackend) Index() Index {
	return b.index
}

// DocumentStore returns the underlying DocumentStore for direct access.
// This is useful for API/Worker when they need to store/retrieve user history.
func (b *DistributedBackend) DocumentStore() DocumentStore {
	return b.docstore
}

// RateLimiter returns the underlying RateLimiter for rate limit checks.
func (b *DistributedBackend) RateLimiter() RateLimiter {
	return b.limiter
}

// Ping checks if the backend services are reachable.
// Returns an error if any dependency (Redis or MongoDB) is unavailable.
func (b *DistributedBackend) Ping(ctx context.Context) error {
	if err := b.index.Ping(ctx); err != nil {
		return err
	}
	return b.docstore.Ping(ctx)
}

// =============================================================================
// Convenience methods for common operations
// =============================================================================

// CheckRateLimit checks if a user can perform an operation.
func (b *DistributedBackend) CheckRateLimit(ctx context.Context, userID string, opType OperationType, quota QuotaConfig) error {
	if b.limiter == nil {
		return nil // Rate limiting disabled
	}
	return b.limiter.CheckRateLimit(ctx, userID, opType, quota)
}

// Ensure DistributedBackend implements Backend
var _ Backend = (*DistributedBackend)(nil)
