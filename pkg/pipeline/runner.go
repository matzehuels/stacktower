package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"

	"github.com/matzehuels/stacktower/pkg/cache"
	"github.com/matzehuels/stacktower/pkg/core/dag"
	dagtransform "github.com/matzehuels/stacktower/pkg/core/dag/transform"
	"github.com/matzehuels/stacktower/pkg/graph"
)

// Runner encapsulates pipeline execution with caching.
// Both CLI and API can use this to avoid duplicating caching logic.
//
// The Runner is stateless except for the cache and logger - it doesn't
// store pipeline results. Multiple goroutines can safely use the same
// Runner with different options.
type Runner struct {
	Cache  cache.Cache
	Keyer  cache.Keyer
	Logger *log.Logger
}

// NewRunner creates a runner with the given cache and keyer.
// If keyer is nil, a DefaultKeyer is used.
// If cache is nil, a NullCache is used (caching disabled).
func NewRunner(c cache.Cache, keyer cache.Keyer, logger *log.Logger) *Runner {
	if keyer == nil {
		keyer = cache.NewDefaultKeyer()
	}
	if c == nil {
		c = cache.NewNullCache()
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Runner{
		Cache:  c,
		Keyer:  keyer,
		Logger: logger,
	}
}

// Execute runs the complete parse → layout → render pipeline with caching.
func (r *Runner) Execute(ctx context.Context, opts Options) (*Result, error) {
	if err := opts.ValidateAndSetDefaults(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}
	r.applyLogger(&opts)

	result := &Result{
		Artifacts: make(map[string][]byte),
	}

	// Stage 1: Parse
	parseStart := time.Now()
	g, parseHit, err := r.ParseWithCacheInfo(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	result.Graph = g
	result.Stats.ParseTime = time.Since(parseStart)
	result.Stats.NodeCount = g.NodeCount()
	result.Stats.EdgeCount = g.EdgeCount()
	result.CacheInfo.ParseHit = parseHit

	// Compute graph hash for cache keys and API responses
	if graphData, err := graph.MarshalGraph(g); err == nil {
		result.GraphHash = cache.Hash(graphData)
	}

	r.Logger.Info("parsed dependencies",
		"nodes", g.NodeCount(),
		"edges", g.EdgeCount(),
		"duration", result.Stats.ParseTime)

	// Apply normalization if requested (controlled by opts.Normalize)
	workGraph := r.PrepareGraph(g, opts)

	// Stage 2: Layout
	layoutStart := time.Now()
	layout, layoutHit, err := r.GenerateLayoutWithCacheInfo(ctx, workGraph, opts)
	if err != nil {
		return nil, fmt.Errorf("layout: %w", err)
	}
	result.Layout = layout
	result.Stats.LayoutTime = time.Since(layoutStart)
	result.CacheInfo.LayoutHit = layoutHit

	r.Logger.Info("computed layout",
		"blocks", len(layout.Blocks),
		"duration", result.Stats.LayoutTime)

	// Stage 3: Render
	renderStart := time.Now()
	artifacts, renderHit, err := r.RenderWithCacheInfo(ctx, layout, workGraph, opts)
	if err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}
	result.Artifacts = artifacts
	result.Stats.RenderTime = time.Since(renderStart)
	result.CacheInfo.RenderHit = renderHit

	r.Logger.Info("rendered outputs",
		"formats", opts.Formats,
		"duration", result.Stats.RenderTime)

	return result, nil
}

// ParseWithCacheInfo resolves dependencies with caching and returns cache hit info.
func (r *Runner) ParseWithCacheInfo(ctx context.Context, opts Options) (*dag.DAG, bool, error) {
	if err := opts.ValidateForParse(); err != nil {
		return nil, false, err
	}
	r.applyLogger(&opts)

	// Compute cache key
	pkgOrManifest := opts.Package
	if opts.Manifest != "" {
		pkgOrManifest = cache.Hash([]byte(opts.Manifest))
	}
	// Include enrichment status in cache key so enriched/non-enriched graphs are cached separately
	enriched := opts.ShouldEnrich() && opts.GitHubToken != ""
	cacheKey := r.Keyer.GraphKey(opts.Language, pkgOrManifest, cache.GraphKeyOpts{
		MaxDepth: opts.MaxDepth,
		MaxNodes: opts.MaxNodes,
		Enriched: enriched,
	})

	// Try cache first (unless refresh requested)
	if !opts.Refresh {
		if data, hit, err := r.Cache.Get(ctx, cacheKey); err == nil && hit {
			g, err := graph.ReadGraph(bytes.NewReader(data))
			if err == nil {
				return g, true, nil // Cache hit
			}
		}
	}

	// Parse
	g, err := Parse(ctx, r.Cache, opts)
	if err != nil {
		return nil, false, err
	}

	// Cache the result
	if !opts.Refresh {
		if data, err := graph.MarshalGraph(g); err == nil {
			_ = r.Cache.Set(ctx, cacheKey, data, cache.TTLGraph)
		}
	}

	return g, false, nil // Cache miss
}

// Parse is a convenience wrapper that calls ParseWithCacheInfo and discards the cache hit info.
func (r *Runner) Parse(ctx context.Context, opts Options) (*dag.DAG, error) {
	g, _, err := r.ParseWithCacheInfo(ctx, opts)
	return g, err
}

// GenerateLayoutWithCacheInfo generates a layout with caching and returns cache hit info.
func (r *Runner) GenerateLayoutWithCacheInfo(ctx context.Context, g *dag.DAG, opts Options) (graph.Layout, bool, error) {
	if err := opts.ValidateForLayout(); err != nil {
		return graph.Layout{}, false, err
	}
	r.applyLogger(&opts)

	// Compute cache key
	graphData, _ := graph.MarshalGraph(g)
	graphHash := cache.Hash(graphData)
	cacheKey := r.Keyer.LayoutKey(graphHash, opts.LayoutKeyOpts())

	// Try cache first
	if data, hit, err := r.Cache.Get(ctx, cacheKey); err == nil && hit {
		cached, err := graph.UnmarshalLayout(data)
		if err == nil {
			return cached, true, nil // Cache hit
		}
		// If deserialization fails, fall through to recompute
	}

	// Generate layout
	layout, err := GenerateLayout(g, opts)
	if err != nil {
		return graph.Layout{}, false, err
	}

	// Cache the result
	if data, err := graph.MarshalLayout(layout); err == nil {
		_ = r.Cache.Set(ctx, cacheKey, data, cache.TTLLayout)
	}

	return layout, false, nil // Cache miss
}

// GenerateLayout is a convenience wrapper that calls GenerateLayoutWithCacheInfo and discards the cache hit info.
func (r *Runner) GenerateLayout(ctx context.Context, g *dag.DAG, opts Options) (graph.Layout, error) {
	layout, _, err := r.GenerateLayoutWithCacheInfo(ctx, g, opts)
	return layout, err
}

// RenderWithCacheInfo generates artifacts with caching and returns cache hit info.
func (r *Runner) RenderWithCacheInfo(ctx context.Context, layout graph.Layout, g *dag.DAG, opts Options) (map[string][]byte, bool, error) {
	if err := opts.ValidateForRender(); err != nil {
		return nil, false, err
	}
	r.applyLogger(&opts)

	// Compute cache key from layout data
	layoutData, err := graph.MarshalLayout(layout)
	if err != nil {
		return nil, false, fmt.Errorf("serialize layout for cache key: %w", err)
	}
	cacheKeyHash := cache.Hash(layoutData)

	// Try to get all formats from cache
	allCached := true
	artifacts := make(map[string][]byte)

	for _, format := range opts.Formats {
		cacheKey := r.Keyer.ArtifactKey(cacheKeyHash, opts.ArtifactKeyOpts(format))
		if data, hit, err := r.Cache.Get(ctx, cacheKey); err == nil && hit {
			artifacts[format] = data
		} else {
			allCached = false
			break
		}
	}

	if allCached && len(artifacts) == len(opts.Formats) {
		return artifacts, true, nil // All artifacts from cache
	}

	// Render all formats
	rendered, err := RenderFromLayout(layout, g, opts)
	if err != nil {
		return nil, false, err
	}

	// Cache each format
	for format, data := range rendered {
		cacheKey := r.Keyer.ArtifactKey(cacheKeyHash, opts.ArtifactKeyOpts(format))
		_ = r.Cache.Set(ctx, cacheKey, data, cache.TTLArtifact)
	}

	return rendered, false, nil // Cache miss
}

// Render is a convenience wrapper that calls RenderWithCacheInfo and discards the cache hit info.
func (r *Runner) Render(ctx context.Context, layout graph.Layout, g *dag.DAG, opts Options) (map[string][]byte, error) {
	artifacts, _, err := r.RenderWithCacheInfo(ctx, layout, g, opts)
	return artifacts, err
}

// GenerateNodelinkLayout generates a nodelink layout from a graph.
//
// Deprecated: Use GenerateLayout instead.
func GenerateNodelinkLayout(g *dag.DAG, opts Options) (graph.Layout, error) {
	return generateNodelinkLayout(g, opts)
}

// PrepareGraph applies normalization if opts.Normalize is true.
// Returns the original graph if normalization is disabled.
func (r *Runner) PrepareGraph(g *dag.DAG, opts Options) *dag.DAG {
	if opts.Normalize {
		workGraph := g.Clone()
		dagtransform.Normalize(workGraph)
		r.Logger.Debug("normalized graph",
			"original_nodes", g.NodeCount(),
			"normalized_nodes", workGraph.NodeCount())
		return workGraph
	}
	return g
}

// Close releases resources held by the runner (primarily the cache).
func (r *Runner) Close() error {
	if r.Cache != nil {
		return r.Cache.Close()
	}
	return nil
}

// applyLogger sets the runner's logger on options if not already set.
func (r *Runner) applyLogger(opts *Options) {
	if opts.Logger == nil {
		opts.Logger = r.Logger
	}
}
