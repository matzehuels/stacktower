package io

import (
	"encoding/json"
	"fmt"
	"io"

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
