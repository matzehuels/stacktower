package io

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/matzehuels/stacktower/pkg/dag"
)

var kindFromString = map[string]dag.NodeKind{
	"subdivider": dag.NodeKindSubdivider,
	"auxiliary":  dag.NodeKindAuxiliary,
}

// ReadJSON decodes a JSON graph from r into a DAG.
// The JSON must have "nodes" and "edges" arrays. Node "kind" and "row"
// fields are optional; missing values use defaults.
func ReadJSON(r io.Reader) (*dag.DAG, error) {
	var data graph
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	g := dag.New(nil)
	for _, n := range data.Nodes {
		nd := dag.Node{ID: n.ID, Meta: n.Meta}
		if n.Row != nil {
			nd.Row = *n.Row
		}
		if k, ok := kindFromString[n.Kind]; ok {
			nd.Kind = k
		}
		if err := g.AddNode(nd); err != nil {
			return nil, fmt.Errorf("node %s: %w", n.ID, err)
		}
	}
	for _, e := range data.Edges {
		if err := g.AddEdge(dag.Edge{From: e.From, To: e.To}); err != nil {
			return nil, fmt.Errorf("edge %s->%s: %w", e.From, e.To, err)
		}
	}

	return g, nil
}

// ImportJSON reads a JSON file at path and returns the decoded DAG.
// This is a convenience wrapper around [ReadJSON] for file-based input.
func ImportJSON(path string) (*dag.DAG, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return ReadJSON(f)
}
