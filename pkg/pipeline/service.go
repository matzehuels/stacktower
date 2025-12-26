package pipeline

import (
	"bytes"
	"context"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/infra/artifact"
	"github.com/matzehuels/stacktower/pkg/infra/common"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
)

// Default TTLs for caching.
const (
	GraphTTL  = common.GraphTTL
	LayoutTTL = common.LayoutTTL
	RenderTTL = common.RenderTTL
)

// Service wraps the pipeline with caching support.
// It provides a unified interface for CLI and API to execute pipeline stages
// with automatic caching through the configured backend.
type Service struct {
	backend artifact.Backend
}

// NewService creates a new pipeline service.
// If backend is nil, caching is disabled (uses NullBackend).
func NewService(backend artifact.Backend) *Service {
	if backend == nil {
		backend = artifact.NullBackend{}
	}
	return &Service{backend: backend}
}

// Close releases resources held by the service.
func (s *Service) Close() error {
	return s.backend.Close()
}

// Parse resolves dependencies with caching.
// Returns the graph, graph data (JSON), and whether it was a cache hit.
func (s *Service) Parse(ctx context.Context, opts Options) (*dag.DAG, []byte, bool, error) {
	if err := opts.ValidateAndSetDefaults(); err != nil {
		return nil, nil, false, err
	}

	// Build cache key from inputs
	inputs := ParseInputs{
		Language:         opts.Language,
		Package:          opts.Package,
		ManifestHash:     artifact.Hash([]byte(opts.Manifest)),
		ManifestFilename: opts.ManifestFilename,
		MaxDepth:         opts.MaxDepth,
		MaxNodes:         opts.MaxNodes,
		Normalize:        opts.Normalize,
		Enrich:           opts.Enrich,
	}
	hash := artifact.HashJSON(inputs)

	// Check cache
	if !opts.Refresh {
		if g, hit, err := s.backend.GetGraph(ctx, hash); err == nil && hit {
			data, _ := serializeGraph(g)
			return g, data, true, nil
		}
	}

	// Parse
	g, err := Parse(ctx, s.backend, opts)
	if err != nil {
		return nil, nil, false, err
	}

	data, err := serializeGraph(g)
	if err != nil {
		return nil, nil, false, err
	}

	// Store in cache
	_ = s.backend.PutGraph(ctx, hash, g, GraphTTL)

	return g, data, false, nil
}

// Layout computes layout with caching.
// Returns layout data (JSON) and whether it was a cache hit.
func (s *Service) Layout(ctx context.Context, g *dag.DAG, opts Options) ([]byte, bool, error) {
	if err := opts.ValidateAndSetDefaults(); err != nil {
		return nil, false, err
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
	hash := artifact.HashJSON(inputs)

	// Check cache
	if data, hit, err := s.backend.GetLayout(ctx, hash); err == nil && hit {
		return data, true, nil
	}

	// Compute layout
	_, data, err := ComputeLayout(g, opts)
	if err != nil {
		return nil, false, err
	}

	// Store in cache
	_ = s.backend.PutLayout(ctx, hash, data, LayoutTTL)

	return data, false, nil
}

// Visualize renders from layout with caching.
// Returns artifacts map and whether it was a cache hit.
func (s *Service) Visualize(ctx context.Context, layoutData []byte, g *dag.DAG, opts Options) (map[string][]byte, bool, error) {
	if err := opts.ValidateAndSetDefaults(); err != nil {
		return nil, false, err
	}

	// Build cache key from inputs
	inputs := VisualizeInputs{
		LayoutHash: artifact.Hash(layoutData),
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
		hash := artifact.HashJSON(formatInputs)

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
		hash := artifact.HashJSON(formatInputs)
		_ = s.backend.PutRender(ctx, hash, format, data, RenderTTL)
	}

	return rendered, false, nil
}

// Render runs layout + visualize with caching.
// Returns artifacts map and whether it was a cache hit.
func (s *Service) Render(ctx context.Context, g *dag.DAG, opts Options) (map[string][]byte, bool, error) {
	if err := opts.ValidateAndSetDefaults(); err != nil {
		return nil, false, err
	}

	// First, get or compute layout
	layoutData, layoutHit, err := s.Layout(ctx, g, opts)
	if err != nil {
		return nil, false, err
	}

	// Then, get or compute renders
	artifacts, renderHit, err := s.Visualize(ctx, layoutData, g, opts)
	if err != nil {
		return nil, false, err
	}

	return artifacts, layoutHit && renderHit, nil
}

// ExecuteFull runs the complete parse → layout → render pipeline with caching.
// This is a convenience method that combines Parse + Render.
func (s *Service) ExecuteFull(ctx context.Context, opts Options) (*Result, bool, error) {
	if err := opts.ValidateAndSetDefaults(); err != nil {
		return nil, false, err
	}

	// Parse
	g, _, parseHit, err := s.Parse(ctx, opts)
	if err != nil {
		return nil, false, err
	}

	// Layout
	layoutData, layoutHit, err := s.Layout(ctx, g, opts)
	if err != nil {
		return nil, false, err
	}

	// Visualize
	artifacts, renderHit, err := s.Visualize(ctx, layoutData, g, opts)
	if err != nil {
		return nil, false, err
	}

	return &Result{
		Graph:      g,
		LayoutData: layoutData,
		Artifacts:  artifacts,
		Stats: Stats{
			NodeCount: g.NodeCount(),
			EdgeCount: g.EdgeCount(),
		},
	}, parseHit && layoutHit && renderHit, nil
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
	var buf bytes.Buffer
	if err := pkgio.WriteJSON(g, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// graphContentHash computes a content hash for a DAG.
func graphContentHash(g *dag.DAG) string {
	data, _ := serializeGraph(g)
	return artifact.Hash(data)
}
