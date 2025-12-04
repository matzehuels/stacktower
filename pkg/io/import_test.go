package io

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matzehuels/stacktower/pkg/dag"
)

func TestReadJSON(t *testing.T) {
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
			g, err := ReadJSON(r)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ReadJSON: %v", err)
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

func TestImportJSON(t *testing.T) {
	content := `{
		"nodes": [{"id": "A"}],
		"edges": []
	}`

	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	g, err := ImportJSON(path)
	if err != nil {
		t.Fatalf("ImportJSON: %v", err)
	}

	if g.NodeCount() != 1 {
		t.Errorf("nodes = %d, want 1", g.NodeCount())
	}
}

func TestImportJSONNotFound(t *testing.T) {
	_, err := ImportJSON("nonexistent.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
