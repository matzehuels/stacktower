package cache

import (
	"context"
	"time"
)

// CombinedCache wraps a LookupCache (Tier 1) and Store (Tier 2) into a unified Cache.
// This is the production configuration: Redis for fast TTL lookups, MongoDB for durable storage.
type CombinedCache struct {
	lookup LookupCache
	store  Store
}

// NewCombinedCache creates a cache from separate lookup and store backends.
func NewCombinedCache(lookup LookupCache, store Store) *CombinedCache {
	return &CombinedCache{lookup: lookup, store: store}
}

// LookupCache methods (Tier 1 - Redis/Memory)

func (c *CombinedCache) GetGraphEntry(ctx context.Context, key string) (*CacheEntry, error) {
	return c.lookup.GetGraphEntry(ctx, key)
}

func (c *CombinedCache) SetGraphEntry(ctx context.Context, key string, entry *CacheEntry) error {
	return c.lookup.SetGraphEntry(ctx, key, entry)
}

func (c *CombinedCache) GetRenderEntry(ctx context.Context, key string) (*CacheEntry, error) {
	return c.lookup.GetRenderEntry(ctx, key)
}

func (c *CombinedCache) SetRenderEntry(ctx context.Context, key string, entry *CacheEntry) error {
	return c.lookup.SetRenderEntry(ctx, key, entry)
}

func (c *CombinedCache) GetHTTP(ctx context.Context, key string) ([]byte, bool, error) {
	return c.lookup.GetHTTP(ctx, key)
}

func (c *CombinedCache) SetHTTP(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	return c.lookup.SetHTTP(ctx, key, data, ttl)
}

func (c *CombinedCache) DeleteHTTP(ctx context.Context, key string) error {
	return c.lookup.DeleteHTTP(ctx, key)
}

// Store methods (Tier 2 - MongoDB)

func (c *CombinedCache) GetGraph(ctx context.Context, id string) (*Graph, error) {
	return c.store.GetGraph(ctx, id)
}

func (c *CombinedCache) StoreGraph(ctx context.Context, graph *Graph) error {
	return c.store.StoreGraph(ctx, graph)
}

func (c *CombinedCache) GetRender(ctx context.Context, id string) (*Render, error) {
	return c.store.GetRender(ctx, id)
}

func (c *CombinedCache) StoreRender(ctx context.Context, render *Render) error {
	return c.store.StoreRender(ctx, render)
}

func (c *CombinedCache) DeleteRender(ctx context.Context, id string) error {
	return c.store.DeleteRender(ctx, id)
}

func (c *CombinedCache) ListRenders(ctx context.Context, userID string, limit, offset int) ([]*Render, int64, error) {
	return c.store.ListRenders(ctx, userID, limit, offset)
}

func (c *CombinedCache) StoreArtifact(ctx context.Context, renderID, filename string, data []byte) (string, error) {
	return c.store.StoreArtifact(ctx, renderID, filename, data)
}

func (c *CombinedCache) GetArtifact(ctx context.Context, artifactID string) ([]byte, error) {
	return c.store.GetArtifact(ctx, artifactID)
}

func (c *CombinedCache) Close() error {
	// Close both - ignore errors from first to ensure both are closed
	lookupErr := c.lookup.Close()
	storeErr := c.store.Close()
	if lookupErr != nil {
		return lookupErr
	}
	return storeErr
}

var _ Cache = (*CombinedCache)(nil)
