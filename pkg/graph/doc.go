// Package graph provides serialization types for dependency graphs and layouts.
//
// This package defines the canonical wire format for Stacktower's graph data,
// used for JSON files, API responses, caching, and cross-tool interoperability.
//
// # Architecture
//
// The package sits at the serialization boundary between internal representations
// and external formats:
//
//   - [Graph], [Layout]: Serialization types (this package)
//   - pkg/core/dag.DAG: Internal graph representation
//   - pkg/core/render/tower/layout.Layout: Internal layout (positions, metadata)
//
// Use [FromDAG]/[ToDAG] and Export/Parse methods to convert between them.
//
// # Core Types
//
//   - [Graph]: Node-link format for dependency graphs
//   - [Layout]: Unified format for visualization layouts (tower or nodelink)
//   - [Node], [Edge]: Shared structural types
//   - [Block]: Positioned element in tower visualizations
//
// # Constants
//
// This package is the single source of truth for visualization constants:
//
//	graph.VizTypeTower      // "tower"
//	graph.VizTypeNodelink   // "nodelink"
//	graph.StyleSimple       // "simple"
//	graph.StyleHanddrawn    // "handdrawn"
//
// # Graph Serialization
//
// Graphs use a simple node-link JSON format:
//
//	{
//	  "nodes": [{"id": "app"}, {"id": "lib-a"}],
//	  "edges": [{"from": "app", "to": "lib-a"}]
//	}
//
// Common operations:
//
//	g, _ := graph.ReadGraphFile("deps.json")    // File → DAG
//	graph.WriteGraphFile(dag, "output.json")    // DAG → File
//	data, _ := graph.MarshalGraph(dag)          // DAG → []byte
//	parsed, _ := graph.UnmarshalGraph(data)     // []byte → Graph
//
// # Layout Serialization
//
// Layouts are discriminated by VizType:
//
//	layout, _ := graph.UnmarshalLayout(data)
//	if layout.IsTower() {
//	    // Use layout.Blocks for positioned blocks
//	} else {
//	    // Use layout.DOT for Graphviz rendering
//	}
//
// # Converting Between Types
//
// For tower layouts:
//
//	// Internal → Serialized (for JSON/API/cache)
//	serialized, err := internalLayout.Export(dag)
//
//	// Serialized → Internal (from JSON/API/cache)
//	internal, err := layout.Parse(serializedLayout)
//
// # Node Metadata
//
// The meta object supports arbitrary key-value data. Recognized keys:
//
//	repo_url          Repository URL (clickable blocks)
//	repo_stars        Star count (popups)
//	repo_owner        Owner for Nebraska ranking
//	repo_maintainers  Maintainer list
//	repo_last_commit  Staleness detection
//	repo_archived     Archived flag
//	description       Popup content
//
// # Concurrency
//
// All functions are safe for concurrent reads but not concurrent writes.
package graph
