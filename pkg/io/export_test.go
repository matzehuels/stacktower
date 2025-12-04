package io

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/matzehuels/stacktower/pkg/dag"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name      string
		build     func() *dag.DAG
		wantNodes int
		wantEdges int
		check     func(t *testing.T, g graph)
	}{
		{
			name:      "Empty",
			build:     func() *dag.DAG { return dag.New(nil) },
			wantNodes: 0,
			wantEdges: 0,
		},
		{
			name: "Simple",
			build: func() *dag.DAG {
				g := dag.New(nil)
				g.AddNode(dag.Node{ID: "a", Meta: dag.Metadata{"version": "1.0"}})
				g.AddNode(dag.Node{ID: "b", Meta: dag.Metadata{"version": "2.0"}})
				g.AddEdge(dag.Edge{From: "a", To: "b"})
				return g
			},
			wantNodes: 2,
			wantEdges: 1,
		},
		{
			name: "PreservesMetadata",
			build: func() *dag.DAG {
				g := dag.New(nil)
				g.AddNode(dag.Node{
					ID: "test",
					Meta: dag.Metadata{
						"version": "1.0",
						"author":  "test-author",
					},
				})
				return g
			},
			wantNodes: 1,
			wantEdges: 0,
			check: func(t *testing.T, g graph) {
				if g.Nodes[0].Meta["version"] != "1.0" {
					t.Errorf("version = %v, want 1.0", g.Nodes[0].Meta["version"])
				}
				if g.Nodes[0].Meta["author"] != "test-author" {
					t.Errorf("author = %v, want test-author", g.Nodes[0].Meta["author"])
				}
			},
		},
		{
			name: "Diamond",
			build: func() *dag.DAG {
				g := dag.New(nil)
				g.AddNode(dag.Node{ID: "a"})
				g.AddNode(dag.Node{ID: "b"})
				g.AddNode(dag.Node{ID: "c"})
				g.AddNode(dag.Node{ID: "d"})
				g.AddEdge(dag.Edge{From: "a", To: "b"})
				g.AddEdge(dag.Edge{From: "a", To: "c"})
				g.AddEdge(dag.Edge{From: "b", To: "d"})
				g.AddEdge(dag.Edge{From: "c", To: "d"})
				return g
			},
			wantNodes: 4,
			wantEdges: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.build()

			var buf bytes.Buffer
			if err := WriteJSON(g, &buf); err != nil {
				t.Fatalf("WriteJSON: %v", err)
			}

			var result graph
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if got := len(result.Nodes); got != tt.wantNodes {
				t.Errorf("nodes = %d, want %d", got, tt.wantNodes)
			}
			if got := len(result.Edges); got != tt.wantEdges {
				t.Errorf("edges = %d, want %d", got, tt.wantEdges)
			}

			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}
