package sink_test

import (
	"fmt"
	"strings"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/layout"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/sink"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/styles"
)

func ExampleRenderSVG() {
	// Build a simple dependency graph
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "lib", Row: 1})
	_ = g.AddNode(dag.Node{ID: "core", Row: 2})
	_ = g.AddEdge(dag.Edge{From: "app", To: "lib"})
	_ = g.AddEdge(dag.Edge{From: "lib", To: "core"})

	// Compute layout
	l := layout.Build(g, 400, 300)

	// Render to SVG
	svg := sink.RenderSVG(l, sink.WithGraph(g))

	fmt.Println("SVG starts with:", string(svg[:4]))
	fmt.Println("Contains viewBox:", strings.Contains(string(svg), "viewBox"))
	// Output:
	// SVG starts with: <svg
	// Contains viewBox: true
}

func ExampleRenderSVG_withStyle() {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "lib", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "app", To: "lib"})

	l := layout.Build(g, 400, 300)

	// Use simple style (clean, minimal appearance)
	svg := sink.RenderSVG(l,
		sink.WithGraph(g),
		sink.WithStyle(styles.Simple{}),
	)

	fmt.Println("Generated SVG length:", len(svg) > 0)
	// Output:
	// Generated SVG length: true
}

func ExampleRenderSVG_withEdges() {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "auth", Row: 1})
	_ = g.AddNode(dag.Node{ID: "cache", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "app", To: "auth"})
	_ = g.AddEdge(dag.Edge{From: "app", To: "cache"})

	l := layout.Build(g, 400, 300)

	// Enable edge rendering to show dependency lines
	svg := sink.RenderSVG(l,
		sink.WithGraph(g),
		sink.WithEdges(),
	)

	// Edges are rendered as dashed lines
	fmt.Println("Has content:", len(svg) > 100)
	// Output:
	// Has content: true
}

func ExampleRenderSVG_withPopups() {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{
		ID:  "fastapi",
		Row: 0,
		Meta: dag.Metadata{
			"version":          "0.100.0",
			"repo_description": "FastAPI framework",
			"repo_stars":       70000,
		},
	})
	_ = g.AddNode(dag.Node{ID: "starlette", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "fastapi", To: "starlette"})

	l := layout.Build(g, 400, 300)

	// Enable hover popups showing package metadata
	svg := sink.RenderSVG(l,
		sink.WithGraph(g),
		sink.WithPopups(),
	)

	// Popups include metadata from nodes
	fmt.Println("Has popup content:", strings.Contains(string(svg), "popup"))
	// Output:
	// Has popup content: true
}
