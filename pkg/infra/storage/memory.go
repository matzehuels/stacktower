package storage

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
)

// MemoryBackend implements Backend, Index, DocumentStore, OperationStore, RateLimiter, and Cache for local development.
// In production, use DistributedBackend with Redis + MongoDB instead.
type MemoryBackend struct {
	mu sync.RWMutex

	// Index tier (Tier 1)
	graphEntries  map[string]*CacheEntry
	renderEntries map[string]*CacheEntry
	httpCache     map[string]*httpEntry

	// DocumentStore tier (Tier 2)
	graphs    map[string]*Graph
	renders   map[string]*Render
	artifacts map[string][]byte

	// Operation log
	operations []*Operation

	// Rate limiting (in-memory sliding window)
	rateLimits   map[string]*rateLimitEntry // userID:opType -> entry
	storageUsage map[string]int64           // userID -> bytes used
}

// httpEntry stores an HTTP response with expiration.
type httpEntry struct {
	Data      []byte
	ExpiresAt time.Time
}

// rateLimitEntry tracks operation counts for rate limiting.
type rateLimitEntry struct {
	Count     int
	WindowEnd time.Time
}

// NewMemoryBackend creates a new in-memory backend for local development and testing.
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		graphEntries:  make(map[string]*CacheEntry),
		renderEntries: make(map[string]*CacheEntry),
		httpCache:     make(map[string]*httpEntry),
		graphs:        make(map[string]*Graph),
		renders:       make(map[string]*Render),
		artifacts:     make(map[string][]byte),
		operations:    make([]*Operation, 0),
		rateLimits:    make(map[string]*rateLimitEntry),
		storageUsage:  make(map[string]int64),
	}
}

// =============================================================================
// Backend interface implementation (primary interface)
// =============================================================================

func (b *MemoryBackend) GetGraph(ctx context.Context, hash string) (*dag.DAG, bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check index
	entry, ok := b.graphEntries[hash]
	if !ok || entry.IsExpired() {
		return nil, false, nil
	}

	// Fetch from store
	stored, ok := b.graphs[entry.DocumentID]
	if !ok {
		return nil, false, nil
	}

	// Deserialize
	g, err := pkgio.ReadJSON(bytes.NewReader(stored.Data))
	if err != nil {
		return nil, false, nil
	}

	return g, true, nil
}

func (b *MemoryBackend) PutGraph(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration) error {
	return b.PutGraphScoped(ctx, hash, g, ttl, ScopeGlobal, "", GraphMeta{})
}

func (b *MemoryBackend) GetGraphScoped(ctx context.Context, hash string, userID string) (*dag.DAG, bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Check index
	entry, ok := b.graphEntries[hash]
	if !ok || entry.IsExpired() {
		return nil, false, nil
	}

	// Fetch from store
	stored, ok := b.graphs[entry.DocumentID]
	if !ok {
		return nil, false, nil
	}

	// Check authorization for user-scoped graphs
	if stored.Scope == ScopeUser && stored.UserID != userID {
		return nil, false, nil // Treat as cache miss
	}

	// Deserialize
	g, err := pkgio.ReadJSON(bytes.NewReader(stored.Data))
	if err != nil {
		return nil, false, nil
	}

	return g, true, nil
}

func (b *MemoryBackend) PutGraphScoped(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration, scope Scope, userID string, meta GraphMeta) error {
	// Validate user-scoped graphs require userID
	if scope == ScopeUser && userID == "" {
		return ErrAccessDenied
	}

	// Serialize
	var buf bytes.Buffer
	if err := pkgio.WriteJSON(g, &buf); err != nil {
		return err
	}
	data := buf.Bytes()

	b.mu.Lock()
	defer b.mu.Unlock()

	// Store in DocumentStore
	id := uuid.New().String()
	b.graphs[id] = &Graph{
		ID:          id,
		Scope:       scope,
		UserID:      userID,
		Language:    meta.Language,
		Package:     meta.Package,
		Repo:        meta.Repo,
		ContentHash: Hash(data),
		NodeCount:   g.NodeCount(),
		EdgeCount:   g.EdgeCount(),
		Data:        data,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Update Index
	b.graphEntries[hash] = &CacheEntry{
		DocumentID: id,
		ExpiresAt:  time.Now().Add(ttl),
	}

	return nil
}

func (b *MemoryBackend) GetLayout(ctx context.Context, hash string) ([]byte, bool, error) {
	layoutKey := "layout:" + hash
	b.mu.RLock()
	defer b.mu.RUnlock()

	entry, ok := b.renderEntries[layoutKey]
	if !ok || entry.IsExpired() {
		return nil, false, nil
	}

	data, ok := b.artifacts[entry.DocumentID]
	if !ok {
		return nil, false, nil
	}

	return data, true, nil
}

func (b *MemoryBackend) PutLayout(ctx context.Context, hash string, data []byte, ttl time.Duration) error {
	layoutKey := "layout:" + hash
	b.mu.Lock()
	defer b.mu.Unlock()

	// Store as artifact
	id := uuid.New().String()
	b.artifacts[id] = data

	// Update index
	b.renderEntries[layoutKey] = &CacheEntry{
		DocumentID: id,
		ExpiresAt:  time.Now().Add(ttl),
	}

	return nil
}

func (b *MemoryBackend) GetRender(ctx context.Context, hash, format string) ([]byte, bool, error) {
	renderKey := "render:" + hash + ":" + format
	b.mu.RLock()
	defer b.mu.RUnlock()

	entry, ok := b.renderEntries[renderKey]
	if !ok || entry.IsExpired() {
		return nil, false, nil
	}

	data, ok := b.artifacts[entry.DocumentID]
	if !ok {
		return nil, false, nil
	}

	return data, true, nil
}

func (b *MemoryBackend) PutRender(ctx context.Context, hash, format string, data []byte, ttl time.Duration) error {
	renderKey := "render:" + hash + ":" + format
	b.mu.Lock()
	defer b.mu.Unlock()

	// Store as artifact
	id := uuid.New().String()
	b.artifacts[id] = data

	// Update index
	b.renderEntries[renderKey] = &CacheEntry{
		DocumentID: id,
		ExpiresAt:  time.Now().Add(ttl),
	}

	return nil
}

func (b *MemoryBackend) GetHTTP(ctx context.Context, namespace, key string) ([]byte, bool, error) {
	cacheKey := namespace + HashKey(key)
	b.mu.RLock()
	defer b.mu.RUnlock()

	entry, ok := b.httpCache[cacheKey]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return nil, false, nil
	}
	return entry.Data, true, nil
}

func (b *MemoryBackend) SetHTTP(ctx context.Context, namespace, key string, data []byte, ttl time.Duration) error {
	cacheKey := namespace + HashKey(key)
	b.mu.Lock()
	defer b.mu.Unlock()

	b.httpCache[cacheKey] = &httpEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (b *MemoryBackend) DeleteHTTP(ctx context.Context, namespace, key string) error {
	cacheKey := namespace + HashKey(key)
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.httpCache, cacheKey)
	return nil
}

// =============================================================================
// Index interface implementation (Tier 1)
// =============================================================================

func (b *MemoryBackend) GetGraphEntry(ctx context.Context, key string) (*CacheEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	entry, ok := b.graphEntries[key]
	if !ok || entry.IsExpired() {
		return nil, nil
	}
	return entry, nil
}

func (b *MemoryBackend) SetGraphEntry(ctx context.Context, key string, entry *CacheEntry) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.graphEntries[key] = entry
	return nil
}

func (b *MemoryBackend) GetRenderEntry(ctx context.Context, key string) (*CacheEntry, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	entry, ok := b.renderEntries[key]
	if !ok || entry.IsExpired() {
		return nil, nil
	}
	return entry, nil
}

func (b *MemoryBackend) SetRenderEntry(ctx context.Context, key string, entry *CacheEntry) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.renderEntries[key] = entry
	return nil
}

// =============================================================================
// DocumentStore interface implementation (Tier 2)
// =============================================================================

// GetGraphDoc retrieves a graph document by ID.
func (b *MemoryBackend) GetGraphDoc(ctx context.Context, id string) (*Graph, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	graph, ok := b.graphs[id]
	if !ok {
		return nil, nil
	}
	return graph, nil
}

func (b *MemoryBackend) StoreGraphDoc(ctx context.Context, graph *Graph) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if graph.ID == "" {
		graph.ID = uuid.New().String()
	}
	graph.CreatedAt = time.Now()
	graph.UpdatedAt = graph.CreatedAt

	b.graphs[graph.ID] = graph
	return nil
}

func (b *MemoryBackend) GetRenderDoc(ctx context.Context, id string) (*Render, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	render, ok := b.renders[id]
	if !ok {
		return nil, nil
	}
	return render, nil
}

func (b *MemoryBackend) StoreRenderDoc(ctx context.Context, render *Render) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if render.ID == "" {
		render.ID = uuid.New().String()
	}
	render.CreatedAt = time.Now()
	render.AccessedAt = render.CreatedAt

	b.renders[render.ID] = render
	return nil
}

func (b *MemoryBackend) DeleteRenderDoc(ctx context.Context, id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	render, ok := b.renders[id]
	if ok {
		// Delete associated artifacts
		if render.Artifacts.SVG != "" {
			delete(b.artifacts, render.Artifacts.SVG)
		}
		if render.Artifacts.PNG != "" {
			delete(b.artifacts, render.Artifacts.PNG)
		}
		if render.Artifacts.PDF != "" {
			delete(b.artifacts, render.Artifacts.PDF)
		}
	}

	delete(b.renders, id)
	return nil
}

func (b *MemoryBackend) ListRenderDocs(ctx context.Context, userID string, limit, offset int) ([]*Render, int64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Filter by user
	var userRenders []*Render
	for _, r := range b.renders {
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

func (b *MemoryBackend) StoreArtifact(ctx context.Context, renderID, filename string, data []byte) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	artifactID := uuid.New().String()
	b.artifacts[artifactID] = data

	return artifactID, nil
}

func (b *MemoryBackend) GetArtifact(ctx context.Context, artifactID string) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	data, ok := b.artifacts[artifactID]
	if !ok {
		return nil, fmt.Errorf("artifact %s: %w", artifactID, ErrNotFound)
	}

	return data, nil
}

// Ping checks if the backend is operational (always returns nil for in-memory).
func (b *MemoryBackend) Ping(ctx context.Context) error {
	return nil
}

func (b *MemoryBackend) Close() error {
	return nil
}

// =============================================================================
// DocumentStore scoped methods
// =============================================================================

func (b *MemoryBackend) GetGraphDocScoped(ctx context.Context, id string, userID string) (*Graph, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	graph, ok := b.graphs[id]
	if !ok {
		return nil, nil
	}

	// Check authorization for user-scoped graphs
	if graph.Scope == ScopeUser && graph.UserID != userID {
		return nil, ErrAccessDenied
	}

	return graph, nil
}

func (b *MemoryBackend) GetRenderDocScoped(ctx context.Context, id string, userID string) (*Render, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	render, ok := b.renders[id]
	if !ok {
		return nil, nil
	}

	// Renders are always user-scoped
	if render.UserID != userID {
		return nil, ErrAccessDenied
	}

	return render, nil
}

func (b *MemoryBackend) DeleteRenderDocScoped(ctx context.Context, id string, userID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	render, ok := b.renders[id]
	if !ok {
		return nil
	}

	// Check ownership
	if render.UserID != userID {
		return ErrAccessDenied
	}

	// Delete associated artifacts
	if render.Artifacts.SVG != "" {
		delete(b.artifacts, render.Artifacts.SVG)
	}
	if render.Artifacts.PNG != "" {
		delete(b.artifacts, render.Artifacts.PNG)
	}
	if render.Artifacts.PDF != "" {
		delete(b.artifacts, render.Artifacts.PDF)
	}

	delete(b.renders, id)
	return nil
}

func (b *MemoryBackend) GetArtifactScoped(ctx context.Context, artifactID string, userID string) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Find the render that owns this artifact
	var ownerRender *Render
	for _, r := range b.renders {
		if r.Artifacts.SVG == artifactID || r.Artifacts.PNG == artifactID || r.Artifacts.PDF == artifactID {
			ownerRender = r
			break
		}
	}

	if ownerRender == nil {
		// Artifact not associated with any render - allow access (might be shared)
		data, ok := b.artifacts[artifactID]
		if !ok {
			return nil, fmt.Errorf("artifact %s: %w", artifactID, ErrNotFound)
		}
		return data, nil
	}

	// Check ownership
	if ownerRender.UserID != userID {
		return nil, ErrAccessDenied
	}

	data, ok := b.artifacts[artifactID]
	if !ok {
		return nil, fmt.Errorf("artifact %s: %w", artifactID, ErrNotFound)
	}
	return data, nil
}

// =============================================================================
// OperationStore implementation
// =============================================================================

func (b *MemoryBackend) RecordOperation(ctx context.Context, op *Operation) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if op.ID == "" {
		op.ID = uuid.New().String()
	}
	if op.CreatedAt.IsZero() {
		op.CreatedAt = time.Now()
	}

	b.operations = append(b.operations, op)
	return nil
}

func (b *MemoryBackend) ListOperations(ctx context.Context, userID string, opType OperationType, limit, offset int) ([]*Operation, int64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Filter operations
	var filtered []*Operation
	for _, op := range b.operations {
		if op.UserID == userID && (opType == "" || op.Type == opType) {
			filtered = append(filtered, op)
		}
	}

	// Sort by created_at desc (newest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	total := int64(len(filtered))

	// Apply pagination
	if offset >= len(filtered) {
		return []*Operation{}, total, nil
	}
	filtered = filtered[offset:]
	if limit < len(filtered) {
		filtered = filtered[:limit]
	}

	return filtered, total, nil
}

func (b *MemoryBackend) CountOperationsInWindow(ctx context.Context, userID string, opType OperationType, windowStart int64) (int64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	windowTime := time.Unix(windowStart, 0)
	var count int64
	for _, op := range b.operations {
		if op.UserID == userID && op.Type == opType && op.CreatedAt.After(windowTime) {
			count++
		}
	}
	return count, nil
}

func (b *MemoryBackend) GetOperationStats(ctx context.Context, userID string) (*UserOperationStats, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := &UserOperationStats{}
	for _, op := range b.operations {
		if op.UserID != userID {
			continue
		}
		stats.TotalOperations++
		switch op.Type {
		case OpTypeParse:
			stats.TotalParses++
		case OpTypeLayout:
			stats.TotalLayouts++
		case OpTypeRender:
			stats.TotalRenders++
		}
		if op.Stats.CacheHit {
			stats.TotalCacheHits++
		}
	}
	stats.StorageBytesUsed = b.storageUsage[userID]
	return stats, nil
}

// =============================================================================
// RateLimiter implementation
// =============================================================================

func (b *MemoryBackend) CheckRateLimit(ctx context.Context, userID string, opType OperationType, quota QuotaConfig) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	key := userID + ":" + string(opType)
	entry, ok := b.rateLimits[key]
	if !ok || time.Now().After(entry.WindowEnd) {
		return nil // No limit or window expired
	}

	var limit int
	switch opType {
	case OpTypeParse:
		limit = quota.MaxParsesPerHour
	case OpTypeLayout:
		limit = quota.MaxLayoutsPerHour
	case OpTypeRender:
		limit = quota.MaxRendersPerHour
	}

	if entry.Count >= limit {
		return ErrRateLimited
	}
	return nil
}

func (b *MemoryBackend) IncrementRateLimit(ctx context.Context, userID string, opType OperationType) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := userID + ":" + string(opType)
	entry, ok := b.rateLimits[key]
	if !ok || time.Now().After(entry.WindowEnd) {
		// Start new window
		b.rateLimits[key] = &rateLimitEntry{
			Count:     1,
			WindowEnd: time.Now().Add(time.Hour),
		}
		return nil
	}

	entry.Count++
	return nil
}

func (b *MemoryBackend) GetRateLimitStatus(ctx context.Context, userID string, quota QuotaConfig) (*RateLimitStatus, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	now := time.Now()
	windowEnd := now.Add(time.Hour).Unix()

	getCount := func(opType OperationType) int {
		key := userID + ":" + string(opType)
		entry, ok := b.rateLimits[key]
		if !ok || now.After(entry.WindowEnd) {
			return 0
		}
		return entry.Count
	}

	return &RateLimitStatus{
		ParsesUsed:        getCount(OpTypeParse),
		ParsesLimit:       quota.MaxParsesPerHour,
		LayoutsUsed:       getCount(OpTypeLayout),
		LayoutsLimit:      quota.MaxLayoutsPerHour,
		RendersUsed:       getCount(OpTypeRender),
		RendersLimit:      quota.MaxRendersPerHour,
		StorageBytesUsed:  b.storageUsage[userID],
		StorageBytesLimit: quota.MaxStorageBytes,
		WindowResetAt:     windowEnd,
	}, nil
}

func (b *MemoryBackend) CheckStorageQuota(ctx context.Context, userID string, bytesToAdd int64, quota QuotaConfig) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	current := b.storageUsage[userID]
	if current+bytesToAdd > quota.MaxStorageBytes {
		return ErrQuotaExceeded
	}
	return nil
}

func (b *MemoryBackend) UpdateStorageUsage(ctx context.Context, userID string, byteDelta int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.storageUsage[userID] += byteDelta
	if b.storageUsage[userID] < 0 {
		b.storageUsage[userID] = 0
	}
	return nil
}

// Ensure MemoryBackend implements all interfaces
var (
	_ Backend        = (*MemoryBackend)(nil)
	_ Index          = (*MemoryBackend)(nil)
	_ DocumentStore  = (*MemoryBackend)(nil)
	_ OperationStore = (*MemoryBackend)(nil)
	_ RateLimiter    = (*MemoryBackend)(nil)
	_ Cache          = (*MemoryBackend)(nil)
)
