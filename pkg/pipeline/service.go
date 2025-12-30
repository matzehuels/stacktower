package pipeline

import (
	"context"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
)

// CacheBackend extends storage.Backend with direct access to Index and DocumentStore.
// This is implemented by DistributedBackend for API/Worker use.
type CacheBackend interface {
	storage.Backend
	Index() storage.Index
	DocumentStore() storage.DocumentStore
}

// Service wraps the pipeline with caching support.
// It provides a unified interface for CLI and API to execute pipeline stages
// with automatic caching through the configured backend.
type Service struct {
	backend      storage.Backend
	cacheBackend CacheBackend // Optional - only available for DistributedBackend
}

// NewService creates a new pipeline service.
// If backend is nil, caching is disabled (uses NullBackend).
func NewService(backend storage.Backend) *Service {
	if backend == nil {
		backend = storage.NullBackend{}
	}
	svc := &Service{backend: backend}
	// Check if backend supports extended cache operations
	if cb, ok := backend.(CacheBackend); ok {
		svc.cacheBackend = cb
	}
	return svc
}

// Close releases resources held by the service.
func (s *Service) Close() error {
	return s.backend.Close()
}

// =============================================================================
// Cache-Only Methods (for API fast-path without computation)
// =============================================================================

// GetCachedGraph checks if a graph is cached and returns it without computation.
// This is the API's "fast path" - return immediately if cached, otherwise queue a job.
//
// Returns:
//   - (graph, data, cacheKey, true) if found in cache
//   - (nil, nil, cacheKey, false) if not cached (cacheKey is still returned for job payload)
//   - (nil, nil, "", false) on validation error
func (s *Service) GetCachedGraph(ctx context.Context, opts Options) (*dag.DAG, []byte, string, bool) {
	if err := opts.ValidateForParse(); err != nil {
		return nil, nil, "", false
	}

	scope, pkgOrManifest, graphOpts := buildGraphKeyInputs(opts)
	hash := storage.Keys.GraphKey(scope, opts.UserID, opts.Language, pkgOrManifest, graphOpts)

	// Check cache only - don't compute
	var g *dag.DAG
	var hit bool
	var err error

	if scope == storage.ScopeUser && opts.UserID != "" {
		g, hit, err = s.backend.GetGraphScoped(ctx, hash, opts.UserID)
	} else {
		g, hit, err = s.backend.GetGraph(ctx, hash)
	}

	if err != nil || !hit {
		return nil, nil, hash, false
	}

	data, _ := serializeGraph(g)
	return g, data, hash, true
}

// Parse resolves dependencies with caching.
// Returns the graph, graph data (JSON), the cache key used, and whether it was a cache hit.
func (s *Service) Parse(ctx context.Context, opts Options) (*dag.DAG, []byte, string, bool, error) {
	if err := opts.ValidateForParse(); err != nil {
		return nil, nil, "", false, err
	}

	scope, pkgOrManifest, graphOpts := buildGraphKeyInputs(opts)
	hash := storage.Keys.GraphKey(scope, opts.UserID, opts.Language, pkgOrManifest, graphOpts)

	// Check cache (use scoped method for user data)
	if !opts.Refresh {
		var g *dag.DAG
		var hit bool
		var err error

		if scope == storage.ScopeUser && opts.UserID != "" {
			g, hit, err = s.backend.GetGraphScoped(ctx, hash, opts.UserID)
		} else {
			g, hit, err = s.backend.GetGraph(ctx, hash)
		}

		if err == nil && hit {
			data, _ := serializeGraph(g)
			return g, data, hash, true, nil
		}
	}

	// Parse
	g, err := Parse(ctx, s.backend, opts)
	if err != nil {
		return nil, nil, "", false, err
	}

	data, err := serializeGraph(g)
	if err != nil {
		return nil, nil, "", false, err
	}

	// Store in cache (use scoped method for user data)
	meta := storage.GraphMeta{
		Language: opts.Language,
		Package:  opts.Package,
		Options:  graphOpts, // Store options for cache key reconstruction
	}
	_ = s.backend.PutGraphScoped(ctx, hash, g, storage.GraphTTL, scope, opts.UserID, meta)

	return g, data, hash, false, nil
}

// Layout computes layout with caching.
// Returns layout data (JSON), the cache key used, and whether it was a cache hit.
// Layout cache is scoped the same way as the source graph to prevent data leakage.
func (s *Service) Layout(ctx context.Context, g *dag.DAG, opts Options) ([]byte, string, bool, error) {
	if err := opts.ValidateForLayout(); err != nil {
		return nil, "", false, err
	}

	scope := storage.DetermineScope(opts.Scope, opts.Manifest != "")
	graphHash := graphContentHash(g)
	layoutOpts := storage.LayoutOptions{
		VizType:   opts.VizType,
		Width:     opts.Width,
		Height:    opts.Height,
		Ordering:  opts.Ordering,
		Merge:     opts.Merge,
		Randomize: opts.Randomize,
		Seed:      opts.Seed,
	}
	hash := storage.Keys.LayoutCacheKey(scope, opts.UserID, graphHash, layoutOpts)

	// Check cache
	if data, hit, err := s.backend.GetLayout(ctx, hash); err == nil && hit {
		return data, hash, true, nil
	}

	// Compute layout
	layoutResult, err := ComputeLayout(g, opts)
	if err != nil {
		return nil, "", false, err
	}

	// Store in cache
	_ = s.backend.PutLayout(ctx, hash, layoutResult.LayoutData, storage.LayoutTTL, scope, opts.UserID)

	return layoutResult.LayoutData, hash, false, nil
}

// Visualize renders from layout with caching.
// Returns artifacts map and whether it was a cache hit.
// Artifact cache is scoped the same way as the source graph to prevent data leakage.
func (s *Service) Visualize(ctx context.Context, layoutData []byte, g *dag.DAG, opts Options) (map[string][]byte, bool, error) {
	if err := opts.ValidateForRender(); err != nil {
		return nil, false, err
	}

	scope := storage.DetermineScope(opts.Scope, opts.Manifest != "")

	// Build base cache key from layout content
	layoutHash := storage.Hash(layoutData)

	// Build render options hash for cache key differentiation
	renderOpts := storage.RenderOptions{
		Formats:   opts.Formats,
		Style:     opts.Style,
		ShowEdges: opts.ShowEdges,
		Popups:    opts.Popups,
	}
	optsHash := storage.OptionsHash(renderOpts)

	// Combine layout hash and options hash for artifact key
	combinedHash := layoutHash + ":" + optsHash

	// Check cache for each format
	allCached := true
	artifacts := make(map[string][]byte)

	for _, format := range opts.Formats {
		// Use scoped artifact cache key generation
		hash := storage.Keys.ArtifactCacheKey(scope, opts.UserID, combinedHash, format)

		if data, hit, err := s.backend.GetRender(ctx, hash, format); err == nil && hit {
			artifacts[format] = data
		} else {
			allCached = false
			break
		}
	}

	if allCached && len(artifacts) == len(opts.Formats) {
		return artifacts, true, nil
	}

	// Render all formats
	rendered, err := RenderFromLayoutData(layoutData, g, opts)
	if err != nil {
		return nil, false, err
	}

	// Store each format in cache using scoped key generation
	for format, data := range rendered {
		hash := storage.Keys.ArtifactCacheKey(scope, opts.UserID, combinedHash, format)
		_ = s.backend.PutRender(ctx, hash, format, data, storage.RenderTTL, scope, opts.UserID)
	}

	return rendered, false, nil
}

// Render runs layout + visualize with caching.
// Returns artifacts map, layout cache key, and whether it was a cache hit.
func (s *Service) Render(ctx context.Context, g *dag.DAG, opts Options) (map[string][]byte, string, bool, error) {
	if err := opts.ValidateForRender(); err != nil {
		return nil, "", false, err
	}

	// First, get or compute layout
	layoutData, layoutKey, layoutHit, err := s.Layout(ctx, g, opts)
	if err != nil {
		return nil, "", false, err
	}

	// Then, get or compute renders
	artifacts, renderHit, err := s.Visualize(ctx, layoutData, g, opts)
	if err != nil {
		return nil, "", false, err
	}

	return artifacts, layoutKey, layoutHit && renderHit, nil
}

// ExecuteFull runs the complete parse → layout → render pipeline with caching.
// This is a convenience method that combines Parse + Render.
// Returns the result struct, the graph cache key, and whether it was a cache hit.
func (s *Service) ExecuteFull(ctx context.Context, opts Options) (*Result, string, bool, error) {
	if err := opts.ValidateAndSetDefaults(); err != nil {
		return nil, "", false, err
	}

	// Parse
	g, graphData, graphKey, parseHit, err := s.Parse(ctx, opts)
	if err != nil {
		return nil, "", false, err
	}

	// Layout
	layoutData, _, layoutHit, err := s.Layout(ctx, g, opts)
	if err != nil {
		return nil, "", false, err
	}

	// Visualize
	artifacts, renderHit, err := s.Visualize(ctx, layoutData, g, opts)
	if err != nil {
		return nil, "", false, err
	}

	return &Result{
		Graph:      g,
		GraphHash:  storage.Hash(graphData),
		LayoutData: layoutData,
		Artifacts:  artifacts,
		Stats: Stats{
			NodeCount: g.NodeCount(),
			EdgeCount: g.EdgeCount(),
		},
	}, graphKey, parseHit && layoutHit && renderHit, nil
}

// =============================================================================
// Cache input types
// =============================================================================
//
// Note: Cache key generation has been consolidated into pkg/infra/storage/keys.go.
// The types below are kept for backward compatibility but new code should use
// storage.Keys.* methods directly.
//
// See storage.ParseInputs, storage.GraphOptions, storage.LayoutOptions, and
// storage.RenderOptions for the canonical type definitions.

// =============================================================================
// Helpers
// =============================================================================

// serializeGraph converts a DAG to JSON bytes.
func serializeGraph(g *dag.DAG) ([]byte, error) {
	return pkgio.SerializeDAG(g)
}

// graphContentHash computes a content hash for a DAG.
func graphContentHash(g *dag.DAG) string {
	data, _ := serializeGraph(g)
	return storage.Hash(data)
}

// buildGraphKeyInputs extracts the components needed for graph cache key generation.
// This is a helper to avoid duplicating scope/hash logic across methods.
func buildGraphKeyInputs(opts Options) (storage.Scope, string, storage.GraphOptions) {
	scope := storage.DetermineScope(opts.Scope, opts.Manifest != "")

	pkgOrManifest := opts.Package
	if opts.Manifest != "" {
		// Use FULL SHA-256 hash for manifests - user-controlled content needs collision resistance
		pkgOrManifest = storage.ManifestHash(opts.Manifest)
	}

	graphOpts := storage.GraphOptions{
		MaxDepth:  opts.MaxDepth,
		MaxNodes:  opts.MaxNodes,
		Normalize: opts.Normalize,
	}

	return scope, pkgOrManifest, graphOpts
}
