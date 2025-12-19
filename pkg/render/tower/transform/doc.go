// Package transform provides post-layout transformations for tower rendering.
//
// # Overview
//
// After computing block positions with [layout.Build], this package provides
// transformations that modify the layout for improved visual output:
//
//   - [MergeSubdividers]: Combines subdivider blocks into continuous vertical columns
//   - [Randomize]: Applies random width variation for a natural, hand-drawn look
//
// These transformations are applied after layout computation but before
// rendering to SVG/JSON output.
//
// # Merging Subdividers
//
// When long edges are subdivided by [dag/transform.Subdivide], each segment
// becomes a separate block. [MergeSubdividers] combines these into single
// continuous vertical blocks for cleaner visualization:
//
//	layout := layout.Build(g, width, height, opts...)
//	layout = transform.MergeSubdividers(layout, g)
//
// Blocks are grouped by their MasterID (the original node they were split from)
// and their horizontal position. Multiple groups can exist for the same master
// if the subdivider chain splits across different horizontal positions.
//
// # Randomization
//
// [Randomize] applies controlled random variation to block widths, creating
// a checkerboard pattern that mimics hand-drawn diagrams:
//
//	layout = transform.Randomize(layout, g, seed, nil)
//
// The randomization:
//
//   - Shrinks alternating rows to create visual rhythm
//   - Uses a seed for reproducible "randomness"
//   - Ensures minimum overlap between connected blocks
//   - Respects minimum block width and gap constraints
//
// # Options
//
// [Options] configures randomization behavior:
//
//   - WidthShrink: Maximum shrink factor (0-1, default 0.85)
//   - MinBlockWidth: Minimum allowed width (default 30px)
//   - MinGap: Minimum gap between blocks (default 5px)
//   - MinOverlap: Minimum overlap for connected blocks (default 10px)
//
// # Pipeline Position
//
// These transformations fit in the rendering pipeline:
//
//	DAG → dag/transform.Normalize → ordering → layout.Build → [this package] → sink.RenderSVG
//
// [layout.Build]: github.com/matzehuels/stacktower/pkg/render/tower/layout.Build
// [dag/transform.Subdivide]: github.com/matzehuels/stacktower/pkg/dag/transform.Subdivide
package transform
