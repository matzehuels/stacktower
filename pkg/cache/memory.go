package cache

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryCache implements both LookupCache and Store for local development.
// In production, use RedisLookupCache + MongoStore instead.
type MemoryCache struct {
	mu sync.RWMutex

	// Lookup cache (tier 1)
	graphEntries  map[string]*CacheEntry
	renderEntries map[string]*CacheEntry

	// Durable store (tier 2)
	graphs    map[string]*Graph
	renders   map[string]*Render
	artifacts map[string][]byte
}

// NewMemoryCache creates a new in-memory cache for local development.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		graphEntries:  make(map[string]*CacheEntry),
		renderEntries: make(map[string]*CacheEntry),
		graphs:        make(map[string]*Graph),
		renders:       make(map[string]*Render),
		artifacts:     make(map[string][]byte),
	}
}

// =============================================================================
// LookupCache implementation (Tier 1)
// =============================================================================

func (c *MemoryCache) GetGraphEntry(ctx context.Context, key string) (*CacheEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.graphEntries[key]
	if !ok {
		return nil, nil
	}

	// Check expiration
	if entry.IsExpired() {
		return nil, nil
	}

	return entry, nil
}

func (c *MemoryCache) SetGraphEntry(ctx context.Context, key string, entry *CacheEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.graphEntries[key] = entry
	return nil
}

func (c *MemoryCache) DeleteGraphEntry(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.graphEntries, key)
	return nil
}

func (c *MemoryCache) GetRenderEntry(ctx context.Context, key string) (*CacheEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.renderEntries[key]
	if !ok {
		return nil, nil
	}

	if entry.IsExpired() {
		return nil, nil
	}

	return entry, nil
}

func (c *MemoryCache) SetRenderEntry(ctx context.Context, key string, entry *CacheEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.renderEntries[key] = entry
	return nil
}

func (c *MemoryCache) DeleteRenderEntry(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.renderEntries, key)
	return nil
}

// =============================================================================
// Store implementation (Tier 2)
// =============================================================================

func (c *MemoryCache) GetGraph(ctx context.Context, id string) (*Graph, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	graph, ok := c.graphs[id]
	if !ok {
		return nil, nil
	}

	return graph, nil
}

func (c *MemoryCache) StoreGraph(ctx context.Context, graph *Graph) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if graph.ID == "" {
		graph.ID = uuid.New().String()
	}
	graph.CreatedAt = time.Now()
	graph.UpdatedAt = graph.CreatedAt

	c.graphs[graph.ID] = graph
	return nil
}

func (c *MemoryCache) DeleteGraph(ctx context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.graphs, id)
	return nil
}

func (c *MemoryCache) GetRender(ctx context.Context, id string) (*Render, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	render, ok := c.renders[id]
	if !ok {
		return nil, nil
	}

	return render, nil
}

func (c *MemoryCache) GetRenderByGraphAndOptions(ctx context.Context, userID, graphHash string, layoutOpts LayoutOptions) (*Render, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, render := range c.renders {
		if render.UserID == userID &&
			render.GraphHash == graphHash &&
			render.LayoutOptions.VizType == layoutOpts.VizType &&
			render.LayoutOptions.Width == layoutOpts.Width &&
			render.LayoutOptions.Height == layoutOpts.Height &&
			render.LayoutOptions.Ordering == layoutOpts.Ordering &&
			render.LayoutOptions.Merge == layoutOpts.Merge &&
			render.LayoutOptions.Randomize == layoutOpts.Randomize &&
			render.LayoutOptions.Seed == layoutOpts.Seed {
			render.AccessedAt = time.Now()
			return render, nil
		}
	}

	return nil, nil
}

func (c *MemoryCache) StoreRender(ctx context.Context, render *Render) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if render.ID == "" {
		render.ID = uuid.New().String()
	}
	render.CreatedAt = time.Now()
	render.AccessedAt = render.CreatedAt

	c.renders[render.ID] = render
	return nil
}

func (c *MemoryCache) DeleteRender(ctx context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	render, ok := c.renders[id]
	if ok {
		// Delete associated artifacts
		if render.Artifacts.SVG != "" {
			delete(c.artifacts, render.Artifacts.SVG)
		}
		if render.Artifacts.PNG != "" {
			delete(c.artifacts, render.Artifacts.PNG)
		}
		if render.Artifacts.PDF != "" {
			delete(c.artifacts, render.Artifacts.PDF)
		}
	}

	delete(c.renders, id)
	return nil
}

func (c *MemoryCache) ListRenders(ctx context.Context, userID string, limit, offset int) ([]*Render, int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Filter by user
	var userRenders []*Render
	for _, r := range c.renders {
		if r.UserID == userID {
			userRenders = append(userRenders, r)
		}
	}

	// Sort by created_at desc
	sort.Slice(userRenders, func(i, j int) bool {
		return userRenders[i].CreatedAt.After(userRenders[j].CreatedAt)
	})

	total := int64(len(userRenders))

	// Apply pagination
	if offset >= len(userRenders) {
		return []*Render{}, total, nil
	}
	userRenders = userRenders[offset:]
	if limit < len(userRenders) {
		userRenders = userRenders[:limit]
	}

	return userRenders, total, nil
}

func (c *MemoryCache) StoreArtifact(ctx context.Context, renderID, filename string, data []byte) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	artifactID := uuid.New().String()
	c.artifacts[artifactID] = data

	return artifactID, nil
}

func (c *MemoryCache) GetArtifact(ctx context.Context, artifactID string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, ok := c.artifacts[artifactID]
	if !ok {
		return nil, fmt.Errorf("artifact %s: %w", artifactID, ErrNotFound)
	}

	return data, nil
}

func (c *MemoryCache) Close() error {
	return nil
}

// Ensure MemoryCache implements Cache
var _ Cache = (*MemoryCache)(nil)
