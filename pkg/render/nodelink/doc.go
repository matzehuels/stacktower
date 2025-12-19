// Package nodelink renders dependency graphs as traditional node-link diagrams.
//
// # Overview
//
// This package produces directed graph visualizations using Graphviz, where
// nodes appear as boxes connected by arrows. It's an alternative to the
// tower visualization for cases where a traditional diagram is preferred.
//
// # Usage
//
// Convert a DAG to DOT format, then render to SVG:
//
//	dot := nodelink.ToDOT(g, nodelink.Options{Detailed: false})
//	svg, err := nodelink.RenderSVG(dot)
//
// For PDF or PNG output, use the render functions:
//
//	pdf, err := nodelink.RenderPDF(dot)
//	png, err := nodelink.RenderPNG(dot, 2.0)  // 2x scale
//
// # Options
//
// The [Options] struct controls diagram generation:
//
//   - Detailed: When true, node labels include all metadata (row, version, etc.)
//
// # DOT Format
//
// The [ToDOT] function produces Graphviz DOT source that can be:
//
//   - Rendered directly via [RenderSVG]
//   - Saved and processed with external Graphviz tools
//   - Customized before rendering
//
// The generated DOT uses top-to-bottom layout (rankdir=TB) with rounded
// box nodes, matching the tower visualization's vertical orientation.
//
// # Dependencies
//
// This package uses [github.com/goccy/go-graphviz] for in-process SVG
// rendering. PDF and PNG conversion requires librsvg (rsvg-convert).
package nodelink
