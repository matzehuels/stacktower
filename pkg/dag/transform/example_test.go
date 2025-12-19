package transform_test

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/dag/transform"
)

func ExampleNormalize() {
	// Build a raw dependency graph (not yet normalized)
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app"})
	_ = g.AddNode(dag.Node{ID: "auth"})
	_ = g.AddNode(dag.Node{ID: "cache"})
	_ = g.AddNode(dag.Node{ID: "db"})

	// Dependencies: app → auth → db, app → cache → db, app → db (transitive)
	_ = g.AddEdge(dag.Edge{From: "app", To: "auth"})
	_ = g.AddEdge(dag.Edge{From: "app", To: "cache"})
	_ = g.AddEdge(dag.Edge{From: "app", To: "db"}) // Transitive - will be removed
	_ = g.AddEdge(dag.Edge{From: "auth", To: "db"})
	_ = g.AddEdge(dag.Edge{From: "cache", To: "db"})

	fmt.Println("Before normalize:")
	fmt.Println("  Nodes:", g.NodeCount())
	fmt.Println("  Edges:", g.EdgeCount())

	// Normalize: assigns layers, removes transitive edges, subdivides long edges
	transform.Normalize(g)

	fmt.Println("After normalize:")
	fmt.Println("  Nodes:", g.NodeCount())
	fmt.Println("  Edges:", g.EdgeCount())
	fmt.Println("  Rows:", g.RowCount())
	// Output:
	// Before normalize:
	//   Nodes: 4
	//   Edges: 5
	// After normalize:
	//   Nodes: 4
	//   Edges: 4
	//   Rows: 3
}

func ExampleTransitiveReduction() {
	// A → B → C with transitive edge A → C
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A", Row: 0})
	_ = g.AddNode(dag.Node{ID: "B", Row: 1})
	_ = g.AddNode(dag.Node{ID: "C", Row: 2})
	_ = g.AddEdge(dag.Edge{From: "A", To: "B"})
	_ = g.AddEdge(dag.Edge{From: "B", To: "C"})
	_ = g.AddEdge(dag.Edge{From: "A", To: "C"}) // Redundant

	fmt.Println("Before reduction:", g.EdgeCount(), "edges")
	transform.TransitiveReduction(g)
	fmt.Println("After reduction:", g.EdgeCount(), "edges")
	// Output:
	// Before reduction: 3 edges
	// After reduction: 2 edges
}

func ExampleAssignLayers() {
	// Create graph without layer assignments
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app"})  // Will be row 0
	_ = g.AddNode(dag.Node{ID: "lib"})  // Will be row 1
	_ = g.AddNode(dag.Node{ID: "core"}) // Will be row 2
	_ = g.AddEdge(dag.Edge{From: "app", To: "lib"})
	_ = g.AddEdge(dag.Edge{From: "lib", To: "core"})

	transform.AssignLayers(g)

	app, _ := g.Node("app")
	lib, _ := g.Node("lib")
	core, _ := g.Node("core")

	fmt.Println("app row:", app.Row)
	fmt.Println("lib row:", lib.Row)
	fmt.Println("core row:", core.Row)
	// Output:
	// app row: 0
	// lib row: 1
	// core row: 2
}

func ExampleSubdivide() {
	// Create graph with a long edge spanning multiple rows
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "deep", Row: 3}) // 3 rows below app
	_ = g.AddEdge(dag.Edge{From: "app", To: "deep"})

	fmt.Println("Before subdivide:")
	fmt.Println("  Nodes:", g.NodeCount())

	transform.Subdivide(g)

	fmt.Println("After subdivide:")
	fmt.Println("  Nodes:", g.NodeCount())

	// Check that subdivider nodes were created
	subdividers := 0
	for _, n := range g.Nodes() {
		if n.IsSubdivider() {
			subdividers++
		}
	}
	fmt.Println("  Subdividers:", subdividers)
	// Output:
	// Before subdivide:
	//   Nodes: 2
	// After subdivide:
	//   Nodes: 4
	//   Subdividers: 2
}

func ExampleBreakCycles() {
	// Create a graph with a cycle (which shouldn't happen in deps, but might)
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A"})
	_ = g.AddNode(dag.Node{ID: "B"})
	_ = g.AddNode(dag.Node{ID: "C"})
	_ = g.AddEdge(dag.Edge{From: "A", To: "B"})
	_ = g.AddEdge(dag.Edge{From: "B", To: "C"})
	_ = g.AddEdge(dag.Edge{From: "C", To: "A"}) // Creates cycle

	fmt.Println("Edges before:", g.EdgeCount())
	transform.BreakCycles(g)
	fmt.Println("Edges after:", g.EdgeCount())
	// Output:
	// Edges before: 3
	// Edges after: 2
}
