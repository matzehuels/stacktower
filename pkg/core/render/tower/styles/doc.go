// Package styles defines visual styles for tower rendering.
//
// # Overview
//
// Stacktower supports multiple visual styles that control how blocks, edges,
// text, and interactive elements are rendered. This package provides:
//
//   - [Style]: The interface that all styles implement
//   - [Simple]: A clean, minimal style with solid colors
//   - [handdrawn]: An XKCD-inspired hand-drawn aesthetic (in subpackage)
//
// # The Style Interface
//
// All styles implement [Style], which provides methods for rendering each
// visual element:
//
//   - RenderDefs: SVG <defs> section (filters, patterns, gradients)
//   - RenderBlock: Individual block shapes
//   - RenderEdge: Dependency edges (when enabled)
//   - RenderText: Block labels
//   - RenderPopup: Hover popup content
//
// # Simple Style
//
// [Simple] provides a clean, professional appearance with:
//
//   - Solid fill colors with subtle gradients
//   - Clean rectangular blocks
//   - Standard sans-serif fonts
//
// Usage:
//
//	svg := sink.RenderSVG(layout, sink.WithStyle(styles.Simple{}))
//
// # Hand-Drawn Style
//
// The [handdrawn] subpackage provides the signature XKCD-inspired aesthetic:
//
//   - Wobbly, imperfect lines (via SVG filters)
//   - Rough, sketchy fills
//   - Comic Sans-style fonts
//   - Textured backgrounds
//   - "Brittle" visual treatment for at-risk packages
//
// The hand-drawn style uses a seed for reproducible randomness:
//
//	style := handdrawn.New(42)  // Seed for consistent wobbly lines
//	svg := sink.RenderSVG(layout, sink.WithStyle(style))
//
// # Block Data
//
// Styles receive [Block] structs containing all information needed for rendering:
//
//   - ID, Label: Identification and display text
//   - X, Y, W, H: Position and dimensions
//   - CX, CY: Center coordinates for text placement
//   - URL: Optional link target
//   - Popup: Metadata for hover popups
//   - Brittle: Flag for visual warning treatment
//
// # Creating Custom Styles
//
// To create a custom style:
//
//  1. Implement the [Style] interface
//  2. Use the provided [Block] and [Edge] data for positioning
//  3. Write SVG elements to the provided bytes.Buffer
//
// Example structure:
//
//	type MyStyle struct {
//	    Color string
//	}
//
//	func (s MyStyle) RenderBlock(buf *bytes.Buffer, b Block) {
//	    fmt.Fprintf(buf, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`,
//	        b.X, b.Y, b.W, b.H, s.Color)
//	}
//
// [handdrawn]: github.com/matzehuels/stacktower/pkg/core/render/tower/styles/handdrawn
package styles
