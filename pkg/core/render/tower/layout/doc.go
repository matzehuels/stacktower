// Package layout computes block positions for tower visualizations.
//
// # Overview
//
// Once a DAG has been normalized and its rows ordered, this package computes
// the exact pixel coordinates for each block. The layout algorithm produces
// a complete [Layout] containing all information needed for rendering:
//
//   - Block positions (left, right, top, bottom coordinates)
//   - Row orderings (the left-to-right sequence within each layer)
//   - Frame dimensions and margins
//
// # Width Allocation
//
// Block widths are computed based on support relationships—blocks that carry
// more weight (support more nodes above) receive more width. This creates a
// visual hierarchy that reinforces the tower metaphor.
//
// Two width-flow directions are available:
//
//   - Bottom-up (default): Width flows from sinks (foundations) upward.
//     Foundational packages appear wider, supporting narrower blocks above.
//
//   - Top-down: Width flows from roots downward. The application at the top
//     is widest, with dependencies progressively narrower below.
//
// # Height Calculation
//
// Row heights are uniform for regular nodes, with auxiliary rows (containing
// only separator beams) receiving reduced height based on [WithAuxiliaryRatio].
//
// # Building a Layout
//
// Use [Build] with a normalized DAG and frame dimensions:
//
//	l := layout.Build(g, 800, 600,
//	    layout.WithMarginRatio(0.05),
//	)
//
// The default orderer is [ordering.OptimalSearch] with a 60-second timeout.
// For faster but potentially suboptimal layouts, use [ordering.Barycentric]:
//
//	l := layout.Build(g, 800, 600,
//	    layout.WithOrderer(ordering.Barycentric{}),
//	)
//
// The returned [Layout] contains a [Block] for each node with computed
// coordinates ready for rendering.
//
// # Options
//
//   - [WithOrderer]: Algorithm for determining row orderings (default: [ordering.OptimalSearch])
//   - [WithAuxiliaryRatio]: Height ratio for auxiliary-only rows (default 0.2)
//   - [WithMarginRatio]: Frame margin as fraction of dimensions (default 0.05)
//   - [WithTopDownWidths]: Use top-down instead of bottom-up width flow
//
// # Block Coordinates
//
// Each [Block] provides:
//
//   - NodeID: The node this block represents
//   - Left, Right: Horizontal bounds
//   - Bottom, Top: Vertical bounds (origin at top-left, Y increases downward)
//   - MidX, MidY: Center coordinates (for text placement)
//
// # Integration
//
// The layout package sits between ordering and rendering in the pipeline:
//
//	DAG → transform.Normalize → ordering.OrderRows → layout.Build → sink.RenderSVG
//
// Sinks in [render/tower/sink] consume the Layout to produce final output in
// various formats (SVG, JSON, PDF, PNG).
//
// [render/tower/sink]: github.com/matzehuels/stacktower/pkg/core/render/tower/sink
package layout
