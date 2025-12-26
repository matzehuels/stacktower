package artifact

import (
	"bytes"
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/infra/cache"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
)

// ProdBackend implements Backend using the production two-tier cache
// (Redis for lookups + MongoDB for storage).
type ProdBackend struct {
	cache cache.Cache
}

// NewProdBackend creates a new production backend wrapping an existing cache.
func NewProdBackend(c cache.Cache) *ProdBackend {
	return &ProdBackend{cache: c}
}

// GetGraph retrieves a cached graph.
func (b *ProdBackend) GetGraph(ctx context.Context, hash string) (*dag.DAG, bool, error) {
	// Check lookup cache for entry
	entry, err := b.cache.GetGraphEntry(ctx, hash)
	if err != nil || entry == nil || entry.IsExpired() {
		return nil, false, nil
	}

	// Fetch from store using the stored ID
	stored, err := b.cache.GetGraph(ctx, entry.MongoID)
	if err != nil || stored == nil {
		return nil, false, nil
	}

	// Deserialize graph
	g, err := pkgio.ReadJSON(bytes.NewReader(stored.Data))
	if err != nil {
		return nil, false, nil
	}

	return g, true, nil
}

// PutGraph stores a graph.
func (b *ProdBackend) PutGraph(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration) error {
	// Serialize graph
	var buf bytes.Buffer
	if err := pkgio.WriteJSON(g, &buf); err != nil {
		return err
	}
	data := buf.Bytes()

	// Create graph record
	stored := &cache.Graph{
		ContentHash: Hash(data),
		NodeCount:   g.NodeCount(),
		EdgeCount:   g.EdgeCount(),
		Data:        data,
	}

	// Store in MongoDB
	if err := b.cache.StoreGraph(ctx, stored); err != nil {
		return err
	}

	// Update Redis lookup
	return b.cache.SetGraphEntry(ctx, hash, &cache.CacheEntry{
		MongoID:   stored.ID,
		ExpiresAt: time.Now().Add(ttl),
	})
}

// GetLayout retrieves cached layout data.
func (b *ProdBackend) GetLayout(ctx context.Context, hash string) ([]byte, bool, error) {
	// For layouts, we use the render entry mechanism with a special prefix
	layoutKey := "layout:" + hash
	entry, err := b.cache.GetRenderEntry(ctx, layoutKey)
	if err != nil || entry == nil || entry.IsExpired() {
		return nil, false, nil
	}

	// Get from artifact storage
	data, err := b.cache.GetArtifact(ctx, entry.MongoID)
	if err != nil {
		return nil, false, nil
	}

	return data, true, nil
}

// PutLayout stores layout data.
func (b *ProdBackend) PutLayout(ctx context.Context, hash string, data []byte, ttl time.Duration) error {
	layoutKey := "layout:" + hash

	// Store as artifact
	artifactID, err := b.cache.StoreArtifact(ctx, "layout", hash+".json", data)
	if err != nil {
		return err
	}

	// Update lookup
	return b.cache.SetRenderEntry(ctx, layoutKey, &cache.CacheEntry{
		MongoID:   artifactID,
		ExpiresAt: time.Now().Add(ttl),
	})
}

// GetRender retrieves a cached render artifact.
func (b *ProdBackend) GetRender(ctx context.Context, hash, format string) ([]byte, bool, error) {
	renderKey := "render:" + hash + ":" + format
	entry, err := b.cache.GetRenderEntry(ctx, renderKey)
	if err != nil || entry == nil || entry.IsExpired() {
		return nil, false, nil
	}

	data, err := b.cache.GetArtifact(ctx, entry.MongoID)
	if err != nil {
		return nil, false, nil
	}

	return data, true, nil
}

// PutRender stores a render artifact.
func (b *ProdBackend) PutRender(ctx context.Context, hash, format string, data []byte, ttl time.Duration) error {
	renderKey := "render:" + hash + ":" + format

	// Store as artifact
	artifactID, err := b.cache.StoreArtifact(ctx, "render", hash+"."+format, data)
	if err != nil {
		return err
	}

	// Update lookup
	return b.cache.SetRenderEntry(ctx, renderKey, &cache.CacheEntry{
		MongoID:   artifactID,
		ExpiresAt: time.Now().Add(ttl),
	})
}

// GetHTTP retrieves a cached HTTP response.
func (b *ProdBackend) GetHTTP(ctx context.Context, namespace, key string) ([]byte, bool, error) {
	cacheKey := namespace + HashKey(key)
	return b.cache.GetHTTP(ctx, cacheKey)
}

// SetHTTP stores an HTTP response.
func (b *ProdBackend) SetHTTP(ctx context.Context, namespace, key string, data []byte, ttl time.Duration) error {
	cacheKey := namespace + HashKey(key)
	return b.cache.SetHTTP(ctx, cacheKey, data, ttl)
}

// DeleteHTTP removes a cached HTTP response.
func (b *ProdBackend) DeleteHTTP(ctx context.Context, namespace, key string) error {
	cacheKey := namespace + HashKey(key)
	return b.cache.DeleteHTTP(ctx, cacheKey)
}

// Close releases resources.
func (b *ProdBackend) Close() error {
	return b.cache.Close()
}

// Cache returns the underlying cache for advanced operations.
// This is useful for API endpoints that need direct cache access (e.g., history).
func (b *ProdBackend) Cache() cache.Cache {
	return b.cache
}

// Ensure ProdBackend implements Backend
var _ Backend = (*ProdBackend)(nil)
