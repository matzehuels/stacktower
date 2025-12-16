package transform

import (
	"testing"

	"github.com/matzehuels/stacktower/pkg/dag"
)

func TestBreakCycles_NoCycles(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a"})
	g.AddNode(dag.Node{ID: "b"})
	g.AddNode(dag.Node{ID: "c"})
	g.AddEdge(dag.Edge{From: "a", To: "b"})
	g.AddEdge(dag.Edge{From: "b", To: "c"})

	removed := BreakCycles(g)

	if removed != 0 {
		t.Errorf("BreakCycles() removed %d edges, want 0", removed)
	}
	if g.EdgeCount() != 2 {
		t.Errorf("EdgeCount() = %d, want 2", g.EdgeCount())
	}
}

func TestBreakCycles_SimpleCycle(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a"})
	g.AddNode(dag.Node{ID: "b"})
	g.AddEdge(dag.Edge{From: "a", To: "b"})
	g.AddEdge(dag.Edge{From: "b", To: "a"})

	removed := BreakCycles(g)

	if removed != 1 {
		t.Errorf("BreakCycles() removed %d edges, want 1", removed)
	}
	if g.EdgeCount() != 1 {
		t.Errorf("EdgeCount() = %d, want 1", g.EdgeCount())
	}
}

func TestBreakCycles_TriangleCycle(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a"})
	g.AddNode(dag.Node{ID: "b"})
	g.AddNode(dag.Node{ID: "c"})
	g.AddEdge(dag.Edge{From: "a", To: "b"})
	g.AddEdge(dag.Edge{From: "b", To: "c"})
	g.AddEdge(dag.Edge{From: "c", To: "a"})

	removed := BreakCycles(g)

	if removed != 1 {
		t.Errorf("BreakCycles() removed %d edges, want 1", removed)
	}
	if g.EdgeCount() != 2 {
		t.Errorf("EdgeCount() = %d, want 2", g.EdgeCount())
	}
}

func TestBreakCycles_MultipleCycles(t *testing.T) {
	// Two separate cycles: a↔b and c↔d
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a"})
	g.AddNode(dag.Node{ID: "b"})
	g.AddNode(dag.Node{ID: "c"})
	g.AddNode(dag.Node{ID: "d"})
	g.AddEdge(dag.Edge{From: "a", To: "b"})
	g.AddEdge(dag.Edge{From: "b", To: "a"})
	g.AddEdge(dag.Edge{From: "c", To: "d"})
	g.AddEdge(dag.Edge{From: "d", To: "c"})

	removed := BreakCycles(g)

	if removed != 2 {
		t.Errorf("BreakCycles() removed %d edges, want 2", removed)
	}
	if g.EdgeCount() != 2 {
		t.Errorf("EdgeCount() = %d, want 2", g.EdgeCount())
	}
}

func TestBreakCycles_SelfLoop(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a"})
	g.AddEdge(dag.Edge{From: "a", To: "a"})

	removed := BreakCycles(g)

	if removed != 1 {
		t.Errorf("BreakCycles() removed %d edges, want 1", removed)
	}
	if g.EdgeCount() != 0 {
		t.Errorf("EdgeCount() = %d, want 0", g.EdgeCount())
	}
}

func TestBreakCycles_DiamondNoCycle(t *testing.T) {
	//   a
	//  / \
	// b   c
	//  \ /
	//   d
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a"})
	g.AddNode(dag.Node{ID: "b"})
	g.AddNode(dag.Node{ID: "c"})
	g.AddNode(dag.Node{ID: "d"})
	g.AddEdge(dag.Edge{From: "a", To: "b"})
	g.AddEdge(dag.Edge{From: "a", To: "c"})
	g.AddEdge(dag.Edge{From: "b", To: "d"})
	g.AddEdge(dag.Edge{From: "c", To: "d"})

	removed := BreakCycles(g)

	if removed != 0 {
		t.Errorf("BreakCycles() removed %d edges, want 0", removed)
	}
	if g.EdgeCount() != 4 {
		t.Errorf("EdgeCount() = %d, want 4", g.EdgeCount())
	}
}

func TestBreakCycles_ResultIsAcyclic(t *testing.T) {
	// Complex graph with cycle
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a"})
	g.AddNode(dag.Node{ID: "b"})
	g.AddNode(dag.Node{ID: "c"})
	g.AddNode(dag.Node{ID: "d"})
	g.AddEdge(dag.Edge{From: "a", To: "b"})
	g.AddEdge(dag.Edge{From: "b", To: "c"})
	g.AddEdge(dag.Edge{From: "c", To: "d"})
	g.AddEdge(dag.Edge{From: "d", To: "b"}) // back-edge creating cycle

	BreakCycles(g)

	// Run again - should find no more cycles
	removed := BreakCycles(g)
	if removed != 0 {
		t.Errorf("Graph still has cycles after BreakCycles()")
	}
}

func TestBreakCycles_EmptyGraph(t *testing.T) {
	g := dag.New(nil)

	removed := BreakCycles(g)

	if removed != 0 {
		t.Errorf("BreakCycles() removed %d edges, want 0", removed)
	}
}

func TestBreakCycles_SingleNode(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "a"})

	removed := BreakCycles(g)

	if removed != 0 {
		t.Errorf("BreakCycles() removed %d edges, want 0", removed)
	}
}
