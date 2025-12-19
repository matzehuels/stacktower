package io

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

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
// The output includes all nodes (with metadata and kind) and edges.
// This format can be re-imported with [ReadJSON] for round-trip processing.
func WriteJSON(g *dag.DAG, w io.Writer) error {
	out := graph{
		Nodes: make([]node, len(g.Nodes())),
		Edges: make([]edge, len(g.Edges())),
	}

	for i, n := range g.Nodes() {
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
// This is a convenience wrapper around [WriteJSON] for file-based output.
func ExportJSON(g *dag.DAG, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()
	return WriteJSON(g, f)
}
