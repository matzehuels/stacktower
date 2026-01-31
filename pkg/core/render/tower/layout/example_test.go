package layout_test

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/layout"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/ordering"
)

func ExampleBuild() {
	// Create a simple dependency graph
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "lib", Row: 1})
	_ = g.AddNode(dag.Node{ID: "core", Row: 2})
	_ = g.AddEdge(dag.Edge{From: "app", To: "lib"})
	_ = g.AddEdge(dag.Edge{From: "lib", To: "core"})

	// Build layout with 800x600 frame
	l := layout.Build(g, 800, 600)

	fmt.Println("Frame:", l.FrameWidth, "x", l.FrameHeight)
	fmt.Println("Block count:", len(l.Blocks))
	fmt.Println("Row count:", len(l.RowOrders))
	// Output:
	// Frame: 800 x 600
	// Block count: 3
	// Row count: 3
}

func ExampleBuild_withOptions() {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "lib", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "app", To: "lib"})

	// Build with custom options
	l := layout.Build(g, 800, 600,
		layout.WithOrderer(ordering.Barycentric{Passes: 24}),
		layout.WithMarginRatio(0.1),     // 10% margins
		layout.WithAuxiliaryRatio(0.15), // Aux rows at 15% height
	)

	fmt.Println("Margin X:", l.MarginX)
	fmt.Println("Margin Y:", l.MarginY)
	// Output:
	// Margin X: 80
	// Margin Y: 60
}

func ExampleBuild_topDownWidths() {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "auth", Row: 1})
	_ = g.AddNode(dag.Node{ID: "cache", Row: 1})
	_ = g.AddNode(dag.Node{ID: "db", Row: 2})
	_ = g.AddEdge(dag.Edge{From: "app", To: "auth"})
	_ = g.AddEdge(dag.Edge{From: "app", To: "cache"})
	_ = g.AddEdge(dag.Edge{From: "auth", To: "db"})
	_ = g.AddEdge(dag.Edge{From: "cache", To: "db"})

	// Top-down: width flows from roots downward
	// Db is shared by both auth and cache, so it receives combined width
	l := layout.Build(g, 800, 600, layout.WithTopDownWidths())

	// All nodes in this balanced graph have reasonable widths
	appBlock := l.Blocks["app"]
	dbBlock := l.Blocks["db"]

	fmt.Println("App has width:", appBlock.Width() > 0)
	fmt.Println("Db has width:", dbBlock.Width() > 0)
	// Output:
	// App has width: true
	// Db has width: true
}

func ExampleLayout_Blocks() {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "lib", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "app", To: "lib"})

	l := layout.Build(g, 400, 300)

	// Access individual block positions
	appBlock := l.Blocks["app"]
	libBlock := l.Blocks["lib"]

	// Blocks have computed coordinates
	fmt.Println("app has position:", appBlock.Left >= 0)
	fmt.Println("lib below app:", libBlock.Top > appBlock.Bottom)
	// Output:
	// app has position: true
	// lib below app: true
}

func ExampleBlock() {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "fastapi", Row: 0, Meta: dag.Metadata{"version": "0.100.0"}})

	l := layout.Build(g, 400, 300)
	block := l.Blocks["fastapi"]

	// Block contains all rendering information
	fmt.Println("NodeID:", block.NodeID)
	fmt.Println("Has dimensions:", block.Right > block.Left && block.Top > block.Bottom)
	fmt.Println("Has center:", block.CenterX() > 0 && block.CenterY() > 0)
	// Output:
	// NodeID: fastapi
	// Has dimensions: true
	// Has center: true
}

func ExampleLayout_RowOrders() {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "auth", Row: 1})
	_ = g.AddNode(dag.Node{ID: "cache", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "app", To: "auth"})
	_ = g.AddEdge(dag.Edge{From: "app", To: "cache"})

	l := layout.Build(g, 400, 300)

	// RowOrders contains the left-to-right sequence for each row
	fmt.Println("Row 0 nodes:", len(l.RowOrders[0]))
	fmt.Println("Row 1 nodes:", len(l.RowOrders[1]))
	// Output:
	// Row 0 nodes: 1
	// Row 1 nodes: 2
}
