// Package nodelink provides node-and-edge graph visualization using Graphviz.
//
// This package implements a traditional graph visualization where nodes are
// represented as boxes and dependencies are shown as directed edges between them.
//
// # Architecture
//
// Unlike the tower visualization which separates layout computation from rendering,
// nodelink uses Graphviz which handles both in a single step:
//
//	Tower:    DAG → layout.Build() → Layout → sink.RenderSVG() → SVG
//	Nodelink: DAG → ToDOT() → DOT → RenderSVG() → SVG
//
// The DOT format serves as the intermediate representation (similar to tower's
// layout.json), enabling re-rendering without re-parsing the dependency graph.
//
// # Pipeline Integration
//
// In the job system, nodelink follows the same three-stage pattern:
//
//	Parse:  manifest/package → graph.json (DAG)
//	Layout: graph.json → layout.dot (Graphviz DOT format)
//	Export: layout.dot → svg/png/pdf
//
// # Layout Engines
//
// Graphviz provides several layout engines via the Engine option:
//
//   - dot: Hierarchical (default) - best for dependency graphs
//   - neato: Spring model - for undirected graphs
//   - fdp: Force-directed - for clustering
//   - circo: Circular - for cyclic structures
//   - twopi: Radial - for tree-like graphs
//
// # Usage
//
// Direct usage (bypassing the job system):
//
//	g, _ := resolver.Resolve(ctx, "requests", opts)
//	dot := nodelink.ToDOT(g, nodelink.Options{})
//	svg, _ := nodelink.RenderSVG(dot)
//
// Via the API:
//
//	// Step 1: Parse
//	POST /api/v1/parse {language: "python", package: "requests"}
//	// → graph.json
//
//	// Step 2: Layout
//	POST /api/v1/layout {graph_path: "job-123/graph.json", viz_type: "nodelink"}
//	// → layout.dot
//
//	// Step 3: Export
//	POST /api/v1/export {layout_path: "job-456/layout.dot", formats: ["svg"]}
//	// → nodelink.svg
//
// # Subdividers
//
// Subdivider nodes (created by dag/transform.Subdivide) are rendered with dashed
// outlines and grey fill to visually distinguish them from regular dependency nodes.
package nodelink
