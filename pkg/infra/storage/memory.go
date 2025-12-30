package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
)

// MemoryBackend implements Backend, Index, DocumentStore, RateLimiter, and Cache for local development.
// In production, use DistributedBackend with Redis + MongoDB instead.
type MemoryBackend struct {
	mu sync.RWMutex

	// Index tier (Tier 1)
	graphEntries  map[string]*CacheEntry
	renderEntries map[string]*CacheEntry
	httpCache     map[string]*httpEntry

	// DocumentStore tier (Tier 2)
	graphs        map[string]*Graph
	renders       map[string]*Render
	artifactMetas map[string]*artifactMeta // artifact ID → data + ownership
	libraries     map[string]*LibraryEntry // "userID:lang:pkg" -> entry

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
		artifactMetas: make(map[string]*artifactMeta),
		libraries:     make(map[string]*LibraryEntry),
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

	// Deserialize - convert document back to JSON, then to DAG
	jsonData, err := json.Marshal(stored.Data)
	if err != nil {
		return nil, false, nil
	}
	g, err := pkgio.ReadJSON(bytes.NewReader(jsonData))
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

	// Deserialize - convert document to JSON-safe format, then to DAG
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

func (b *MemoryBackend) PutGraphScoped(ctx context.Context, hash string, g *dag.DAG, ttl time.Duration, scope Scope, userID string, meta GraphMeta) error {
	// Validate user-scoped graphs require userID
	if scope == ScopeUser && userID == "" {
		return ErrAccessDenied
	}

	// Serialize to JSON
	var buf bytes.Buffer
	if err := pkgio.WriteJSON(g, &buf); err != nil {
		return err
	}
	jsonData := buf.Bytes()

	// Unmarshal JSON to interface{} for BSON-style storage
	var dataDoc interface{}
	if err := json.Unmarshal(jsonData, &dataDoc); err != nil {
		return err
	}

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
		Options:     meta.Options, // Store options for cache key reconstruction
		ContentHash: Hash(jsonData),
		NodeCount:   g.NodeCount(),
		EdgeCount:   g.EdgeCount(),
		Data:        dataDoc, // Store as document
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
	// hash already includes scope prefix from Keys.LayoutCacheKey (e.g., "layout:global:..." or "layout:user:...")
	b.mu.RLock()
	defer b.mu.RUnlock()

	entry, ok := b.renderEntries[hash]
	if !ok || entry.IsExpired() {
		return nil, false, nil
	}

	meta, ok := b.artifactMetas[entry.DocumentID]
	if !ok {
		return nil, false, nil
	}

	return meta.Data, true, nil
}

func (b *MemoryBackend) PutLayout(ctx context.Context, hash string, data []byte, ttl time.Duration, scope Scope, userID string) error {
	// hash already includes scope prefix from Keys.LayoutCacheKey
	b.mu.Lock()
	defer b.mu.Unlock()

	// Store with ownership metadata - userID empty for global scope
	id := uuid.New().String()
	b.artifactMetas[id] = &artifactMeta{Data: data, UserID: userID}

	// Update index
	b.renderEntries[hash] = &CacheEntry{
		DocumentID: id,
		ExpiresAt:  time.Now().Add(ttl),
	}

	return nil
}

func (b *MemoryBackend) GetRender(ctx context.Context, hash, format string) ([]byte, bool, error) {
	// hash already includes scope and format prefix from Keys.ArtifactCacheKey
	// (e.g., "artifact:global:..." or "artifact:user:...")
	b.mu.RLock()
	defer b.mu.RUnlock()

	entry, ok := b.renderEntries[hash]
	if !ok || entry.IsExpired() {
		return nil, false, nil
	}

	meta, ok := b.artifactMetas[entry.DocumentID]
	if !ok {
		return nil, false, nil
	}

	return meta.Data, true, nil
}

func (b *MemoryBackend) PutRender(ctx context.Context, hash, format string, data []byte, ttl time.Duration, scope Scope, userID string) error {
	// hash already includes scope and format prefix from Keys.ArtifactCacheKey
	b.mu.Lock()
	defer b.mu.Unlock()

	// Store with ownership metadata - userID empty for global scope
	id := uuid.New().String()
	b.artifactMetas[id] = &artifactMeta{Data: data, UserID: userID}

	// Update index
	b.renderEntries[hash] = &CacheEntry{
		DocumentID: id,
		ExpiresAt:  time.Now().Add(ttl),
	}

	return nil
}

func (b *MemoryBackend) GetHTTP(ctx context.Context, namespace, key string) ([]byte, bool, error) {
	cacheKey := namespace + ":" + HashKey(key)
	b.mu.RLock()
	defer b.mu.RUnlock()

	entry, ok := b.httpCache[cacheKey]
	if !ok || time.Now().After(entry.ExpiresAt) {
		return nil, false, nil
	}
	return entry.Data, true, nil
}

func (b *MemoryBackend) SetHTTP(ctx context.Context, namespace, key string, data []byte, ttl time.Duration) error {
	cacheKey := namespace + ":" + HashKey(key)
	b.mu.Lock()
	defer b.mu.Unlock()

	b.httpCache[cacheKey] = &httpEntry{
		Data:      data,
		ExpiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (b *MemoryBackend) DeleteHTTP(ctx context.Context, namespace, key string) error {
	cacheKey := namespace + ":" + HashKey(key)
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

func (b *MemoryBackend) UpsertRenderDoc(ctx context.Context, render *Render) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if render.ID == "" {
		render.ID = uuid.New().String()
	}
	now := time.Now()
	render.AccessedAt = now

	// If exists, preserve CreatedAt; otherwise set it
	if existing, ok := b.renders[render.ID]; ok {
		render.CreatedAt = existing.CreatedAt
	} else {
		render.CreatedAt = now
	}

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
			delete(b.artifactMetas, render.Artifacts.SVG)
		}
		if render.Artifacts.PNG != "" {
			delete(b.artifactMetas, render.Artifacts.PNG)
		}
		if render.Artifacts.PDF != "" {
			delete(b.artifactMetas, render.Artifacts.PDF)
		}
	}

	delete(b.renders, id)
	return nil
}

// artifactMeta stores artifact data with ownership metadata.
type artifactMeta struct {
	Data   []byte
	UserID string // Empty string = shared/global artifact
}

func (b *MemoryBackend) StoreArtifact(ctx context.Context, renderID, filename string, data []byte, userID string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	artifactID := uuid.New().String()
	// Store with ownership metadata - userID may be empty for global artifacts
	b.artifactMetas[artifactID] = &artifactMeta{Data: data, UserID: userID}

	return artifactID, nil
}

func (b *MemoryBackend) GetArtifact(ctx context.Context, artifactID string) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	meta, ok := b.artifactMetas[artifactID]
	if !ok {
		return nil, fmt.Errorf("artifact %s: %w", artifactID, ErrNotFound)
	}

	return meta.Data, nil
}

// Ping checks if the backend is operational (always returns nil for in-memory).
func (b *MemoryBackend) Ping(ctx context.Context) error {
	return nil
}

func (b *MemoryBackend) CountUniqueTowers(ctx context.Context) (int64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	// Count unique (language, package) combinations
	towers := make(map[string]bool)
	for _, render := range b.renders {
		key := render.Source.Language + ":" + render.Source.Package
		towers[key] = true
	}
	return int64(len(towers)), nil
}

func (b *MemoryBackend) CountUniqueUsers(ctx context.Context) (int64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	users := make(map[string]bool)
	for _, render := range b.renders {
		users[render.UserID] = true
	}
	return int64(len(users)), nil
}

func (b *MemoryBackend) CountUniqueDependencies(ctx context.Context) (int64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	// Sum node counts from unique towers (distinct language+package)
	towers := make(map[string]int)
	for _, render := range b.renders {
		key := render.Source.Language + ":" + render.Source.Package
		// Take the max node_count for each unique tower
		if render.NodeCount > towers[key] {
			towers[key] = render.NodeCount
		}
	}
	var total int64
	for _, count := range towers {
		total += int64(count)
	}
	return total, nil
}

func (b *MemoryBackend) ListPackageSuggestions(ctx context.Context, language string, query string, limit int) ([]PackageSuggestion, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// Count users per (language, package) from libraries
	type pkgKey struct {
		language string
		pkg      string
	}
	counts := make(map[pkgKey]int)
	for _, entry := range b.libraries {
		if language != "" && entry.Language != language {
			continue
		}
		if query != "" && !strings.HasPrefix(strings.ToLower(entry.Package), strings.ToLower(query)) {
			continue
		}
		key := pkgKey{language: entry.Language, pkg: entry.Package}
		counts[key]++
	}

	// Convert to slice and sort by popularity
	suggestions := make([]PackageSuggestion, 0, len(counts))
	for key, count := range counts {
		suggestions = append(suggestions, PackageSuggestion{
			Language:   key.language,
			Package:    key.pkg,
			Popularity: count,
		})
	}

	// Sort by popularity descending
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Popularity > suggestions[j].Popularity
	})

	// Limit results
	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions, nil
}

// =============================================================================
// Explore
// =============================================================================

// ListExplore returns public towers for the explore page.
// sortBy: "popular" (default) or "recent"
func (b *MemoryBackend) ListExplore(ctx context.Context, language, sortBy string, limit, offset int) ([]ExploreEntry, int64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Group renders by (language, package)
	type groupKey struct {
		language string
		pkg      string
	}
	groups := make(map[groupKey]*ExploreEntry)

	for _, render := range b.renders {
		// Only include canonical renders (user_id = "") for packages
		if render.UserID != "" || render.Source.Type != "package" {
			continue
		}
		if language != "" && render.Source.Language != language {
			continue
		}

		key := groupKey{language: render.Source.Language, pkg: render.Source.Package}

		artifactSVG := ""
		if render.Artifacts.SVG != "" {
			artifactSVG = "/api/v1/artifacts/" + render.Artifacts.SVG
		}
		artifactPNG := ""
		if render.Artifacts.PNG != "" {
			artifactPNG = "/api/v1/artifacts/" + render.Artifacts.PNG
		}
		artifactPDF := ""
		if render.Artifacts.PDF != "" {
			artifactPDF = "/api/v1/artifacts/" + render.Artifacts.PDF
		}

		vizType := ExploreVizType{
			VizType:     render.LayoutOptions.VizType,
			RenderID:    render.ID,
			GraphID:     render.GraphID,
			ArtifactSVG: artifactSVG,
			ArtifactPNG: artifactPNG,
			ArtifactPDF: artifactPDF,
		}

		if existing, ok := groups[key]; ok {
			existing.VizTypes = append(existing.VizTypes, vizType)
			if render.CreatedAt.After(existing.CreatedAt) {
				existing.CreatedAt = render.CreatedAt
			}
			if render.NodeCount > existing.NodeCount {
				existing.NodeCount = render.NodeCount
			}
			if render.EdgeCount > existing.EdgeCount {
				existing.EdgeCount = render.EdgeCount
			}
		} else {
			groups[key] = &ExploreEntry{
				Source:    render.Source,
				NodeCount: render.NodeCount,
				EdgeCount: render.EdgeCount,
				CreatedAt: render.CreatedAt,
				VizTypes:  []ExploreVizType{vizType},
			}
		}
	}

	// Add popularity counts from libraries
	for key, entry := range groups {
		count := 0
		for _, lib := range b.libraries {
			if lib.Language == key.language && lib.Package == key.pkg {
				count++
			}
		}
		entry.PopularityCount = count
	}

	// Convert map to slice and sort
	entries := make([]ExploreEntry, 0, len(groups))
	for _, entry := range groups {
		entries = append(entries, *entry)
	}

	// Sort based on sortBy parameter
	if sortBy == "recent" {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].CreatedAt.After(entries[j].CreatedAt)
		})
	} else {
		// Default: sort by popularity, then by recent
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].PopularityCount != entries[j].PopularityCount {
				return entries[i].PopularityCount > entries[j].PopularityCount
			}
			return entries[i].CreatedAt.After(entries[j].CreatedAt)
		})
	}

	totalCount := int64(len(entries))

	// Apply pagination
	if offset >= len(entries) {
		return []ExploreEntry{}, totalCount, nil
	}
	entries = entries[offset:]
	if len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, totalCount, nil
}

// =============================================================================
// Canonical Renders & User Library
// =============================================================================

// libraryKey generates a unique key for the libraries map.
func libraryKey(userID, language, pkg string) string {
	return userID + ":" + language + ":" + pkg
}

// GetCanonicalRender looks up a canonical (shared) render for a public package.
func (b *MemoryBackend) GetCanonicalRender(ctx context.Context, language, pkg, vizType string) (*Render, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, r := range b.renders {
		if r.UserID == "" && r.Source.Type == "package" &&
			r.Source.Language == language && r.Source.Package == pkg &&
			r.LayoutOptions.VizType == vizType {
			return r, nil
		}
	}
	return nil, nil
}

// SaveToLibrary adds a package to a user's library.
func (b *MemoryBackend) SaveToLibrary(ctx context.Context, userID, language, pkg string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	key := libraryKey(userID, language, pkg)
	if _, ok := b.libraries[key]; ok {
		return nil // Already saved
	}

	b.libraries[key] = &LibraryEntry{
		ID:       uuid.New().String(),
		UserID:   userID,
		Language: language,
		Package:  pkg,
		SavedAt:  time.Now(),
	}
	return nil
}

// RemoveFromLibrary removes a package from a user's library.
func (b *MemoryBackend) RemoveFromLibrary(ctx context.Context, userID, language, pkg string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.libraries, libraryKey(userID, language, pkg))
	return nil
}

// IsInLibrary checks if a package is in a user's library.
func (b *MemoryBackend) IsInLibrary(ctx context.Context, userID, language, pkg string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	_, ok := b.libraries[libraryKey(userID, language, pkg)]
	return ok, nil
}

// ListLibrary returns a user's saved packages.
func (b *MemoryBackend) ListLibrary(ctx context.Context, userID string, limit, offset int) ([]LibraryEntry, int64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var entries []LibraryEntry
	for _, entry := range b.libraries {
		if entry.UserID == userID {
			entries = append(entries, *entry)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SavedAt.After(entries[j].SavedAt)
	})

	total := int64(len(entries))

	if offset >= len(entries) {
		return []LibraryEntry{}, total, nil
	}
	entries = entries[offset:]
	if limit > 0 && limit < len(entries) {
		entries = entries[:limit]
	}

	return entries, total, nil
}

// ListPrivateRenders returns a user's private repo renders.
func (b *MemoryBackend) ListPrivateRenders(ctx context.Context, userID string, limit, offset int) ([]*Render, int64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var renders []*Render
	for _, r := range b.renders {
		if r.UserID == userID && r.Source.Type == "manifest" {
			renders = append(renders, r)
		}
	}

	sort.Slice(renders, func(i, j int) bool {
		return renders[i].AccessedAt.After(renders[j].AccessedAt)
	})

	total := int64(len(renders))

	if offset >= len(renders) {
		return []*Render{}, total, nil
	}
	renders = renders[offset:]
	if limit > 0 && limit < len(renders) {
		renders = renders[:limit]
	}

	return renders, total, nil
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
		delete(b.artifactMetas, render.Artifacts.SVG)
	}
	if render.Artifacts.PNG != "" {
		delete(b.artifactMetas, render.Artifacts.PNG)
	}
	if render.Artifacts.PDF != "" {
		delete(b.artifactMetas, render.Artifacts.PDF)
	}

	delete(b.renders, id)
	return nil
}

func (b *MemoryBackend) GetArtifactScoped(ctx context.Context, artifactID string, userID string) ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	meta, ok := b.artifactMetas[artifactID]
	if !ok {
		return nil, fmt.Errorf("artifact %s: %w", artifactID, ErrNotFound)
	}

	// Authorization logic:
	// - Empty UserID in metadata → shared/global artifact, allow access
	// - Non-empty UserID → must match requesting user
	if meta.UserID != "" && meta.UserID != userID {
		return nil, ErrAccessDenied
	}

	return meta.Data, nil
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
	_ Backend       = (*MemoryBackend)(nil)
	_ Index         = (*MemoryBackend)(nil)
	_ DocumentStore = (*MemoryBackend)(nil)
	_ RateLimiter   = (*MemoryBackend)(nil)
	_ Cache         = (*MemoryBackend)(nil)
)
