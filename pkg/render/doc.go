// Package render provides visualization rendering for dependency graphs.
//
// # Overview
//
// This package contains the rendering pipeline that transforms dependency
// graphs into visual outputs. It provides:
//
//   - Generic format conversion (SVG to PDF/PNG)
//   - Tower visualization (in [tower] subpackage)
//   - Node-link diagrams (in [nodelink] subpackage)
//
// # Format Conversion
//
// The [ToPDF] and [ToPNG] functions convert any SVG to other formats using
// the external rsvg-convert tool (from librsvg). These are used by both
// tower and node-link renderers.
//
//	svg := tower.RenderSVG(layout, opts...)
//	pdf, err := render.ToPDF(svg)
//	png, err := render.ToPNG(svg, 2.0)  // 2x scale
//
// # Tower Visualization
//
// The [tower] subpackage renders dependency graphs as stacked physical towers
// where blocks rest on what they depend on. This is Stacktower's signature
// visualization style, inspired by XKCD #2347.
//
// Key tower subpackages:
//   - [tower/layout]: Block position computation
//   - [tower/ordering]: Row ordering algorithms (barycentric, optimal)
//   - [tower/sink]: Output formats (SVG, JSON)
//   - [tower/styles]: Visual styles (handdrawn, simple)
//
// # Node-Link Diagrams
//
// The [nodelink] subpackage renders traditional directed graph diagrams
// using Graphviz. Nodes appear as boxes connected by arrows.
//
//	dot := nodelink.ToDOT(g, nodelink.Options{})
//	svg, err := nodelink.RenderSVG(dot)
//	pdf, err := render.ToPDF(svg)
//
// [tower]: github.com/matzehuels/stacktower/pkg/render/tower
// [tower/layout]: github.com/matzehuels/stacktower/pkg/render/tower/layout
// [tower/ordering]: github.com/matzehuels/stacktower/pkg/render/tower/ordering
// [tower/sink]: github.com/matzehuels/stacktower/pkg/render/tower/sink
// [tower/styles]: github.com/matzehuels/stacktower/pkg/render/tower/styles
// [nodelink]: github.com/matzehuels/stacktower/pkg/render/nodelink
package render
