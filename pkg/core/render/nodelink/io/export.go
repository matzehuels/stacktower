package io

import (
	"encoding/json"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/nodelink"
)

// WriteOption configures layout serialization.
type WriteOption func(*writeConfig)

type writeConfig struct {
	engine  string
	style   string
	width   float64
	height  float64
	graph   *dag.DAG
	options nodelink.Options
}

// WithEngine sets the graphviz engine name in the output.
func WithEngine(engine string) WriteOption {
	return func(c *writeConfig) { c.engine = engine }
}

// WithStyle sets the style name in the output.
func WithStyle(style string) WriteOption {
	return func(c *writeConfig) { c.style = style }
}

// WithDimensions sets the frame dimensions in the output.
func WithDimensions(width, height float64) WriteOption {
	return func(c *writeConfig) { c.width = width; c.height = height }
}

// WithGraph attaches the DAG for DOT generation and metadata.
func WithGraph(g *dag.DAG) WriteOption {
	return func(c *writeConfig) { c.graph = g }
}

// WithOptions sets the nodelink rendering options.
func WithOptions(opts nodelink.Options) WriteOption {
	return func(c *writeConfig) { c.options = opts }
}

// WriteLayout serializes a nodelink layout to JSON.
// The layout includes the DOT string and metadata for re-rendering.
//
// Example:
//
//	data, err := io.WriteLayout(
//	    io.WithGraph(g),
//	    io.WithDimensions(800, 600),
//	    io.WithEngine("dot"),
//	)
func WriteLayout(opts ...WriteOption) ([]byte, error) {
	cfg := &writeConfig{
		engine: "dot", // default engine
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Generate DOT from graph
	dot := ""
	nodeCount, edgeCount := 0, 0
	if cfg.graph != nil {
		dot = nodelink.ToDOT(cfg.graph, cfg.options)
		nodeCount = cfg.graph.NodeCount()
		edgeCount = cfg.graph.EdgeCount()
	}

	data := &LayoutData{
		VizType:   VizType,
		DOT:       dot,
		Width:     cfg.width,
		Height:    cfg.height,
		Engine:    cfg.engine,
		Style:     cfg.style,
		NodeCount: nodeCount,
		EdgeCount: edgeCount,
	}

	return json.MarshalIndent(data, "", "  ")
}

// WriteLayoutFromDOT creates layout data from an existing DOT string.
func WriteLayoutFromDOT(dot string, opts ...WriteOption) ([]byte, error) {
	cfg := &writeConfig{
		engine: "dot",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	data := &LayoutData{
		VizType: VizType,
		DOT:     dot,
		Width:   cfg.width,
		Height:  cfg.height,
		Engine:  cfg.engine,
		Style:   cfg.style,
	}

	// Get counts from graph if provided
	if cfg.graph != nil {
		data.NodeCount = cfg.graph.NodeCount()
		data.EdgeCount = cfg.graph.EdgeCount()
	}

	return json.MarshalIndent(data, "", "  ")
}
