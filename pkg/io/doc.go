// Package io provides JSON import and export for directed acyclic graphs (DAGs).
//
// # Overview
//
// This package enables serialization of dependency graphs to and from a simple
// JSON format. The format is designed for:
//
//   - Visualization of any directed graph, not just package dependencies
//   - Integration with external tools that produce or consume graph data
//   - Caching of parsed dependency data for faster re-rendering
//   - Round-trip preservation: import, render, export, and re-import identically
//
// # JSON Format
//
// The format has two required top-level arrays:
//
//	{
//	  "nodes": [
//	    {"id": "app"},
//	    {"id": "lib-a"},
//	    {"id": "lib-b"}
//	  ],
//	  "edges": [
//	    {"from": "app", "to": "lib-a"},
//	    {"from": "lib-a", "to": "lib-b"}
//	  ]
//	}
//
// # Node Fields
//
// Required:
//   - id: Unique string identifier (also used as the display label)
//
// Optional:
//   - row: Pre-assigned layer (computed automatically if omitted)
//   - kind: Internal node type ("subdivider" or "auxiliary")
//   - meta: Freeform object for package metadata
//
// # Metadata Keys
//
// The meta object can contain any data, but certain keys are recognized by
// render features:
//
//   - repo_url: Clickable link for blocks
//   - repo_stars: Star count for popups
//   - repo_owner: Repository owner for Nebraska ranking
//   - repo_maintainers: Maintainer list for Nebraska ranking
//   - repo_last_commit: Last commit date for staleness detection
//   - repo_archived: Whether the repository is archived
//   - summary/description: Displayed in hover popups
//
// # Import
//
// Use [ImportJSON] to read a graph from a file path, or [ReadJSON] to read
// from any io.Reader:
//
//	g, err := io.ImportJSON("deps.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Both functions validate the JSON structure and DAG constraints (no cycles,
// no duplicate node IDs). Errors are wrapped with context about which node or
// edge caused the problem.
//
// # Export
//
// Use [ExportJSON] to write a graph to a file, or [WriteJSON] to write to any
// io.Writer:
//
//	err := io.ExportJSON(g, "output.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// The export includes all node and edge data, including synthetic nodes
// (subdividers, auxiliaries) and their metadata. Row assignments, node kinds,
// and all metadata are preserved. This enables full round-trip fidelity:
// import a graph, transform it, export the result, and re-import identically.
//
// # Concurrency
//
// All functions in this package are safe to call concurrently with other
// readers of the same DAG, but not with concurrent modifications to the DAG.
// The [ReadJSON] and [ImportJSON] functions create independent DAG instances
// that can be used and modified freely after import.
//
// # Layout Export
//
// This package exports the logical graph structure only (nodes, edges, metadata).
// For external tools that need computed layout positions, use the JSON sink in
// [render/tower/sink], which exports the complete [layout.Layout] including
// block coordinates, row orderings, and all render options.
//
// [render/tower/sink]: github.com/matzehuels/stacktower/pkg/render/tower/sink
// [layout.Layout]: github.com/matzehuels/stacktower/pkg/render/tower/layout.Layout
package io
