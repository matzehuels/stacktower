package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/nodelink"
	nodelinkio "github.com/matzehuels/stacktower/pkg/core/render/nodelink/io"
	towerio "github.com/matzehuels/stacktower/pkg/core/render/tower/io"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/layout"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/sink"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/styles/handdrawn"
)

// Render generates output artifacts in the requested formats.
// If l is empty (Blocks == nil), it will be deserialized from layoutData.
func Render(l layout.Layout, layoutData []byte, g *dag.DAG, opts Options) (map[string][]byte, error) {
	if opts.IsNodelink() {
		return renderNodelink(layoutData, opts)
	}
	return renderTower(l, layoutData, g, opts)
}

// renderNodelink generates nodelink outputs using Graphviz.
func renderNodelink(layoutData []byte, opts Options) (map[string][]byte, error) {
	// Extract DOT from layout JSON
	dot, err := extractDOT(layoutData)
	if err != nil {
		return nil, fmt.Errorf("extract DOT from layout: %w", err)
	}

	artifacts := make(map[string][]byte)

	for _, format := range opts.Formats {
		var data []byte
		var err error

		switch format {
		case "svg":
			data, err = nodelink.RenderSVG(dot)
		case "png":
			data, err = nodelink.RenderPNG(dot, 2.0)
		case "pdf":
			data, err = nodelink.RenderPDF(dot)
		case "json":
			// Return the layout JSON itself
			data = layoutData
		default:
			return nil, fmt.Errorf("unsupported nodelink format: %s", format)
		}

		if err != nil {
			return nil, fmt.Errorf("render %s: %w", format, err)
		}
		artifacts[format] = data
	}

	return artifacts, nil
}

// extractDOT extracts the DOT string from nodelink layout JSON.
// Falls back to treating the data as raw DOT if JSON parsing fails.
func extractDOT(data []byte) (string, error) {
	// Try to parse as nodelink layout JSON
	layoutData, _, err := nodelinkio.ReadLayout(bytes.NewReader(data))
	if err == nil && layoutData.DOT != "" {
		return layoutData.DOT, nil
	}

	// Fall back to treating data as raw DOT (for backwards compatibility)
	if len(data) > 0 && data[0] != '{' {
		return string(data), nil
	}

	return "", fmt.Errorf("invalid nodelink layout: %w", err)
}

// renderTower generates tower outputs.
func renderTower(l layout.Layout, layoutData []byte, g *dag.DAG, opts Options) (map[string][]byte, error) {
	// If layout is empty (Blocks == nil), deserialize from data
	if l.Blocks == nil && len(layoutData) > 0 {
		var meta towerio.LayoutMeta
		var err error
		l, meta, err = towerio.ReadLayout(bytes.NewReader(layoutData))
		if err != nil {
			return nil, fmt.Errorf("parse layout: %w", err)
		}
		// Use meta values if options not set
		if opts.Style == "" {
			opts.Style = meta.Style
		}
		if opts.Seed == 0 {
			opts.Seed = meta.Seed
		}
		if !opts.Merge {
			opts.Merge = meta.Merged
		}
	}

	svgOpts := buildSVGOptions(g, opts)
	artifacts := make(map[string][]byte)

	for _, format := range opts.Formats {
		var data []byte
		var err error

		switch format {
		case "svg":
			data = sink.RenderSVG(l, svgOpts...)
		case "png":
			data, err = sink.RenderPNG(l, sink.WithPNGSVGOptions(svgOpts...))
		case "pdf":
			data, err = sink.RenderPDF(l, sink.WithPDFSVGOptions(svgOpts...))
		case "json":
			data = layoutData
		default:
			return nil, fmt.Errorf("unsupported tower format: %s", format)
		}

		if err != nil {
			return nil, fmt.Errorf("render %s: %w", format, err)
		}
		artifacts[format] = data
	}

	return artifacts, nil
}

// buildSVGOptions builds SVG rendering options.
func buildSVGOptions(g *dag.DAG, opts Options) []sink.SVGOption {
	var svgOpts []sink.SVGOption

	if g != nil {
		svgOpts = append(svgOpts, sink.WithGraph(g))
	}
	if opts.ShowEdges {
		svgOpts = append(svgOpts, sink.WithEdges())
	}
	if opts.Merge {
		svgOpts = append(svgOpts, sink.WithMerged())
	}

	if opts.Style == "handdrawn" {
		seed := opts.Seed
		if seed == 0 {
			seed = 42
		}
		svgOpts = append(svgOpts, sink.WithStyle(handdrawn.New(seed)))

		if opts.Popups && g != nil {
			svgOpts = append(svgOpts, sink.WithPopups())
		}
	}

	return svgOpts
}

// RenderFromLayoutData renders output from serialized layout data.
// This is useful when the layout was computed elsewhere (e.g., cached).
func RenderFromLayoutData(layoutData []byte, g *dag.DAG, opts Options) (map[string][]byte, error) {
	// Detect viz type from layout JSON
	vizType := detectVizType(layoutData)
	if vizType == nodelinkio.VizType {
		opts.VizType = VizTypeNodelink
		return renderNodelink(layoutData, opts)
	}

	l, meta, err := towerio.ReadLayout(bytes.NewReader(layoutData))
	if err != nil {
		return nil, fmt.Errorf("parse layout: %w", err)
	}

	// Apply meta values if options not set
	if opts.Style == "" {
		opts.Style = meta.Style
	}
	if opts.Seed == 0 {
		opts.Seed = meta.Seed
	}
	if !opts.Merge {
		opts.Merge = meta.Merged
	}

	return renderTower(l, layoutData, g, opts)
}

// detectVizType reads the viz_type field from layout JSON.
func detectVizType(data []byte) string {
	var header struct {
		VizType string `json:"viz_type"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return ""
	}
	return header.VizType
}
