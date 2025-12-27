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

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/layout"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/ordering"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

// =============================================================================
// Default values - single source of truth for CLI, API, and Worker
// =============================================================================

const (
	// DefaultMaxDepth is the maximum dependency traversal depth.
	DefaultMaxDepth = 10

	// DefaultMaxNodes is the maximum number of nodes to fetch.
	DefaultMaxNodes = 5000

	// DefaultWidth is the default frame width in pixels.
	DefaultWidth = 800.0

	// DefaultHeight is the default frame height in pixels.
	DefaultHeight = 600.0

	// DefaultVizType is the default visualization type.
	DefaultVizType = "tower"

	// DefaultStyle is the default visual style.
	DefaultStyle = "handdrawn"

	// DefaultSeed is the default random seed for reproducibility.
	DefaultSeed = uint64(42)

	// DefaultOrdering is the default ordering algorithm.
	DefaultOrdering = "optimal"
)

// VizType constants for visualization types.
const (
	VizTypeTower    = "tower"
	VizTypeNodelink = "nodelink"
)

// Style constants for visual styles.
const (
	StyleSimple    = "simple"
	StyleHanddrawn = "handdrawn"
)

// Format constants for output formats.
const (
	FormatSVG = "svg"
	FormatPNG = "png"
	FormatPDF = "pdf"
)

// ValidFormats is the set of supported output formats.
var ValidFormats = map[string]bool{
	FormatSVG: true,
	FormatPNG: true,
	FormatPDF: true,
}

// ValidStyles is the set of supported visual styles.
var ValidStyles = map[string]bool{
	StyleSimple:    true,
	StyleHanddrawn: true,
}

// ValidVizTypes is the set of supported visualization types.
var ValidVizTypes = map[string]bool{
	VizTypeTower:    true,
	VizTypeNodelink: true,
}

// Options contains all configuration for the visualization pipeline.
// This struct supports JSON serialization for API requests.
type Options struct {
	// User context (for scoped storage and authorization)
	UserID string        `json:"user_id,omitempty"`
	Scope  storage.Scope `json:"scope,omitempty"`

	// Parse options
	Language         string `json:"language"`
	Package          string `json:"package,omitempty"`
	Manifest         string `json:"manifest,omitempty"`
	ManifestFilename string `json:"manifest_filename,omitempty"`
	Repo             string `json:"repo,omitempty"` // GitHub repository (owner/repo) for tracking
	MaxDepth         int    `json:"max_depth,omitempty"`
	MaxNodes         int    `json:"max_nodes,omitempty"`
	Enrich           bool   `json:"enrich,omitempty"`
	Refresh          bool   `json:"refresh,omitempty"`
	Normalize        bool   `json:"normalize,omitempty"`

	// Layout options
	VizType   string  `json:"viz_type,omitempty"`
	Width     float64 `json:"width,omitempty"`
	Height    float64 `json:"height,omitempty"`
	Ordering  string  `json:"ordering,omitempty"`
	Merge     bool    `json:"merge,omitempty"`
	Randomize bool    `json:"randomize,omitempty"`
	Seed      uint64  `json:"seed,omitempty"`

	// Render options
	Formats   []string `json:"formats,omitempty"`
	Style     string   `json:"style,omitempty"`
	ShowEdges bool     `json:"show_edges,omitempty"`
	Nebraska  bool     `json:"nebraska,omitempty"`
	Popups    bool     `json:"popups,omitempty"`

	// Runtime options (not serialized)
	Logger      *log.Logger      `json:"-"`
	GitHubToken string           `json:"-"`
	Orderer     ordering.Orderer `json:"-"`
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

// ValidateFormat checks that a format is valid.
func ValidateFormat(format string) error {
	if !ValidFormats[format] {
		return fmt.Errorf("invalid format: %q (must be one of: svg, png, pdf)", format)
	}
	return nil
}

// ValidateFormats checks that all formats are valid.
func ValidateFormats(formats []string) error {
	for _, f := range formats {
		if err := ValidateFormat(f); err != nil {
			return err
		}
	}
	return nil
}

// ValidateStyle checks that a style is valid.
func ValidateStyle(style string) error {
	if !ValidStyles[style] {
		return fmt.Errorf("invalid style: %q (must be one of: simple, handdrawn)", style)
	}
	return nil
}

// ValidateVizType checks that a visualization type is valid.
func ValidateVizType(vizType string) error {
	if !ValidVizTypes[vizType] {
		return fmt.Errorf("invalid viz_type: %q (must be one of: tower, nodelink)", vizType)
	}
	return nil
}

// ValidateAndSetDefaults checks required fields and applies defaults for the full pipeline.
// Use ValidateForParse, ValidateForLayout, or ValidateForRender for stage-specific validation.
func (o *Options) ValidateAndSetDefaults() error {
	if err := o.ValidateForParse(); err != nil {
		return err
	}
	o.SetLayoutDefaults()
	o.SetRenderDefaults()
	return nil
}

// ValidateForParse checks required fields for parsing.
func (o *Options) ValidateForParse() error {
	if o.Language == "" {
		return fmt.Errorf("language is required")
	}
	if o.Package == "" && o.Manifest == "" {
		return fmt.Errorf("package or manifest is required")
	}
	if o.Manifest != "" && o.ManifestFilename == "" {
		return fmt.Errorf("manifest_filename is required when manifest is provided")
	}

	// Parse defaults.
	if o.MaxDepth == 0 {
		o.MaxDepth = DefaultMaxDepth
	}
	if o.MaxNodes == 0 {
		o.MaxNodes = DefaultMaxNodes
	}

	// Logger default.
	if o.Logger == nil {
		o.Logger = log.NewWithOptions(io.Discard, log.Options{})
	}

	return nil
}

// SetLayoutDefaults sets default values for layout computation.
func (o *Options) SetLayoutDefaults() {
	if o.VizType == "" {
		o.VizType = DefaultVizType
	}
	if o.Width == 0 {
		o.Width = DefaultWidth
	}
	if o.Height == 0 {
		o.Height = DefaultHeight
	}
	if o.Seed == 0 {
		o.Seed = DefaultSeed
	}
	if o.Logger == nil {
		o.Logger = log.NewWithOptions(io.Discard, log.Options{})
	}
}

// ValidateForLayout validates and sets defaults for layout computation.
func (o *Options) ValidateForLayout() error {
	o.SetLayoutDefaults()
	if err := ValidateVizType(o.VizType); err != nil {
		return err
	}
	return nil
}

// SetRenderDefaults sets default values for rendering.
func (o *Options) SetRenderDefaults() {
	if len(o.Formats) == 0 {
		o.Formats = []string{FormatSVG}
	}
	if o.Style == "" {
		o.Style = DefaultStyle
	}
	if o.Logger == nil {
		o.Logger = log.NewWithOptions(io.Discard, log.Options{})
	}
}

// ValidateForRender validates and sets defaults for rendering.
func (o *Options) ValidateForRender() error {
	o.SetLayoutDefaults()
	o.SetRenderDefaults()
	if err := ValidateVizType(o.VizType); err != nil {
		return err
	}
	if err := ValidateFormats(o.Formats); err != nil {
		return err
	}
	if err := ValidateStyle(o.Style); err != nil {
		return err
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
// The backend parameter provides caching for HTTP responses during dependency resolution.
// Use storage.NullBackend{} to disable caching.
func Execute(ctx context.Context, backend storage.Backend, opts Options) (*Result, error) {
	if err := opts.ValidateAndSetDefaults(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	result := &Result{
		Artifacts: make(map[string][]byte),
	}

	// Stage 1: Parse
	parseStart := time.Now()
	g, err := Parse(ctx, backend, opts)
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
