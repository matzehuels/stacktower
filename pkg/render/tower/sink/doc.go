// Package sink provides output format renderers for tower visualizations.
//
// # Overview
//
// A "sink" transforms a computed [layout.Layout] into a final output format.
// This package provides renderers for:
//
//   - SVG: Scalable vector graphics with interactivity
//   - JSON: Layout data export for external tools
//   - PDF: Print-ready output (requires rsvg-convert)
//   - PNG: Raster image output (requires rsvg-convert)
//
// # SVG Output
//
// [RenderSVG] produces interactive SVG with:
//
//   - Visual styles (hand-drawn XKCD-style or clean simple style)
//   - Hover highlighting of related blocks
//   - Optional popups showing package metadata
//   - Optional "Nebraska guy" ranking panel
//   - Optional dependency edge visualization
//
// Basic usage:
//
//	svg := sink.RenderSVG(layout,
//	    sink.WithGraph(g),
//	    sink.WithStyle(handdrawn.New(seed)),
//	    sink.WithPopups(),
//	)
//
// # SVG Options
//
//   - [WithGraph]: Required for edge rendering and metadata access
//   - [WithStyle]: Visual style ([styles.Simple] or [handdrawn.New])
//   - [WithEdges]: Show dependency edges as dashed lines
//   - [WithMerged]: Merge subdivider blocks into continuous columns
//   - [WithPopups]: Enable hover popups with package metadata
//   - [WithNebraska]: Add maintainer ranking panel
//
// # JSON Output
//
// [RenderJSON] exports the complete layout as JSON, enabling:
//
//   - Integration with external visualization tools
//   - Caching of layout computations
//   - Round-trip rendering (re-import and render identically)
//
// The JSON includes all render options (style, seed, randomize, merged) and
// node flags (auxiliary, synthetic) needed to reproduce the exact visual.
//
// # PDF and PNG Output
//
// [RenderPDF] and [RenderPNG] render the layout as PDF/PNG by first generating
// SVG, then converting via [render.ToPDF] and [render.ToPNG]:
//
//	pdf, err := sink.RenderPDF(layout, opts...)
//	png, err := sink.RenderPNG(layout, sink.WithScale(2), opts...)
//
// These require librsvg to be installed:
//   - macOS: brew install librsvg
//   - Linux: apt install librsvg2-bin
//
// The conversion functions are shared with [nodelink] so both visualization
// types can export to PDF/PNG.
//
// [render.ToPDF]: github.com/matzehuels/stacktower/pkg/render.ToPDF
// [render.ToPNG]: github.com/matzehuels/stacktower/pkg/render.ToPNG
// [nodelink]: github.com/matzehuels/stacktower/pkg/render/nodelink
//
// # Adding New Formats
//
// To add a new output format:
//
//  1. Create a renderer function: func RenderFoo(l layout.Layout, opts ...FooOption) ([]byte, error)
//  2. Define option types for configuration
//  3. Access l.Blocks for positioned blocks, l.RowOrders for orderings
//  4. Register in internal/cli/render.go for CLI support
//
// The existing sinks provide examples: svg.go for full-featured output,
// json.go for data export, pdf.go/png.go for format conversion wrappers.
//
// [layout.Layout]: github.com/matzehuels/stacktower/pkg/render/tower/layout.Layout
// [styles.Simple]: github.com/matzehuels/stacktower/pkg/render/tower/styles.Simple
// [handdrawn.New]: github.com/matzehuels/stacktower/pkg/render/tower/styles/handdrawn.New
package sink
