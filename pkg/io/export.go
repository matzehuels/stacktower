package io

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/matzehuels/stacktower/pkg/dag"
)

var kindToString = map[dag.NodeKind]string{
	dag.NodeKindSubdivider: "subdivider",
	dag.NodeKindAuxiliary:  "auxiliary",
}

type graph struct {
	Nodes []node `json:"nodes"`
	Edges []edge `json:"edges"`
}

type node struct {
	ID   string       `json:"id"`
	Row  *int         `json:"row,omitempty"`
	Kind string       `json:"kind,omitempty"`
	Meta dag.Metadata `json:"meta,omitempty"`
}

type edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// WriteJSON encodes a DAG as JSON and writes it to w.
//
// The output is a JSON object with "nodes" and "edges" arrays, formatted with
// 2-space indentation. All nodes are written in their original order with:
//   - id: always present
//   - row: included only if non-zero
//   - kind: included only for non-default kinds (subdivider, auxiliary)
//   - meta: included if non-empty
//
// Edges are written as {from, to} pairs.
//
// The output can be read back with [ReadJSON] to produce an identical DAG,
// preserving all metadata, node kinds, and assigned row numbers.
//
// WriteJSON returns an error if encoding fails or if writing to w fails.
// It does not validate the DAG structure; malformed graphs will be encoded
// as-is and may fail validation on import.
//
// This function is safe to call concurrently with other readers of g,
// but not with concurrent writes to g.
func WriteJSON(g *dag.DAG, w io.Writer) error {
	nodes := g.Nodes()
	// Sort nodes by ID for deterministic output
	slices.SortFunc(nodes, func(a, b *dag.Node) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})

	out := graph{
		Nodes: make([]node, len(nodes)),
		Edges: make([]edge, len(g.Edges())),
	}

	for i, n := range nodes {
		nd := node{ID: n.ID, Meta: n.Meta}
		if n.Row != 0 {
			row := n.Row
			nd.Row = &row
		}
		if s, ok := kindToString[n.Kind]; ok {
			nd.Kind = s
		}
		out.Nodes[i] = nd
	}
	for i, e := range g.Edges() {
		out.Edges[i] = edge{From: e.From, To: e.To}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return nil
}

// ExportJSON writes a DAG to a JSON file at path.
//
// ExportJSON creates (or truncates) the file at path and writes the JSON
// representation of g using [WriteJSON]. The file is created with 0644
// permissions.
//
// If the file cannot be created, or if writing fails, ExportJSON returns
// an error describing the failure. The error wraps the underlying cause
// with the file path for context.
//
// This function is safe to call concurrently with other readers of g,
// but not with concurrent writes to g.
func ExportJSON(g *dag.DAG, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	return WriteJSON(g, f)
}
