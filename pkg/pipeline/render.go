package pipeline

import (
	"bytes"
	"fmt"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/nodelink"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/feature"
	towerio "github.com/matzehuels/stacktower/pkg/core/render/tower/io"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/layout"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/sink"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/styles/handdrawn"
)

// RenderOptions contains options for output rendering.
type RenderOptions struct {
	VizType   string
	Formats   []string
	Style     string
	ShowEdges bool
	Nebraska  bool
	Popups    bool
	Merge     bool
	Seed      uint64
}

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
	dot := string(layoutData)
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
			// JSON layout not supported for nodelink
			continue
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
			data, err = sink.RenderJSON(l, buildJSONOptions(g, opts)...)
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

		if opts.Nebraska && g != nil {
			svgOpts = append(svgOpts, sink.WithNebraska(feature.RankNebraska(g, 5)))
		}
		if opts.Popups && g != nil {
			svgOpts = append(svgOpts, sink.WithPopups())
		}
	}

	return svgOpts
}

// buildJSONOptions builds JSON rendering options.
func buildJSONOptions(g *dag.DAG, opts Options) []sink.JSONOption {
	var jsonOpts []sink.JSONOption

	if g != nil {
		jsonOpts = append(jsonOpts, sink.WithJSONGraph(g))
	}
	if opts.Merge {
		jsonOpts = append(jsonOpts, sink.WithJSONMerged())
	}
	if opts.Randomize {
		jsonOpts = append(jsonOpts, sink.WithJSONRandomize(opts.Seed))
	}
	if opts.Style != "" {
		jsonOpts = append(jsonOpts, sink.WithJSONStyle(opts.Style))
	}
	if opts.Nebraska && g != nil {
		jsonOpts = append(jsonOpts, sink.WithJSONNebraska(feature.RankNebraska(g, 5)))
	}

	return jsonOpts
}

// RenderFromLayoutData renders output from serialized layout data.
// This is useful when the layout was computed elsewhere (e.g., cached).
func RenderFromLayoutData(layoutData []byte, g *dag.DAG, opts Options) (map[string][]byte, error) {
	if opts.IsNodelink() {
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
