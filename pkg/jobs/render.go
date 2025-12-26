package jobs

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// Default values for visualization options.
const (
	DefaultMaxDepth = 10
	DefaultMaxNodes = 5000
	DefaultWidth    = 800.0
	DefaultHeight   = 600.0
	DefaultVizType  = "tower"
	DefaultStyle    = "handdrawn"
	DefaultSeed     = uint64(42)
)

// RenderPayload defines input for the full rendering pipeline.
//
// A render job runs the complete pipeline: parse → layout → visualize.
// This is a convenience job that combines all three stages.
//
// Result:
//
//	{
//	  "graph_path": "job-123/graph.json",
//	  "layout_path": "job-123/layout.json",
//	  "svg": "job-123/tower.svg",
//	  "nodes": 50,
//	  "edges": 80
//	}
type RenderPayload struct {
	// --- Parse options ---

	// Language is the package ecosystem (required).
	Language string `json:"language"`

	// Package is the package name or manifest file path (required unless Manifest is provided).
	Package string `json:"package"`

	// Manifest is the raw manifest file content (e.g., package.json contents).
	// When provided, the manifest is parsed directly instead of looking up a package.
	Manifest string `json:"manifest,omitempty"`

	// ManifestFilename is the filename of the manifest (e.g., "package.json").
	// Required when Manifest is provided.
	ManifestFilename string `json:"manifest_filename,omitempty"`

	// MaxDepth limits dependency traversal depth.
	MaxDepth int `json:"max_depth,omitempty"`

	// MaxNodes limits total nodes in the graph.
	MaxNodes int `json:"max_nodes,omitempty"`

	// Enrich fetches GitHub metadata.
	Enrich bool `json:"enrich,omitempty"`

	// Refresh bypasses the dependency cache.
	Refresh bool `json:"refresh,omitempty"`

	// Normalize applies DAG normalization.
	Normalize bool `json:"normalize,omitempty"`

	// --- Layout options ---

	// VizType selects visualization algorithm: "tower" (default) or "nodelink".
	VizType string `json:"viz_type,omitempty"`

	// Width of the output frame.
	Width float64 `json:"width,omitempty"`

	// Height of the output frame.
	Height float64 `json:"height,omitempty"`

	// Ordering algorithm (tower only).
	Ordering string `json:"ordering,omitempty"`

	// Randomize block widths (tower only).
	Randomize bool `json:"randomize,omitempty"`

	// Merge subdivider blocks (tower only).
	Merge bool `json:"merge,omitempty"`

	// Seed for randomization.
	Seed uint64 `json:"seed,omitempty"`

	// Engine for graphviz layout (nodelink only).
	Engine string `json:"engine,omitempty"`

	// --- Export options ---

	// Formats to generate: "svg", "png", "pdf".
	Formats []string `json:"formats,omitempty"`

	// Style for rendering.
	Style string `json:"style,omitempty"`

	// ShowEdges renders dependency edges.
	ShowEdges bool `json:"show_edges,omitempty"`

	// Nebraska adds maintainer ranking (tower only).
	Nebraska bool `json:"nebraska,omitempty"`

	// Popups enables hover popups (tower only).
	Popups bool `json:"popups,omitempty"`

	// Webhook is an optional callback URL.
	Webhook string `json:"webhook,omitempty"`
}

// ValidateAndSetDefaults checks required fields and applies defaults.
func (p *RenderPayload) ValidateAndSetDefaults() error {
	if p.Language == "" {
		return fmt.Errorf("language is required")
	}
	if p.Package == "" && p.Manifest == "" {
		return fmt.Errorf("package or manifest is required")
	}
	if p.Manifest != "" && p.ManifestFilename == "" {
		return fmt.Errorf("manifest_filename is required when manifest is provided")
	}

	if p.MaxDepth == 0 {
		p.MaxDepth = DefaultMaxDepth
	}
	if p.MaxNodes == 0 {
		p.MaxNodes = DefaultMaxNodes
	}
	if p.Width == 0 {
		p.Width = DefaultWidth
	}
	if p.Height == 0 {
		p.Height = DefaultHeight
	}
	if p.VizType == "" {
		p.VizType = DefaultVizType
	}
	if p.Style == "" {
		p.Style = DefaultStyle
	}
	if len(p.Formats) == 0 {
		p.Formats = []string{"svg"}
	}
	if p.Seed == 0 {
		p.Seed = DefaultSeed
	}
	return nil
}

// ToPipelineOptions converts the payload to pipeline.Options.
func (p *RenderPayload) ToPipelineOptions() pipeline.Options {
	return pipeline.Options{
		Language:         p.Language,
		Package:          p.Package,
		Manifest:         p.Manifest,
		ManifestFilename: p.ManifestFilename,
		MaxDepth:         p.MaxDepth,
		MaxNodes:         p.MaxNodes,
		Enrich:           p.Enrich,
		Refresh:          p.Refresh,
		Normalize:        p.Normalize,
		VizType:          p.VizType,
		Width:            p.Width,
		Height:           p.Height,
		Ordering:         p.Ordering,
		Merge:            p.Merge,
		Randomize:        p.Randomize,
		Seed:             p.Seed,
		Formats:          p.Formats,
		Style:            p.Style,
		ShowEdges:        p.ShowEdges,
		Nebraska:         p.Nebraska,
		Popups:           p.Popups,
	}
}

// ToParsePayload extracts parse parameters.
func (p *RenderPayload) ToParsePayload() *ParsePayload {
	return &ParsePayload{
		Language:         p.Language,
		Package:          p.Package,
		Manifest:         p.Manifest,
		ManifestFilename: p.ManifestFilename,
		MaxDepth:         p.MaxDepth,
		MaxNodes:         p.MaxNodes,
		Enrich:           p.Enrich,
		Refresh:          p.Refresh,
		Normalize:        p.Normalize,
	}
}

// ToLayoutPayload extracts layout parameters.
func (p *RenderPayload) ToLayoutPayload(graphPath string) *LayoutPayload {
	return &LayoutPayload{
		GraphPath: graphPath,
		VizType:   p.VizType,
		Width:     p.Width,
		Height:    p.Height,
		Ordering:  p.Ordering,
		Randomize: p.Randomize,
		Merge:     p.Merge,
		Seed:      p.Seed,
		Engine:    p.Engine,
	}
}

// ToVisualizePayload extracts visualize parameters.
func (p *RenderPayload) ToVisualizePayload(layoutPath string) *VisualizePayload {
	formats := p.Formats
	if len(formats) == 0 {
		formats = []string{"svg"}
	}
	return &VisualizePayload{
		LayoutPath: layoutPath,
		VizType:    p.VizType,
		Formats:    formats,
		Style:      p.Style,
		ShowEdges:  p.ShowEdges,
		Popups:     p.Popups,
	}
}
