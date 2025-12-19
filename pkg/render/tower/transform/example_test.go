package transform_test

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
	"github.com/matzehuels/stacktower/pkg/render/tower/transform"
)

func ExampleMergeSubdividers() {
	// Create a graph with subdivider nodes (from edge subdivision)
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0, Kind: dag.NodeKindRegular})
	_ = g.AddNode(dag.Node{ID: "lib", Row: 3, Kind: dag.NodeKindRegular})

	// Subdivider nodes break the long edge into segments
	_ = g.AddNode(dag.Node{
		ID:       "lib#1",
		Row:      1,
		Kind:     dag.NodeKindSubdivider,
		MasterID: "lib",
	})
	_ = g.AddNode(dag.Node{
		ID:       "lib#2",
		Row:      2,
		Kind:     dag.NodeKindSubdivider,
		MasterID: "lib",
	})

	_ = g.AddEdge(dag.Edge{From: "app", To: "lib#1"})
	_ = g.AddEdge(dag.Edge{From: "lib#1", To: "lib#2"})
	_ = g.AddEdge(dag.Edge{From: "lib#2", To: "lib"})

	// Create a layout with separate blocks for each subdivider
	l := layout.Layout{
		FrameWidth:  800,
		FrameHeight: 600,
		Blocks: map[string]layout.Block{
			"app":   {NodeID: "app", Left: 100, Right: 200, Top: 50, Bottom: 0},
			"lib#1": {NodeID: "lib#1", Left: 100, Right: 200, Top: 150, Bottom: 100},
			"lib#2": {NodeID: "lib#2", Left: 100, Right: 200, Top: 250, Bottom: 200},
			"lib":   {NodeID: "lib", Left: 100, Right: 200, Top: 350, Bottom: 300},
		},
		RowOrders: map[int][]string{
			0: {"app"},
			1: {"lib#1"},
			2: {"lib#2"},
			3: {"lib"},
		},
	}

	// Merge subdividers into a single continuous block
	merged := transform.MergeSubdividers(l, g)

	// The subdivider segments are now merged
	fmt.Printf("Original blocks: %d\n", len(l.Blocks))
	fmt.Printf("Merged blocks: %d\n", len(merged.Blocks))
	fmt.Printf("Subdividers removed from row orders: %v\n", len(merged.RowOrders[1]) == 0)
	// Output:
	// Original blocks: 4
	// Merged blocks: 2
	// Subdividers removed from row orders: true
}

func ExampleRandomize() {
	// Create a simple layout
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "lib", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "app", To: "lib"})

	l := layout.Layout{
		FrameWidth:  800,
		FrameHeight: 400,
		Blocks: map[string]layout.Block{
			"app": {NodeID: "app", Left: 100, Right: 300, Top: 100, Bottom: 0},
			"lib": {NodeID: "lib", Left: 100, Right: 300, Top: 300, Bottom: 200},
		},
		RowOrders: map[int][]string{
			0: {"app"},
			1: {"lib"},
		},
	}

	// Apply randomization with a fixed seed for reproducibility
	randomized := transform.Randomize(l, g, 42, nil)

	// Block widths are now varied (but deterministic with same seed)
	// Note: Row 0 is not randomized (the root), only subsequent rows are
	appWidth := randomized.Blocks["app"].Right - randomized.Blocks["app"].Left
	libWidth := randomized.Blocks["lib"].Right - randomized.Blocks["lib"].Left

	fmt.Printf("Randomization applied\n")
	fmt.Printf("App width changed: %v\n", appWidth != 200)
	fmt.Printf("Lib width changed: %v\n", libWidth != 200)
	// Output:
	// Randomization applied
	// App width changed: false
	// Lib width changed: true
}

func ExampleRandomize_customOptions() {
	// Create a layout
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "service", Row: 0})
	_ = g.AddNode(dag.Node{ID: "database", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "service", To: "database"})

	l := layout.Layout{
		FrameWidth:  800,
		FrameHeight: 400,
		Blocks: map[string]layout.Block{
			"service":  {NodeID: "service", Left: 100, Right: 400, Top: 100, Bottom: 0},
			"database": {NodeID: "database", Left: 100, Right: 400, Top: 300, Bottom: 200},
		},
		RowOrders: map[int][]string{
			0: {"service"},
			1: {"database"},
		},
	}

	// Use custom options for more aggressive randomization
	opts := &transform.Options{
		WidthShrink:   0.7,  // More shrinking (default 0.85)
		MinBlockWidth: 50.0, // Larger minimum (default 30)
		MinGap:        10.0, // Larger gaps (default 5)
		MinOverlap:    20.0, // More overlap required (default 10)
	}

	randomized := transform.Randomize(l, g, 123, opts)

	fmt.Printf("Custom randomization applied\n")
	fmt.Printf("Blocks modified: %d\n", len(randomized.Blocks))
	// Output:
	// Custom randomization applied
	// Blocks modified: 2
}

func ExampleRandomize_deterministicWithSeed() {
	// Demonstrate that the same seed produces identical results
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})

	l := layout.Layout{
		FrameWidth:  800,
		FrameHeight: 400,
		Blocks: map[string]layout.Block{
			"a": {NodeID: "a", Left: 100, Right: 300, Top: 100, Bottom: 0},
			"b": {NodeID: "b", Left: 100, Right: 300, Top: 300, Bottom: 200},
		},
		RowOrders: map[int][]string{
			0: {"a"},
			1: {"b"},
		},
	}

	// Apply randomization twice with the same seed
	result1 := transform.Randomize(l, g, 999, nil)
	result2 := transform.Randomize(l, g, 999, nil)

	// Results are identical
	width1 := result1.Blocks["a"].Right - result1.Blocks["a"].Left
	width2 := result2.Blocks["a"].Right - result2.Blocks["a"].Left

	fmt.Printf("Same seed produces identical results: %v\n", width1 == width2)
	// Output:
	// Same seed produces identical results: true
}
