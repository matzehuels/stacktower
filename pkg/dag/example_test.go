package dag_test

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/dag"
)

func ExampleDAG_basic() {
	// Create a simple dependency graph: app → lib → core
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "lib", Row: 1})
	_ = g.AddNode(dag.Node{ID: "core", Row: 2})
	_ = g.AddEdge(dag.Edge{From: "app", To: "lib"})
	_ = g.AddEdge(dag.Edge{From: "lib", To: "core"})

	fmt.Println("Nodes:", g.NodeCount())
	fmt.Println("Edges:", g.EdgeCount())
	fmt.Println("Rows:", g.RowCount())
	// Output:
	// Nodes: 3
	// Edges: 2
	// Rows: 3
}

func ExampleDAG_traversal() {
	// Build a graph with fan-out: app depends on auth and cache
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "auth", Row: 1})
	_ = g.AddNode(dag.Node{ID: "cache", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "app", To: "auth"})
	_ = g.AddEdge(dag.Edge{From: "app", To: "cache"})

	// Query relationships
	fmt.Println("Children of app:", g.Children("app"))
	fmt.Println("Parents of auth:", g.Parents("auth"))
	fmt.Println("Out-degree of app:", g.OutDegree("app"))
	// Output:
	// Children of app: [auth cache]
	// Parents of auth: [app]
	// Out-degree of app: 2
}

func ExampleDAG_Sources() {
	// Find root nodes (packages with no dependencies above them)
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "cli", Row: 0})
	_ = g.AddNode(dag.Node{ID: "shared", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "app", To: "shared"})
	_ = g.AddEdge(dag.Edge{From: "cli", To: "shared"})

	sources := g.Sources()
	fmt.Println("Source count:", len(sources))
	// Output:
	// Source count: 2
}

func ExampleDAG_metadata() {
	// Attach package metadata to nodes
	g := dag.New(dag.Metadata{"name": "my-project"})
	_ = g.AddNode(dag.Node{
		ID:  "fastapi",
		Row: 0,
		Meta: dag.Metadata{
			"version":     "0.100.0",
			"description": "FastAPI framework",
			"repo_stars":  70000,
		},
	})

	node, _ := g.Node("fastapi")
	fmt.Println("Package:", node.ID)
	fmt.Println("Version:", node.Meta["version"])
	// Output:
	// Package: fastapi
	// Version: 0.100.0
}

func ExampleNode_synthetic() {
	// Synthetic nodes are created during graph transformation
	regular := dag.Node{ID: "lib", Kind: dag.NodeKindRegular}
	subdivider := dag.Node{ID: "lib_sub_1", Kind: dag.NodeKindSubdivider, MasterID: "lib"}
	auxiliary := dag.Node{ID: "Sep_1_a_b", Kind: dag.NodeKindAuxiliary}

	fmt.Println("Regular is synthetic:", regular.IsSynthetic())
	fmt.Println("Subdivider is synthetic:", subdivider.IsSynthetic())
	fmt.Println("Subdivider effective ID:", subdivider.EffectiveID())
	fmt.Println("Auxiliary is synthetic:", auxiliary.IsSynthetic())
	// Output:
	// Regular is synthetic: false
	// Subdivider is synthetic: true
	// Subdivider effective ID: lib
	// Auxiliary is synthetic: true
}

func ExampleCountLayerCrossings() {
	// Count edge crossings between two rows
	// This uses a Fenwick tree for O(E log V) performance
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 0})
	_ = g.AddNode(dag.Node{ID: "x", Row: 1})
	_ = g.AddNode(dag.Node{ID: "y", Row: 1})

	// Create crossing edges: a→y, b→x (these cross when a is left of b)
	_ = g.AddEdge(dag.Edge{From: "a", To: "y"})
	_ = g.AddEdge(dag.Edge{From: "b", To: "x"})

	upper := []string{"a", "b"}
	lower := []string{"x", "y"}
	crossings := dag.CountLayerCrossings(g, upper, lower)
	fmt.Println("Crossings:", crossings)

	// Reorder to eliminate crossing
	upper = []string{"b", "a"}
	crossings = dag.CountLayerCrossings(g, upper, lower)
	fmt.Println("After reorder:", crossings)
	// Output:
	// Crossings: 1
	// After reorder: 0
}
