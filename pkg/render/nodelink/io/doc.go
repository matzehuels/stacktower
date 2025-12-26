// Package io provides types for nodelink layout data.
//
// Note: For nodelink visualization, the DOT format (stored as layout.dot) serves
// as the layout artifact. Graphviz handles both layout computation and rendering
// in a single step, so there's no separate JSON layout format like tower uses.
//
// This package provides types that could be used if you wanted to:
//   - Extract computed positions from graphviz output
//   - Store positions in a JSON format for external tools
//   - Implement custom rendering without graphviz
//
// For the standard workflow, see [github.com/matzehuels/stacktower/pkg/render/nodelink].
package io

// VizType identifier for nodelink layouts.
const VizType = "nodelink"
