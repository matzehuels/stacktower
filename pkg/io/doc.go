// Package io provides JSON import and export for dependency graphs.
//
// # Overview
//
// Stacktower uses a simple JSON format as its interchange format. This allows:
//
//   - Visualization of any directed graph, not just package dependencies
//   - Integration with external tools that produce or consume graph data
//   - Caching of parsed dependency data for faster re-rendering
//   - Round-trip preservation of layout decisions and render options
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
// # Export
//
// Use [ExportJSON] to write a graph to a file, or [WriteJSON] to write to any
// io.Writer:
//
//	err := io.ExportJSON(g, "output.json")
//
// The export includes all node and edge data, including synthetic nodes
// (subdividers, auxiliaries) and their metadata. This enables round-trip:
// import a graph, render it, export the result, and re-render identically.
//
// # Layout Export
//
// For external tools that need computed positions, use the JSON sink in
// [render/tower/sink] which exports the complete [layout.Layout] including
// block coordinates, row orderings, and all render options.
//
// [render/tower/sink]: github.com/matzehuels/stacktower/pkg/render/tower/sink
// [layout.Layout]: github.com/matzehuels/stacktower/pkg/render/tower/layout.Layout
package io
