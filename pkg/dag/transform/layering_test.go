package transform

import (
	"testing"

	"github.com/matzehuels/stacktower/pkg/dag"
)

func TestAssignLayers_SimpleChain(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a"})
	_ = g.AddNode(dag.Node{ID: "b"})
	_ = g.AddNode(dag.Node{ID: "c"})
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})
	_ = g.AddEdge(dag.Edge{From: "b", To: "c"})

	AssignLayers(g)

	checkRow(t, g, "a", 0)
	checkRow(t, g, "b", 1)
	checkRow(t, g, "c", 2)
}

func TestAssignLayers_Diamond(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a"})
	_ = g.AddNode(dag.Node{ID: "b"})
	_ = g.AddNode(dag.Node{ID: "c"})
	_ = g.AddNode(dag.Node{ID: "d"})

	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})
	_ = g.AddEdge(dag.Edge{From: "a", To: "c"})
	_ = g.AddEdge(dag.Edge{From: "b", To: "d"})
	_ = g.AddEdge(dag.Edge{From: "c", To: "d"})

	AssignLayers(g)

	checkRow(t, g, "a", 0)
	checkRow(t, g, "b", 1)
	checkRow(t, g, "c", 1)
	checkRow(t, g, "d", 2)
}

func checkRow(t *testing.T, g *dag.DAG, id string, expected int) {
	t.Helper()
	n, ok := g.Node(id)
	if !ok {
		t.Fatalf("node %s not found", id)
	}
	if n.Row != expected {
		t.Errorf("node %s: expected row %d, got %d", id, expected, n.Row)
	}
}

func TestAssignLayers_EmptyGraph(t *testing.T) {
	g := dag.New(nil)
	AssignLayers(g)
	if g.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", g.NodeCount())
	}
}

func TestAssignLayers_SingleNode(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a"})
	AssignLayers(g)
	checkRow(t, g, "a", 0)
}

func TestAssignLayers_DisconnectedNodes(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a"})
	_ = g.AddNode(dag.Node{ID: "b"})
	_ = g.AddNode(dag.Node{ID: "c"})

	AssignLayers(g)

	checkRow(t, g, "a", 0)
}

func TestAssignLayers_MultipleRoots(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a"})
	_ = g.AddNode(dag.Node{ID: "b"})
	_ = g.AddNode(dag.Node{ID: "c"})
	_ = g.AddNode(dag.Node{ID: "d"})

	_ = g.AddEdge(dag.Edge{From: "a", To: "c"})
	_ = g.AddEdge(dag.Edge{From: "b", To: "d"})

	AssignLayers(g)

	checkRow(t, g, "a", 0)
	checkRow(t, g, "b", 0)
	checkRow(t, g, "c", 1)
	checkRow(t, g, "d", 1)
}

func TestAssignLayers_LongestPath(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a"})
	_ = g.AddNode(dag.Node{ID: "b"})
	_ = g.AddNode(dag.Node{ID: "c"})
	_ = g.AddNode(dag.Node{ID: "d"})

	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})
	_ = g.AddEdge(dag.Edge{From: "a", To: "d"})
	_ = g.AddEdge(dag.Edge{From: "b", To: "c"})
	_ = g.AddEdge(dag.Edge{From: "c", To: "d"})

	AssignLayers(g)

	checkRow(t, g, "a", 0)
	checkRow(t, g, "b", 1)
	checkRow(t, g, "c", 2)
	checkRow(t, g, "d", 3)
}

func TestAssignLayers_ComplexDAG(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a"})
	_ = g.AddNode(dag.Node{ID: "b"})
	_ = g.AddNode(dag.Node{ID: "c"})
	_ = g.AddNode(dag.Node{ID: "d"})
	_ = g.AddNode(dag.Node{ID: "e"})
	_ = g.AddNode(dag.Node{ID: "f"})

	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})
	_ = g.AddEdge(dag.Edge{From: "a", To: "c"})
	_ = g.AddEdge(dag.Edge{From: "b", To: "d"})
	_ = g.AddEdge(dag.Edge{From: "c", To: "d"})
	_ = g.AddEdge(dag.Edge{From: "c", To: "e"})
	_ = g.AddEdge(dag.Edge{From: "d", To: "f"})
	_ = g.AddEdge(dag.Edge{From: "e", To: "f"})

	AssignLayers(g)

	checkRow(t, g, "a", 0)
	checkRow(t, g, "b", 1)
	checkRow(t, g, "c", 1)
	checkRow(t, g, "d", 2)
	checkRow(t, g, "e", 2)
	checkRow(t, g, "f", 3)
}

func TestAssignLayers_PreservesTopologicalOrder(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a"})
	_ = g.AddNode(dag.Node{ID: "b"})
	_ = g.AddNode(dag.Node{ID: "c"})
	_ = g.AddEdge(dag.Edge{From: "a", To: "b"})
	_ = g.AddEdge(dag.Edge{From: "b", To: "c"})

	AssignLayers(g)

	nodeA, _ := g.Node("a")
	nodeB, _ := g.Node("b")
	nodeC, _ := g.Node("c")

	if nodeA.Row >= nodeB.Row {
		t.Errorf("parent a (row %d) should be before child b (row %d)", nodeA.Row, nodeB.Row)
	}
	if nodeB.Row >= nodeC.Row {
		t.Errorf("parent b (row %d) should be before child c (row %d)", nodeB.Row, nodeC.Row)
	}
}

func TestAssignLayers_FanInFanOut(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a"})
	_ = g.AddNode(dag.Node{ID: "b"})
	_ = g.AddNode(dag.Node{ID: "c"})
	_ = g.AddNode(dag.Node{ID: "d"})
	_ = g.AddNode(dag.Node{ID: "e"})
	_ = g.AddNode(dag.Node{ID: "f"})

	_ = g.AddEdge(dag.Edge{From: "a", To: "c"})
	_ = g.AddEdge(dag.Edge{From: "b", To: "c"})
	_ = g.AddEdge(dag.Edge{From: "c", To: "d"})
	_ = g.AddEdge(dag.Edge{From: "c", To: "e"})
	_ = g.AddEdge(dag.Edge{From: "d", To: "f"})
	_ = g.AddEdge(dag.Edge{From: "e", To: "f"})

	AssignLayers(g)

	checkRow(t, g, "a", 0)
	checkRow(t, g, "b", 0)
	checkRow(t, g, "c", 1)
	checkRow(t, g, "d", 2)
	checkRow(t, g, "e", 2)
	checkRow(t, g, "f", 3)
}
