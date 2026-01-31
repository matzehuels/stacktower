package graph_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/graph"
)

func ExampleWriteGraph() {
	// Create a simple dependency graph
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "app", Row: 0})
	_ = g.AddNode(dag.Node{ID: "lib", Row: 1, Meta: dag.Metadata{"version": "1.0.0"}})
	_ = g.AddEdge(dag.Edge{From: "app", To: "lib"})

	// Write to a buffer (or any io.Writer)
	var buf bytes.Buffer
	if err := graph.WriteGraph(g, &buf); err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("JSON output:")
	fmt.Println(buf.String())
	// Output:
	// JSON output:
	// {
	//   "nodes": [
	//     {
	//       "id": "app"
	//     },
	//     {
	//       "id": "lib",
	//       "row": 1,
	//       "meta": {
	//         "version": "1.0.0"
	//       }
	//     }
	//   ],
	//   "edges": [
	//     {
	//       "from": "app",
	//       "to": "lib"
	//     }
	//   ]
	// }
}

func ExampleReadGraph() {
	// JSON input representing a dependency graph
	jsonData := `{
		"nodes": [
			{"id": "app"},
			{"id": "lib", "row": 1}
		],
		"edges": [
			{"from": "app", "to": "lib"}
		]
	}`

	// Parse the JSON
	g, err := graph.ReadGraph(bytes.NewReader([]byte(jsonData)))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Nodes:", g.NodeCount())
	fmt.Println("Edges:", g.EdgeCount())
	fmt.Println("Children of app:", g.Children("app"))
	// Output:
	// Nodes: 2
	// Edges: 1
	// Children of app: [lib]
}

func ExampleWriteGraphFile() {
	// Build a simple graph
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "server"})
	_ = g.AddNode(dag.Node{ID: "database", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "server", To: "database"})

	// Export to a file
	tmpDir := os.TempDir()
	path := filepath.Join(tmpDir, "exported-graph.json")
	defer os.Remove(path)

	if err := graph.WriteGraphFile(g, path); err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Verify the file was created
	if _, err := os.Stat(path); err == nil {
		fmt.Println("Graph exported successfully")
	}
	// Output:
	// Graph exported successfully
}

func ExampleReadGraphFile() {
	// Create a temporary JSON file
	tmpDir := os.TempDir()
	path := filepath.Join(tmpDir, "example-graph.json")

	jsonData := []byte(`{
		"nodes": [
			{"id": "root"},
			{"id": "child-a", "row": 1},
			{"id": "child-b", "row": 1}
		],
		"edges": [
			{"from": "root", "to": "child-a"},
			{"from": "root", "to": "child-b"}
		]
	}`)

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer os.Remove(path)

	// Import the graph
	g, err := graph.ReadGraphFile(path)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Imported", g.NodeCount(), "nodes")
	fmt.Println("Root has", g.OutDegree("root"), "children")
	// Output:
	// Imported 3 nodes
	// Root has 2 children
}

func ExampleReadGraph_withMetadata() {
	// JSON with package metadata (as produced by dependency parsing)
	jsonData := `{
		"nodes": [
			{
				"id": "fastapi",
				"meta": {
					"version": "1.0.0",
					"description": "FastAPI framework",
					"repo_stars": 70000
				}
			},
			{
				"id": "pydantic",
				"row": 1,
				"meta": {
					"version": "2.0.0"
				}
			}
		],
		"edges": [
			{"from": "fastapi", "to": "pydantic"}
		]
	}`

	g, _ := graph.ReadGraph(bytes.NewReader([]byte(jsonData)))
	node, _ := g.Node("fastapi")

	fmt.Println("Package:", node.ID)
	fmt.Println("Version:", node.Meta["version"])
	fmt.Println("Stars:", node.Meta["repo_stars"])
	// Output:
	// Package: fastapi
	// Version: 1.0.0
	// Stars: 70000
}
