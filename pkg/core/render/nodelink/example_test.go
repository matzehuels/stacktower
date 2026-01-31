package nodelink_test

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/nodelink"
)

func ExampleToDOT() {
	// Create a simple dependency graph
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "database", Row: 1})
	_ = g.AddNode(dag.Node{ID: "auth", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "app", To: "database"})
	_ = g.AddEdge(dag.Edge{From: "app", To: "auth"})

	// Convert to DOT format
	_ = nodelink.ToDOT(g, nodelink.Options{})

	// The DOT output can be rendered with Graphviz
	fmt.Println("Generated DOT format for visualization")
	// Output:
	// Generated DOT format for visualization
}

func ExampleToDOT_detailed() {
	// Create a graph with metadata
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{
		ID:   "fastapi",
		Row:  0,
		Meta: dag.Metadata{"version": "0.100.0"},
	})
	_ = g.AddNode(dag.Node{
		ID:   "pydantic",
		Row:  1,
		Meta: dag.Metadata{"version": "2.0.0"},
	})
	_ = g.AddEdge(dag.Edge{From: "fastapi", To: "pydantic"})

	// Use detailed mode to include metadata in labels
	_ = nodelink.ToDOT(g, nodelink.Options{Detailed: true})

	// The detailed DOT includes row numbers and metadata
	fmt.Println("Generated detailed DOT with metadata")
	// Output:
	// Generated detailed DOT with metadata
}

func ExampleRenderSVG() {
	// Create a simple graph
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "web", Row: 0})
	_ = g.AddNode(dag.Node{ID: "api", Row: 1})
	_ = g.AddNode(dag.Node{ID: "db", Row: 2})
	_ = g.AddEdge(dag.Edge{From: "web", To: "api"})
	_ = g.AddEdge(dag.Edge{From: "api", To: "db"})

	// Convert to DOT
	dot := nodelink.ToDOT(g, nodelink.Options{})

	// Render to SVG (requires Graphviz)
	svg, err := nodelink.RenderSVG(dot)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Generated SVG (%d bytes)\n", len(svg))
	// Output varies based on Graphviz installation
}

func ExampleRenderPDF() {
	// Create a dependency graph
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "frontend", Row: 0})
	_ = g.AddNode(dag.Node{ID: "backend", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "frontend", To: "backend"})

	// Convert to DOT
	dot := nodelink.ToDOT(g, nodelink.Options{})

	// Render to PDF (requires Graphviz and librsvg)
	pdf, err := nodelink.RenderPDF(dot)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Generated PDF (%d bytes)\n", len(pdf))
	// Output varies based on tool installation
}

func ExampleRenderPNG() {
	// Create a graph
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "service", Row: 0})
	_ = g.AddNode(dag.Node{ID: "cache", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "service", To: "cache"})

	// Convert to DOT
	dot := nodelink.ToDOT(g, nodelink.Options{})

	// Render to high-resolution PNG (requires Graphviz and librsvg)
	png, err := nodelink.RenderPNG(dot, 2.0) // 2x scale for retina displays
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Generated PNG (%d bytes)\n", len(png))
	// Output varies based on tool installation
}
