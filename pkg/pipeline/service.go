package pipeline

import (
	"context"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
)

// Service wraps the pipeline with caching support.
// It provides a unified interface for CLI and API to execute pipeline stages
// with automatic caching through the configured backend.
type Service struct {
	backend storage.Backend
}

// NewService creates a new pipeline service.
// If backend is nil, caching is disabled (uses NullBackend).
func NewService(backend storage.Backend) *Service {
	if backend == nil {
		backend = storage.NullBackend{}
	}
	return &Service{backend: backend}
}

// Close releases resources held by the service.
func (s *Service) Close() error {
	return s.backend.Close()
}

// Parse resolves dependencies with caching.
// Returns the graph, graph data (JSON), the cache key used, and whether it was a cache hit.
func (s *Service) Parse(ctx context.Context, opts Options) (*dag.DAG, []byte, string, bool, error) {
	if err := opts.ValidateForParse(); err != nil {
		return nil, nil, "", false, err
	}

	// Determine scope from options (default to global for public packages)
	scope := opts.Scope
	if scope == "" {
		if opts.Manifest != "" {
			scope = storage.ScopeUser // Private manifests are user-scoped
		} else {
			scope = storage.ScopeGlobal // Public packages are global
		}
	}

	// Build cache key from inputs
	// For user-scoped data, include userID in the hash
	inputs := ParseInputs{
		Language:         opts.Language,
		Package:          opts.Package,
		ManifestHash:     storage.Hash([]byte(opts.Manifest)),
		ManifestFilename: opts.ManifestFilename,
		MaxDepth:         opts.MaxDepth,
		MaxNodes:         opts.MaxNodes,
		Normalize:        opts.Normalize,
		Enrich:           opts.Enrich,
	}

	// Include user ID in hash for user-scoped data
	hash := storage.HashJSON(inputs)
	if scope == storage.ScopeUser && opts.UserID != "" {
		hash = storage.HashJSON(struct {
			ParseInputs
			UserID string `json:"user_id"`
		}{inputs, opts.UserID})
	}

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
	}
	_ = s.backend.PutGraphScoped(ctx, hash, g, storage.GraphTTL, scope, opts.UserID, meta)

	return g, data, hash, false, nil
}

// Layout computes layout with caching.
// Returns layout data (JSON), the cache key used, and whether it was a cache hit.
func (s *Service) Layout(ctx context.Context, g *dag.DAG, opts Options) ([]byte, string, bool, error) {
	if err := opts.ValidateForLayout(); err != nil {
		return nil, "", false, err
	}

	// Build cache key from inputs
	inputs := LayoutInputs{
		GraphHash: graphContentHash(g),
		VizType:   opts.VizType,
		Width:     opts.Width,
		Height:    opts.Height,
		Ordering:  opts.Ordering,
		Merge:     opts.Merge,
		Randomize: opts.Randomize,
		Seed:      opts.Seed,
		Nebraska:  opts.Nebraska,
	}
	hash := storage.HashJSON(inputs)

	// Check cache
	if data, hit, err := s.backend.GetLayout(ctx, hash); err == nil && hit {
		return data, hash, true, nil
	}

	// Compute layout
	_, data, err := ComputeLayout(g, opts)
	if err != nil {
		return nil, "", false, err
	}

	// Store in cache
	_ = s.backend.PutLayout(ctx, hash, data, storage.LayoutTTL)

	return data, hash, false, nil
}

// Visualize renders from layout with caching.
// Returns artifacts map and whether it was a cache hit.
// Note: Visualize doesn't strictly need to return a single cache key as it handles multiple formats,
// but for consistency/pipeline usage, we can return the base key or inputs hash.
func (s *Service) Visualize(ctx context.Context, layoutData []byte, g *dag.DAG, opts Options) (map[string][]byte, bool, error) {
	if err := opts.ValidateForRender(); err != nil {
		return nil, false, err
	}

	// Build cache key from inputs
	inputs := VisualizeInputs{
		LayoutHash: storage.Hash(layoutData),
		Formats:    opts.Formats,
		Style:      opts.Style,
		ShowEdges:  opts.ShowEdges,
		Popups:     opts.Popups,
	}

	// Check cache for each format
	allCached := true
	artifacts := make(map[string][]byte)

	for _, format := range opts.Formats {
		formatInputs := inputs
		formatInputs.Formats = []string{format}
		hash := storage.HashJSON(formatInputs)

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

	// Store each format in cache
	for format, data := range rendered {
		formatInputs := inputs
		formatInputs.Formats = []string{format}
		hash := storage.HashJSON(formatInputs)
		_ = s.backend.PutRender(ctx, hash, format, data, storage.RenderTTL)
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
	g, _, graphKey, parseHit, err := s.Parse(ctx, opts)
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
		LayoutData: layoutData,
		Artifacts:  artifacts,
		Stats: Stats{
			NodeCount: g.NodeCount(),
			EdgeCount: g.EdgeCount(),
		},
	}, graphKey, parseHit && layoutHit && renderHit, nil
}

// =============================================================================
// Cache input types (for generating cache keys)
// =============================================================================

// ParseInputs defines the inputs that affect graph parsing.
type ParseInputs struct {
	Language         string `json:"language"`
	Package          string `json:"package,omitempty"`
	ManifestHash     string `json:"manifest_hash,omitempty"`
	ManifestFilename string `json:"manifest_filename,omitempty"`
	MaxDepth         int    `json:"max_depth"`
	MaxNodes         int    `json:"max_nodes"`
	Normalize        bool   `json:"normalize"`
	Enrich           bool   `json:"enrich"`
}

// LayoutInputs defines the inputs that affect layout computation.
type LayoutInputs struct {
	GraphHash string  `json:"graph_hash"`
	VizType   string  `json:"viz_type"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Ordering  string  `json:"ordering,omitempty"`
	Merge     bool    `json:"merge,omitempty"`
	Randomize bool    `json:"randomize,omitempty"`
	Seed      uint64  `json:"seed,omitempty"`
	Nebraska  bool    `json:"nebraska,omitempty"`
}

// VisualizeInputs defines the inputs that affect visualization.
type VisualizeInputs struct {
	LayoutHash string   `json:"layout_hash"`
	Formats    []string `json:"formats"`
	Style      string   `json:"style,omitempty"`
	ShowEdges  bool     `json:"show_edges,omitempty"`
	Popups     bool     `json:"popups,omitempty"`
}

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
