package ordering_test

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/ordering"
)

func ExampleBarycentric() {
	// Create a graph with potential for crossings
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "auth", Row: 1})
	_ = g.AddNode(dag.Node{ID: "cache", Row: 1})
	_ = g.AddNode(dag.Node{ID: "db", Row: 2})
	_ = g.AddEdge(dag.Edge{From: "app", To: "auth"})
	_ = g.AddEdge(dag.Edge{From: "app", To: "cache"})
	_ = g.AddEdge(dag.Edge{From: "auth", To: "db"})
	_ = g.AddEdge(dag.Edge{From: "cache", To: "db"})

	// Barycentric orderer with 24 refinement passes
	orderer := ordering.Barycentric{Passes: 24}
	orders := orderer.OrderRows(g)

	fmt.Println("Row count:", len(orders))
	fmt.Println("Row 1 has 2 nodes:", len(orders[1]) == 2)
	// Output:
	// Row count: 3
	// Row 1 has 2 nodes: true
}

func ExampleBarycentric_crossingMinimization() {
	// Classic crossing example: X pattern
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 0})
	_ = g.AddNode(dag.Node{ID: "x", Row: 1})
	_ = g.AddNode(dag.Node{ID: "y", Row: 1})

	// Create crossing edges: a→y, b→x
	_ = g.AddEdge(dag.Edge{From: "a", To: "y"})
	_ = g.AddEdge(dag.Edge{From: "b", To: "x"})

	// Before ordering, this creates a crossing
	initialCrossings := dag.CountLayerCrossings(g, []string{"a", "b"}, []string{"x", "y"})
	fmt.Println("Initial crossings:", initialCrossings)

	// Barycentric ordering eliminates the crossing
	orderer := ordering.Barycentric{}
	orders := orderer.OrderRows(g)

	finalCrossings := dag.CountLayerCrossings(g, orders[0], orders[1])
	fmt.Println("After ordering:", finalCrossings)
	// Output:
	// Initial crossings: 1
	// After ordering: 0
}

func ExampleOptimalSearch() {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "a", Row: 1})
	_ = g.AddNode(dag.Node{ID: "b", Row: 1})
	_ = g.AddNode(dag.Node{ID: "c", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "app", To: "a"})
	_ = g.AddEdge(dag.Edge{From: "app", To: "b"})
	_ = g.AddEdge(dag.Edge{From: "app", To: "c"})

	// Optimal search with progress callback
	var explored int
	orderer := ordering.OptimalSearch{
		Timeout: ordering.DefaultTimeoutFast,
		Progress: func(exp, pruned, best int) {
			explored = exp
		},
	}
	orders := orderer.OrderRows(g)

	fmt.Println("Found ordering:", len(orders) > 0)
	fmt.Println("Explored states:", explored >= 0)
	// Output:
	// Found ordering: true
	// Explored states: true
}

func ExampleQuality() {
	// Quality presets provide sensible timeout defaults
	fmt.Println("Fast timeout:", ordering.DefaultTimeoutFast)
	fmt.Println("Balanced timeout:", ordering.DefaultTimeoutBalanced)
	fmt.Println("Optimal timeout:", ordering.DefaultTimeoutOptimal)
	// Output:
	// Fast timeout: 100ms
	// Balanced timeout: 5s
	// Optimal timeout: 1m0s
}

func Example_ordererInterface() {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "lib", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "app", To: "lib"})

	// Both orderers implement the same interface
	var orderer ordering.Orderer

	// Fast heuristic for interactive use
	orderer = ordering.Barycentric{Passes: 12}
	fastOrders := orderer.OrderRows(g)

	// Exact algorithm for publication quality
	orderer = ordering.OptimalSearch{Timeout: ordering.DefaultTimeoutFast}
	optimalOrders := orderer.OrderRows(g)

	fmt.Println("Fast result rows:", len(fastOrders))
	fmt.Println("Optimal result rows:", len(optimalOrders))
	// Output:
	// Fast result rows: 2
	// Optimal result rows: 2
}
