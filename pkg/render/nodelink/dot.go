package nodelink

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/goccy/go-graphviz"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render"
)

// Options configures node-link diagram rendering.
type Options struct {
	// Detailed includes row numbers and metadata in node labels.
	// When false, only the node ID is shown.
	Detailed bool
}

// ToDOT converts a DAG to Graphviz DOT format for node-link visualization.
// The resulting DOT string can be rendered using [RenderSVG], [RenderPDF], or [RenderPNG].
//
// Subdivider nodes (created by [dag/transform.Subdivide]) are rendered with dashed
// outlines and grey fill to distinguish them from regular nodes.
func ToDOT(g *dag.DAG, opts Options) string {
	var buf bytes.Buffer
	buf.WriteString("digraph G {\n")
	buf.WriteString("  rankdir=TB;\n")
	buf.WriteString("  bgcolor=\"transparent\";\n")
	buf.WriteString("  node [shape=box, style=\"rounded,filled\", fillcolor=white, fontsize=24, margin=\"0.2,0.1\"];\n")
	buf.WriteString("  ranksep=0.5;\n")
	buf.WriteString("  nodesep=0.3;\n")
	buf.WriteString("\n")

	for _, n := range g.Nodes() {
		label := fmtLabel(*n, opts.Detailed)
		attrs := fmtAttrs(*n, label)
		fmt.Fprintf(&buf, "  %q [%s];\n", n.ID, strings.Join(attrs, ", "))
	}

	buf.WriteString("\n")
	for _, e := range g.Edges() {
		fmt.Fprintf(&buf, "  %q -> %q;\n", e.From, e.To)
	}

	buf.WriteString("}\n")
	return buf.String()
}

func fmtLabel(n dag.Node, detailed bool) string {
	if !detailed {
		return n.ID
	}

	parts := []string{fmt.Sprintf("row: %d", n.Row)}
	for _, k := range slices.Sorted(maps.Keys(n.Meta)) {
		parts = append(parts, fmt.Sprintf("%s: %v", k, n.Meta[k]))
	}

	return n.ID + "\n" + strings.Join(parts, "\n")
}

func fmtAttrs(n dag.Node, label string) []string {
	attrs := []string{fmt.Sprintf("label=%q", label)}
	if n.IsSubdivider() {
		attrs = append(attrs, "style=\"rounded,filled,dashed\"", "fillcolor=lightgrey", "fontcolor=black")
	}
	return attrs
}

// RenderSVG renders a DOT graph to SVG using Graphviz.
// Returns the SVG bytes ready for display or further conversion with [render.ToPDF] or [render.ToPNG].
func RenderSVG(dot string) ([]byte, error) {
	ctx := context.Background()
	gv, err := graphviz.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("init graphviz: %w", err)
	}
	defer gv.Close()

	g, err := graphviz.ParseBytes([]byte(dot))
	if err != nil {
		return nil, fmt.Errorf("parse DOT: %w", err)
	}
	defer g.Close()

	var buf bytes.Buffer
	if err := gv.Render(ctx, g, graphviz.SVG, &buf); err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}
	return normalizeViewBox(buf.Bytes()), nil
}

var (
	svgTagRe  = regexp.MustCompile(`<svg[^>]*>`)
	viewBoxRe = regexp.MustCompile(`viewBox="([0-9.]+)\s+([0-9.]+)\s+([0-9.]+)\s+([0-9.]+)"`)
)

func normalizeViewBox(svg []byte) []byte {
	match := viewBoxRe.FindSubmatch(svg)
	if match == nil {
		return svg
	}

	w, _ := strconv.ParseFloat(string(match[3]), 64)
	h, _ := strconv.ParseFloat(string(match[4]), 64)
	if w == 0 || h == 0 {
		return svg
	}

	newSvg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.2f %.2f" width="%.0f" height="%.0f">`,
		w, h, w, h)

	return svgTagRe.ReplaceAll(svg, []byte(newSvg))
}

// RenderPDF renders a DOT graph as PDF via SVG conversion.
// This is a convenience wrapper around [RenderSVG] and [render.ToPDF].
//
// Requires librsvg: brew install librsvg (macOS), apt install librsvg2-bin (Linux).
func RenderPDF(dot string) ([]byte, error) {
	svg, err := RenderSVG(dot)
	if err != nil {
		return nil, err
	}
	return render.ToPDF(svg)
}

// RenderPNG renders a DOT graph as PNG via SVG conversion.
// This is a convenience wrapper around [RenderSVG] and [render.ToPNG].
//
// A scale of 2.0 produces a 2x resolution image suitable for high-DPI displays.
//
// Requires librsvg: brew install librsvg (macOS), apt install librsvg2-bin (Linux).
func RenderPNG(dot string, scale float64) ([]byte, error) {
	svg, err := RenderSVG(dot)
	if err != nil {
		return nil, err
	}
	return render.ToPNG(svg, scale)
}
