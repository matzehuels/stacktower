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

func ExampleDAG_Validate() {
	// Validate checks for consecutive rows and cycles
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 1})
	_ = g.AddNode(dag.Node{ID: "c", Row: 2})
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})
	_ = g.AddEdge(dag.Edge{From: "b", To: "c"})

	if err := g.Validate(); err != nil {
		fmt.Println("Invalid:", err)
	} else {
		fmt.Println("Valid DAG")
	}
	// Output:
	// Valid DAG
}

func ExampleDAG_Validate_nonConsecutive() {
	// Edges must connect consecutive rows
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 2}) // skips row 1
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})

	if err := g.Validate(); err != nil {
		fmt.Println("Error:", err)
	}
	// Output:
	// Error: edges must connect consecutive rows
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

func ExampleCountCrossings() {
	// Count total crossings across all row pairs
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A", Row: 0})
	_ = g.AddNode(dag.Node{ID: "B", Row: 0})
	_ = g.AddNode(dag.Node{ID: "C", Row: 1})
	_ = g.AddNode(dag.Node{ID: "D", Row: 1})
	_ = g.AddNode(dag.Node{ID: "E", Row: 2})
	_ = g.AddNode(dag.Node{ID: "F", Row: 2})

	// Create a crossing pattern
	_ = g.AddEdge(dag.Edge{From: "A", To: "D"})
	_ = g.AddEdge(dag.Edge{From: "B", To: "C"})
	_ = g.AddEdge(dag.Edge{From: "C", To: "F"})
	_ = g.AddEdge(dag.Edge{From: "D", To: "E"})

	orders := map[int][]string{
		0: {"A", "B"},
		1: {"C", "D"},
		2: {"E", "F"},
	}

	total := dag.CountCrossings(g, orders)
	fmt.Println("Total crossings:", total)
	// Output:
	// Total crossings: 2
}

func ExamplePosMap() {
	// Convert a node ordering to a position lookup map
	ordering := []string{"app", "lib", "core"}
	positions := dag.PosMap(ordering)

	fmt.Println("Position of 'lib':", positions["lib"])
	fmt.Println("Position of 'core':", positions["core"])
	// Output:
	// Position of 'lib': 1
	// Position of 'core': 2
}

func ExampleDAG_ChildrenInRow() {
	// Query children in a specific row
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 1})
	_ = g.AddNode(dag.Node{ID: "c", Row: 2})
	_ = g.AddNode(dag.Node{ID: "d", Row: 2})
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})
	_ = g.AddEdge(dag.Edge{From: "a", To: "c"}) // skips row 1
	_ = g.AddEdge(dag.Edge{From: "a", To: "d"}) // skips row 1

	// Find children specifically in row 2
	childrenInRow2 := g.ChildrenInRow("a", 2)
	fmt.Println("Children in row 2:", len(childrenInRow2))
	// Output:
	// Children in row 2: 2
}

func ExampleNewCrossingWorkspace() {
	// Reuse a workspace for efficient crossing calculations
	// Determine maximum row width in your graph
	maxWidth := 10

	// Create a workspace sized for that maximum
	ws := dag.NewCrossingWorkspace(maxWidth)

	// Now use ws with CountCrossingsIdx for optimization loops
	// (typically used internally by ordering algorithms)
	_ = ws
}
