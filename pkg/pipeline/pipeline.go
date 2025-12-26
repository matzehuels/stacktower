// Package pipeline provides the core visualization pipeline for Stacktower.
//
// This package implements the complete parse → layout → render pipeline that
// can be used by CLI, API, and worker components. By centralizing this logic,
// we ensure consistent behavior across all entry points and avoid code duplication.
//
// # Architecture
//
// The pipeline consists of three stages:
//
//  1. Parse: Resolve dependencies from package registries or manifest files
//  2. Layout: Compute visual positions for the dependency graph
//  3. Render: Generate output in various formats (SVG, PNG, PDF, JSON)
//
// Each stage can be run independently or as part of the complete pipeline.
//
// # Usage
//
// Run the complete pipeline:
//
//	opts := pipeline.Options{
//	    Language: "python",
//	    Package:  "requests",
//	    VizType:  "tower",
//	    Formats:  []string{"svg"},
//	}
//	result, err := pipeline.Execute(ctx, opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	svg := result.Artifacts["svg"]
//
// Run individual stages:
//
//	// Parse only
//	g, err := pipeline.Parse(ctx, parseOpts)
//
//	// Layout with existing graph
//	layout, err := pipeline.Layout(g, layoutOpts)
//
//	// Render with existing layout
//	artifacts, err := pipeline.Render(layout, g, renderOpts)
package pipeline

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/log"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
	"github.com/matzehuels/stacktower/pkg/render/tower/ordering"
)

// Options contains all configuration for the visualization pipeline.
type Options struct {
	// Parse options
	Language         string // Package ecosystem (required)
	Package          string // Package name (required unless Manifest is provided)
	Manifest         string // Raw manifest content
	ManifestFilename string // Filename for manifest parsing
	MaxDepth         int    // Max dependency depth (default: 10)
	MaxNodes         int    // Max nodes to fetch (default: 5000)
	Enrich           bool   // Fetch GitHub metadata
	Refresh          bool   // Bypass cache
	Normalize        bool   // Apply DAG normalization

	// Layout options
	VizType   string  // "tower" or "nodelink" (default: "tower")
	Width     float64 // Frame width (default: 800)
	Height    float64 // Frame height (default: 600)
	Ordering  string  // Ordering algorithm: "optimal", "barycentric"
	Merge     bool    // Merge subdivider blocks
	Randomize bool    // Randomize block widths
	Seed      uint64  // Random seed (default: 42)

	// Render options
	Formats   []string // Output formats: "svg", "png", "pdf", "json"
	Style     string   // Visual style: "simple", "handdrawn" (default)
	ShowEdges bool     // Draw dependency edges
	Nebraska  bool     // Show Nebraska maintainer ranking
	Popups    bool     // Enable hover popups

	// Runtime options
	Logger      *log.Logger      // Logger for progress messages
	GitHubToken string           // Token for metadata enrichment
	Orderer     ordering.Orderer // Custom ordering algorithm
}

// Result contains the outputs of a pipeline run.
type Result struct {
	// Graph is the parsed dependency graph.
	Graph *dag.DAG

	// Layout contains the computed visual positions.
	Layout layout.Layout

	// LayoutData is the serialized layout (JSON).
	LayoutData []byte

	// Artifacts contains rendered outputs keyed by format.
	Artifacts map[string][]byte

	// Stats contains timing and size information.
	Stats Stats
}

// Stats contains pipeline execution statistics.
type Stats struct {
	NodeCount  int
	EdgeCount  int
	ParseTime  time.Duration
	LayoutTime time.Duration
	RenderTime time.Duration
}

// ValidateAndSetDefaults checks required fields and applies defaults.
func (o *Options) ValidateAndSetDefaults() error {
	if o.Language == "" {
		return fmt.Errorf("language is required")
	}
	if o.Package == "" && o.Manifest == "" {
		return fmt.Errorf("package or manifest is required")
	}
	if o.Manifest != "" && o.ManifestFilename == "" {
		return fmt.Errorf("manifest_filename is required when manifest is provided")
	}

	// Parse defaults
	if o.MaxDepth == 0 {
		o.MaxDepth = 10
	}
	if o.MaxNodes == 0 {
		o.MaxNodes = 5000
	}

	// Layout defaults
	if o.VizType == "" {
		o.VizType = "tower"
	}
	if o.Width == 0 {
		o.Width = 800
	}
	if o.Height == 0 {
		o.Height = 600
	}
	if o.Seed == 0 {
		o.Seed = 42
	}

	// Render defaults
	if len(o.Formats) == 0 {
		o.Formats = []string{"svg"}
	}
	if o.Style == "" {
		o.Style = "handdrawn"
	}

	// Logger default
	if o.Logger == nil {
		o.Logger = log.NewWithOptions(io.Discard, log.Options{})
	}

	return nil
}

// IsTower returns true if this is a tower visualization.
func (o *Options) IsTower() bool {
	return o.VizType == "" || o.VizType == "tower"
}

// IsNodelink returns true if this is a nodelink visualization.
func (o *Options) IsNodelink() bool {
	return o.VizType == "nodelink"
}

// Execute runs the complete parse → layout → render pipeline.
func Execute(ctx context.Context, opts Options) (*Result, error) {
	if err := opts.ValidateAndSetDefaults(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	result := &Result{
		Artifacts: make(map[string][]byte),
	}

	// Stage 1: Parse
	parseStart := time.Now()
	g, err := Parse(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	result.Graph = g
	result.Stats.ParseTime = time.Since(parseStart)
	result.Stats.NodeCount = g.NodeCount()
	result.Stats.EdgeCount = g.EdgeCount()

	opts.Logger.Info("parsed dependencies",
		"nodes", g.NodeCount(),
		"edges", g.EdgeCount(),
		"duration", result.Stats.ParseTime)

	// Stage 2: Layout
	layoutStart := time.Now()
	l, layoutData, err := ComputeLayout(g, opts)
	if err != nil {
		return nil, fmt.Errorf("layout: %w", err)
	}
	result.Layout = l
	result.LayoutData = layoutData
	result.Stats.LayoutTime = time.Since(layoutStart)

	opts.Logger.Info("computed layout",
		"blocks", len(l.Blocks),
		"duration", result.Stats.LayoutTime)

	// Stage 3: Render
	renderStart := time.Now()
	artifacts, err := Render(l, layoutData, g, opts)
	if err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}
	result.Artifacts = artifacts
	result.Stats.RenderTime = time.Since(renderStart)

	opts.Logger.Info("rendered outputs",
		"formats", opts.Formats,
		"duration", result.Stats.RenderTime)

	return result, nil
}
