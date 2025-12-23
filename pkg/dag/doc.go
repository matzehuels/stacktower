// Package dag provides a directed acyclic graph (DAG) optimized for
// row-based layered layouts used in tower visualizations.
//
// # Overview
//
// Stacktower renders dependency graphs as physical towers where blocks rest
// on what they depend on. This package provides the core data structure that
// organizes nodes into horizontal rows (layers), with edges connecting nodes
// in consecutive rows only.
//
// The row-based constraint is essential for the Sugiyama-style layered graph
// drawing that powers tower visualizations. It enables efficient crossing
// detection and ordering algorithms.
//
// # Basic Usage
//
// Create a new graph with [New], add nodes with [DAG.AddNode], and edges with
// [DAG.AddEdge]. Nodes must have unique IDs, and edges can only connect
// existing nodes in consecutive rows (From.Row+1 == To.Row):
//
//	g := dag.New(nil)
//	g.AddNode(dag.Node{ID: "app", Row: 0})
//	g.AddNode(dag.Node{ID: "lib", Row: 1})
//	g.AddEdge(dag.Edge{From: "app", To: "lib"})
//
// Query the graph structure with [DAG.Children], [DAG.Parents], [DAG.NodesInRow],
// and related methods. Use [DAG.Validate] to verify structural integrity before
// rendering or transformations.
//
// # Node Types
//
// The package supports three node kinds to handle real-world graph structures:
//
//   - [NodeKindRegular]: Original graph vertices from dependency data
//   - [NodeKindSubdivider]: Synthetic nodes that break long edges into segments
//   - [NodeKindAuxiliary]: Helper nodes for layout (e.g., separator beams)
//
// Subdivider nodes maintain a [Node.MasterID] linking back to their origin,
// allowing them to be visually merged into continuous vertical blocks during
// rendering. Auxiliary nodes act as "separator beams" that resolve impossible
// crossing patterns by grouping edges through a shared intermediate.
//
// # Edge Crossings
//
// A key challenge in tower layouts is minimizing (ideally eliminating) edge
// crossings. When two edges cross, the corresponding blocks cannot physically
// support each other in a stacked tower.
//
// The [CountCrossings] and [CountLayerCrossings] functions use a Fenwick tree
// (binary indexed tree) to count inversions in O(E log V) time, enabling
// fast evaluation of millions of candidate orderings during optimization.
//
// # Metadata
//
// Both nodes and the graph itself support arbitrary metadata via [Metadata] maps.
// This is used to store package information (version, description, repository URL)
// and render options (style, seed) that flow through the pipeline. Metadata maps
// are never nil after creation - empty maps are automatically initialized.
//
// # Concurrency
//
// DAG instances are not safe for concurrent use. Callers must synchronize access
// if multiple goroutines read or modify the same graph. Immutable operations like
// counting crossings on a read-only graph can safely run in parallel across
// different goroutines.
//
// # Related Packages
//
// The [transform] subpackage provides graph transformations:
//   - Transitive reduction (remove redundant edges)
//   - Edge subdivision (break long edges into segments)
//   - Span overlap resolution (insert separator beams)
//   - Layer assignment (assign rows based on depth)
//
// The [perm] subpackage provides permutation algorithms including the PQ-tree
// data structure for efficiently generating only valid orderings that preserve
// crossing-free constraints.
//
// [transform]: github.com/matzehuels/stacktower/pkg/dag/transform
// [perm]: github.com/matzehuels/stacktower/pkg/dag/perm
package dag
