package nodelink

import (
	"strings"
	"testing"

	"stacktower/pkg/dag"
)

func TestToDOT(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *dag.DAG
		opts     Options
		contains []string
	}{
		{
			name:     "EmptyGraph",
			setup:    func() *dag.DAG { return dag.New(nil) },
			opts:     Options{},
			contains: []string{"digraph G"},
		},
		{
			name: "SimpleChain",
			setup: func() *dag.DAG {
				g := dag.New(nil)
				_ = g.AddNode(dag.Node{ID: "a"})
				_ = g.AddNode(dag.Node{ID: "b"})
				_ = g.AddEdge(dag.Edge{From: "a", To: "b"})
				return g
			},
			opts:     Options{},
			contains: []string{`"a"`, `"b"`, `"a" -> "b"`},
		},
		{
			name: "Detailed",
			setup: func() *dag.DAG {
				g := dag.New(nil)
				_ = g.AddNode(dag.Node{
					ID:   "pkg",
					Meta: dag.Metadata{"name": "Package Name"},
					Row:  5,
				})
				return g
			},
			opts:     Options{Detailed: true},
			contains: []string{"Package Name", "row: 5"},
		},
		{
			name: "SubdividerNode",
			setup: func() *dag.DAG {
				g := dag.New(nil)
				_ = g.AddNode(dag.Node{ID: "sub", Kind: dag.NodeKindSubdivider})
				return g
			},
			opts:     Options{},
			contains: []string{`style="rounded,filled,dashed"`, "fillcolor=lightgrey"},
		},
		{
			name: "AuxiliaryNode",
			setup: func() *dag.DAG {
				g := dag.New(nil)
				_ = g.AddNode(dag.Node{ID: "aux", Kind: dag.NodeKindAuxiliary})
				return g
			},
			opts:     Options{},
			contains: []string{`"aux"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.setup()
			dot := ToDOT(g, tt.opts)

			for _, s := range tt.contains {
				if !strings.Contains(dot, s) {
					t.Errorf("expected DOT to contain %q", s)
				}
			}
		})
	}
}

func TestRenderSVG(t *testing.T) {
	tests := []struct {
		name    string
		dot     string
		wantErr bool
	}{
		{
			name:    "ValidDOT",
			dot:     `digraph G { "a" -> "b"; }`,
			wantErr: false,
		},
		{
			name:    "InvalidDOT",
			dot:     "not valid DOT {",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svg, err := RenderSVG(tt.dot)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}

			if err != nil {
				t.Skipf("graphviz not available: %v", err)
			}

			if !strings.Contains(string(svg), "svg") {
				t.Error("expected SVG output")
			}
		})
	}
}
