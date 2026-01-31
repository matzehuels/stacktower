package graph

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matzehuels/stacktower/pkg/core/dag"
)

func TestMarshalGraph(t *testing.T) {
	tests := []struct {
		name      string
		build     func() *dag.DAG
		wantNodes int
		wantEdges int
		check     func(t *testing.T, g Graph)
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
			check: func(t *testing.T, g Graph) {
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

			data, err := MarshalGraph(g)
			if err != nil {
				t.Fatalf("MarshalGraph: %v", err)
			}

			var result Graph
			if err := json.Unmarshal(data, &result); err != nil {
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

func TestReadGraph(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNodes int
		wantEdges int
		wantErr   bool
		check     func(t *testing.T, g *dag.DAG)
	}{
		{
			name: "Valid",
			input: `{
				"nodes": [
					{"id": "A", "meta": {"version": "1.0"}},
					{"id": "B"}
				],
				"edges": [
					{"from": "A", "to": "B"}
				]
			}`,
			wantNodes: 2,
			wantEdges: 1,
			check: func(t *testing.T, g *dag.DAG) {
				n, ok := g.Node("A")
				if !ok {
					t.Fatal("node A not found")
				}
				if n.Meta["version"] != "1.0" {
					t.Errorf("version = %v, want 1.0", n.Meta["version"])
				}
			},
		},
		{
			name: "Empty",
			input: `{
				"nodes": [],
				"edges": []
			}`,
			wantNodes: 0,
			wantEdges: 0,
		},
		{
			name:    "Invalid",
			input:   `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			g, err := ReadGraph(r)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ReadGraph: %v", err)
			}

			if got := g.NodeCount(); got != tt.wantNodes {
				t.Errorf("nodes = %d, want %d", got, tt.wantNodes)
			}
			if got := g.EdgeCount(); got != tt.wantEdges {
				t.Errorf("edges = %d, want %d", got, tt.wantEdges)
			}

			if tt.check != nil {
				tt.check(t, g)
			}
		})
	}
}

func TestReadGraphFile(t *testing.T) {
	content := `{
		"nodes": [{"id": "A"}],
		"edges": []
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	g, err := ReadGraphFile(path)
	if err != nil {
		t.Fatalf("ReadGraphFile: %v", err)
	}

	if g.NodeCount() != 1 {
		t.Errorf("nodes = %d, want 1", g.NodeCount())
	}
}

func TestReadGraphFileNotFound(t *testing.T) {
	_, err := ReadGraphFile("nonexistent.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestWriteGraph(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a"})
	g.AddNode(dag.Node{ID: "b"})
	g.AddEdge(dag.Edge{From: "a", To: "b"})

	var buf bytes.Buffer
	if err := WriteGraph(g, &buf); err != nil {
		t.Fatalf("WriteGraph: %v", err)
	}

	var result Graph
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(result.Nodes) != 2 {
		t.Errorf("nodes = %d, want 2", len(result.Nodes))
	}
}
