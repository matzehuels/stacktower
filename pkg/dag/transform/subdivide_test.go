package transform

import (
	"testing"

	"stacktower/pkg/dag"
)

func TestSubdivide_NoEdges(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 0})

	Subdivide(g)

	if g.EdgeCount() != 0 {
		t.Errorf("expected 0 edges (no connections), got %d", g.EdgeCount())
	}
}

func TestSubdivide_AllConsecutiveRows(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 1})
	_ = g.AddNode(dag.Node{ID: "c", Row: 2})
	_ = g.AddNode(dag.Node{ID: "d", Row: 3})
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})
	_ = g.AddEdge(dag.Edge{From: "b", To: "c"})
	_ = g.AddEdge(dag.Edge{From: "c", To: "d"})

	Subdivide(g)

	if g.NodeCount() != 4 {
		t.Errorf("expected 4 nodes (no subdivision needed), got %d", g.NodeCount())
	}
	if g.EdgeCount() != 3 {
		t.Errorf("expected 3 edges, got %d", g.EdgeCount())
	}
}

func TestSubdivide_VeryLongEdge(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 10})
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})

	Subdivide(g)

	expectedSubdividers := 9
	subdividerCount := 0
	for _, n := range g.Nodes() {
		if n.IsSubdivider() {
			subdividerCount++
		}
	}

	if subdividerCount != expectedSubdividers {
		t.Errorf("expected %d subdividers, got %d", expectedSubdividers, subdividerCount)
	}

	expectedEdges := 10
	if g.EdgeCount() != expectedEdges {
		t.Errorf("expected %d edges, got %d", expectedEdges, g.EdgeCount())
	}
}

func TestSubdivide_MixedEdgeLengths(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 1})
	_ = g.AddNode(dag.Node{ID: "c", Row: 5})
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})
	_ = g.AddEdge(dag.Edge{From: "b", To: "c"})

	edgesBefore := g.EdgeCount()
	Subdivide(g)

	if g.EdgeCount() <= edgesBefore {
		t.Errorf("expected more edges after subdivision, had %d, got %d", edgesBefore, g.EdgeCount())
	}
}

func TestSubdivide_SinkExtension_SingleSink(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 1})
	_ = g.AddNode(dag.Node{ID: "c", Row: 5})
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})
	_ = g.AddEdge(dag.Edge{From: "a", To: "c"})

	Subdivide(g)

	maxRow := 0
	for _, n := range g.Nodes() {
		if n.Row > maxRow {
			maxRow = n.Row
		}
	}

	sinkNode, ok := g.Node("b")
	if !ok {
		t.Fatal("sink node b not found")
	}

	descendants := findDescendants(g, "b")
	hasDescendantAtMaxRow := false
	for _, desc := range descendants {
		if desc.Row == maxRow {
			hasDescendantAtMaxRow = true
			break
		}
	}

	if !hasDescendantAtMaxRow {
		t.Errorf("sink node b (row %d) should extend to max row %d", sinkNode.Row, maxRow)
	}
}

func TestSubdivide_SinkExtension_MultipleSinksAtDifferentLevels(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 2})
	_ = g.AddNode(dag.Node{ID: "c", Row: 5})
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})
	_ = g.AddEdge(dag.Edge{From: "a", To: "c"})

	Subdivide(g)

	maxRow := 0
	for _, n := range g.Nodes() {
		if n.Row > maxRow {
			maxRow = n.Row
		}
	}

	sinkCount := 0
	for _, n := range g.Nodes() {
		if g.OutDegree(n.ID) == 0 {
			sinkCount++
			if n.Row != maxRow {
				t.Errorf("sink %s at row %d should be at max row %d", n.ID, n.Row, maxRow)
			}
		}
	}

	if sinkCount < 2 {
		t.Errorf("expected at least 2 sinks, got %d", sinkCount)
	}
}

func TestSubdivide_PreservesOriginalNodeProperties(t *testing.T) {
	meta := dag.Metadata{"version": "1.0"}
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0, Meta: meta})
	_ = g.AddNode(dag.Node{ID: "b", Row: 3})
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})

	Subdivide(g)

	nodeA, ok := g.Node("a")
	if !ok {
		t.Fatal("original node a not found")
	}
	if nodeA.Meta["version"] != "1.0" {
		t.Error("original node metadata should be preserved")
	}
}

func TestSubdivide_SubdividerNaming(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "parent", Row: 0})
	_ = g.AddNode(dag.Node{ID: "child", Row: 3})
	_ = g.AddEdge(dag.Edge{From: "parent", To: "child"})

	Subdivide(g)

	subdividers := make([]*dag.Node, 0)
	for _, n := range g.Nodes() {
		if n.IsSubdivider() {
			subdividers = append(subdividers, n)
		}
	}

	if len(subdividers) != 2 {
		t.Fatalf("expected 2 subdividers, got %d", len(subdividers))
	}

	for _, sub := range subdividers {
		if sub.MasterID != "parent" {
			t.Errorf("subdivider %s should have MasterID 'parent', got '%s'", sub.ID, sub.MasterID)
		}
	}
}

func TestSubdivide_IDGenerator_HandlesCollisions(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "a_sub_1", Row: 1})
	_ = g.AddNode(dag.Node{ID: "b", Row: 3})
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})

	Subdivide(g)

	ids := make(map[string]bool)
	for _, n := range g.Nodes() {
		if ids[n.ID] {
			t.Errorf("duplicate ID found: %s", n.ID)
		}
		ids[n.ID] = true
	}
}

func TestSubdivide_ComplexGraph(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 0})
	_ = g.AddNode(dag.Node{ID: "c", Row: 2})
	_ = g.AddNode(dag.Node{ID: "d", Row: 5})
	_ = g.AddNode(dag.Node{ID: "e", Row: 3})

	_ = g.AddEdge(dag.Edge{From: "a", To: "c"})
	_ = g.AddEdge(dag.Edge{From: "b", To: "d"})
	_ = g.AddEdge(dag.Edge{From: "c", To: "e"})

	nodesBefore := g.NodeCount()
	Subdivide(g)
	nodesAfter := g.NodeCount()

	if nodesAfter <= nodesBefore {
		t.Errorf("expected more nodes after subdivision, had %d, got %d", nodesBefore, nodesAfter)
	}

	maxRow := 0
	for _, n := range g.Nodes() {
		if n.Row > maxRow {
			maxRow = n.Row
		}
	}

	allSinksAtMaxRow := true
	for _, n := range g.Nodes() {
		if g.OutDegree(n.ID) == 0 && n.Row != maxRow {
			allSinksAtMaxRow = false
			break
		}
	}

	if !allSinksAtMaxRow {
		t.Error("all sinks should be extended to max row")
	}
}

func TestSubdivide_MaintainsConnectivity(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 5})
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})

	Subdivide(g)

	path := findPath(g, "a", "b")
	if len(path) == 0 {
		t.Error("should maintain path from a to b after subdivision")
	}
}

func findDescendants(g *dag.DAG, nodeID string) []*dag.Node {
	descendants := make([]*dag.Node, 0)
	visited := make(map[string]bool)
	queue := []string{nodeID}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if visited[curr] {
			continue
		}
		visited[curr] = true

		children := g.Children(curr)
		for _, child := range children {
			if n, ok := g.Node(child); ok {
				descendants = append(descendants, n)
				queue = append(queue, child)
			}
		}
	}

	return descendants
}

func findPath(g *dag.DAG, from, to string) []string {
	if from == to {
		return []string{from}
	}

	visited := make(map[string]bool)
	parent := make(map[string]string)
	queue := []string{from}
	visited[from] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr == to {
			path := make([]string, 0)
			for node := to; node != ""; node = parent[node] {
				path = append([]string{node}, path...)
				if node == from {
					break
				}
			}
			return path
		}

		for _, child := range g.Children(curr) {
			if !visited[child] {
				visited[child] = true
				parent[child] = curr
				queue = append(queue, child)
			}
		}
	}

	return nil
}
