// Package dto provides serialization types (Data Transfer Objects) for dependency graphs and layouts.
//
// This package is the canonical import for all serialization types - no aliases needed.
//
// # Architecture
//
// This package provides the **serialization layer** - the boundary between internal
// representations and external formats (JSON files, API responses, caches).
//
// Key distinction:
//
//   - [Graph], [Layout]: DTOs for serialization (this package)
//   - pkg/core/dag.DAG: Internal graph representation
//   - pkg/core/render/tower/layout.Layout: Internal tower layout (positions, metadata)
//
// The internal layout.Layout stores block positions and rendering metadata.
// The dto.Layout is the serialized form suitable for JSON/API/cache.
// Use ToDTO()/FromDTO() methods to convert between them.
//
// # Core Types
//
// The package defines three primary types:
//
//   - [Graph]: Serialization format for dependency graphs (nodes + edges)
//   - [Layout]: Unified format for visualization layouts (tower or nodelink)
//   - [Node]/[Edge]: Shared types used by both Graph and Layout
//
// # Constants
//
// This package is the single source of truth for all string constants:
//
// Visualization types:
//   - [VizTypeTower]: Tower visualization ("tower")
//   - [VizTypeNodelink]: Node-link visualization ("nodelink")
//
// Visual styles:
//   - [StyleSimple]: Simple geometric style ("simple")
//   - [StyleHanddrawn]: Hand-drawn sketch style ("handdrawn")
//
// Node kinds:
//   - [KindSubdivider]: Subdivider node kind
//   - [KindAuxiliary]: Auxiliary node kind
//
// # Graph Serialization
//
// Graphs use a simple JSON format:
//
//	{
//	  "nodes": [{"id": "app"}, {"id": "lib-a"}],
//	  "edges": [{"from": "app", "to": "lib-a"}]
//	}
//
// Serialization functions:
//
//	g, _ := dto.ReadGraphFile("deps.json")     // File → DAG
//	dto.WriteGraphFile(dag, "output.json")     // DAG → File
//	data, _ := dto.MarshalGraph(dag)           // DAG → []byte
//	graph, _ := dto.UnmarshalGraph(data)       // []byte → Graph DTO
//
// Round-trip fidelity is guaranteed: import → transform → export → re-import
// produces identical results.
//
// # Layout Serialization
//
// Layouts use a discriminated union format (check viz_type):
//
//	layout, _ := dto.UnmarshalLayout(data)
//	if layout.IsTower() {
//	    // Use layout.Blocks for positioned blocks
//	} else {
//	    // Use layout.DOT for Graphviz rendering
//	}
//
// Serialization functions:
//
//	layout, _ := dto.ReadLayoutFile("viz.json")
//	dto.WriteLayoutFile(layout, "output.json")
//
// # Converting Between Types
//
// For tower layouts, convert between internal and DTO forms:
//
//	// Internal → DTO (for serialization)
//	dto, err := internalLayout.ToDTO(graph)
//	dto.WriteLayoutFile(dto, "layout.json")
//
//	// DTO → Internal (from serialization)
//	dtoLayout, _ := dto.ReadLayoutFile("layout.json")
//	internalLayout, err := layout.FromDTO(dtoLayout)
//
// # Node Metadata
//
// The meta object supports arbitrary data. Recognized keys:
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
package dto
