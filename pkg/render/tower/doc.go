// Package tower provides the physical tower visualization engine.
//
// # Overview
//
// Stacktower's primary visualization is a "tower" of blocks, where each block
// represents a package and rests on the blocks it depends on. This package
// implements the multi-stage pipeline required to transform a DAG into a
// 2D tower layout:
//
//  1. Ordering ([ordering]): Determine horizontal sequence of blocks in each row to minimize crossings.
//  2. Layout ([layout]): Compute (x, y) coordinates and dimensions (w, h) for every block.
//  3. Styles ([styles]): Define the visual appearance (simple, hand-drawn, colors, text).
//  4. Sink ([sink]): Export the final layout to various formats (SVG, JSON, PNG, PDF).
//
// # Rendering Pipeline
//
// The rendering process typically follows these steps:
//
//	g := dag.New(...)
//	// ... populate graph ...
//
//	// 1. Transform the graph into a row-based structure
//	transform.Normalize(g)
//
//	// 2. Compute the physical layout
//	l := layout.Build(g, width, height, layout.WithOrderer(ordering.Barycentric{}))
//
//	// 3. Render to a specific format
//	svg := sink.RenderSVG(l, sink.WithStyle(styles.NewSimple()))
//
// # Subpackages
//
//   - [layout]: The core layout engine that positions blocks based on row orderings.
//   - [ordering]: Algorithms for determining the best horizontal arrangement of blocks.
//   - [sink]: Final output generators for different file formats.
//   - [styles]: Visual themes and drawing primitives.
//   - [transform]: Graph transformations specific to tower visualizations (e.g., merging subdividers).
//   - [feature]: High-level visualization features like Nebraska ranking and brittle detection.
//
// [layout]: github.com/matzehuels/stacktower/pkg/render/tower/layout
// [ordering]: github.com/matzehuels/stacktower/pkg/render/tower/ordering
// [sink]: github.com/matzehuels/stacktower/pkg/render/tower/sink
// [styles]: github.com/matzehuels/stacktower/pkg/render/tower/styles
// [transform]: github.com/matzehuels/stacktower/pkg/render/tower/transform
// [feature]: github.com/matzehuels/stacktower/pkg/render/tower/feature
package tower
