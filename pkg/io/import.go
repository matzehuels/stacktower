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
//
// The input must be a JSON object with "nodes" and "edges" arrays:
//
//	{
//	  "nodes": [{"id": "a"}, {"id": "b"}],
//	  "edges": [{"from": "a", "to": "b"}]
//	}
//
// Each node must have an "id" field. Optional fields:
//   - row: integer layer assignment (defaults to 0)
//   - kind: "subdivider" or "auxiliary" (defaults to normal node)
//   - meta: object with arbitrary key-value pairs
//
// Each edge must have "from" and "to" fields that reference node IDs.
//
// ReadJSON returns an error if:
//   - The JSON is malformed or invalid
//   - A node has a duplicate ID
//   - An edge references an unknown node ID
//   - Adding a node or edge violates DAG constraints (e.g., creates a cycle)
//
// Errors are wrapped with context describing which node or edge caused
// the problem. Use errors.Is or errors.As to check for specific DAG errors.
//
// The returned DAG is independent of r and can be modified safely after
// ReadJSON returns. ReadJSON does not close r.
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
//
// ImportJSON opens the file, decodes it using [ReadJSON], and closes the
// file. If the file cannot be opened, or if decoding fails, ImportJSON
// returns an error describing the failure. The error wraps the underlying
// cause with the file path for context.
//
// ImportJSON returns the same validation errors as [ReadJSON] for malformed
// graphs or DAG constraint violations.
func ImportJSON(path string) (*dag.DAG, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return ReadJSON(f)
}
