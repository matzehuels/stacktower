package perm

import (
	"strings"
	"testing"
)

func TestPQTreeToDOT(t *testing.T) {
	tree := NewPQTree(3)

	dot := tree.ToDOT([]string{"a", "b", "c"})

	// Check basic DOT structure
	if !strings.HasPrefix(dot, "digraph PQTree {") {
		t.Error("ToDOT() should start with 'digraph PQTree {'")
	}
	if !strings.HasSuffix(strings.TrimSpace(dot), "}") {
		t.Error("ToDOT() should end with '}'")
	}

	// Check for expected attributes
	expected := []string{
		"rankdir=TB",
		"bgcolor=\"transparent\"",
		"fontname=",
		"arrowhead=none",
	}
	for _, exp := range expected {
		if !strings.Contains(dot, exp) {
			t.Errorf("ToDOT() missing %q", exp)
		}
	}
}

func TestPQTreeToDOTWithLabels(t *testing.T) {
	tree := NewPQTree(3)
	labels := []string{"alpha", "beta", "gamma"}

	dot := tree.ToDOT(labels)

	// Should contain the labels
	for _, label := range labels {
		if !strings.Contains(dot, label) {
			t.Errorf("ToDOT() should contain label %q", label)
		}
	}
}

func TestPQTreeToDOTEmptyTree(t *testing.T) {
	tree := &PQTree{}

	dot := tree.ToDOT(nil)

	// Should still produce valid DOT without crashing
	if !strings.Contains(dot, "digraph PQTree {") {
		t.Error("ToDOT() should produce valid DOT for empty tree")
	}
}

func TestPQTreeToDOTSingleNode(t *testing.T) {
	tree := NewPQTree(1)

	dot := tree.ToDOT([]string{"single"})

	if !strings.Contains(dot, "single") {
		t.Error("ToDOT() should contain single node label")
	}
	if !strings.Contains(dot, "shape=box") {
		t.Error("ToDOT() leaf node should have box shape")
	}
}

func TestPQTreeToDOTAfterReduce(t *testing.T) {
	tree := NewPQTree(5)

	// Reduce to create P and Q nodes
	tree.Reduce([]int{1, 2, 3})

	labels := []string{"a", "b", "c", "d", "e"}
	dot := tree.ToDOT(labels)

	// Should contain internal nodes
	// P nodes have ellipse shape, Q nodes have box shape
	if !strings.Contains(dot, "shape=") {
		t.Error("ToDOT() should contain shape attributes for nodes")
	}
}

func TestWriteDOTNodeLeaf(t *testing.T) {
	tree := NewPQTree(1)

	dot := tree.ToDOT([]string{"leaf-node"})

	// Leaf nodes should have rounded box style
	if !strings.Contains(dot, "style=\"filled,rounded\"") {
		t.Error("ToDOT() leaf nodes should have filled,rounded style")
	}
}

func TestWriteDOTNodePNode(t *testing.T) {
	tree := NewPQTree(3)
	// Reduce to create internal structure
	tree.Reduce([]int{0, 1})

	dot := tree.ToDOT([]string{"a", "b", "c"})

	// P nodes should be labeled "P"
	if !strings.Contains(dot, `label="P"`) {
		t.Error("ToDOT() should contain P node")
	}
}

func TestWriteDOTNodeQNode(t *testing.T) {
	tree := NewPQTree(4)
	// Reduce twice to potentially create Q node
	tree.Reduce([]int{0, 1})
	tree.Reduce([]int{1, 2})

	dot := tree.ToDOT([]string{"a", "b", "c", "d"})

	// Q nodes should be labeled "Q"
	if !strings.Contains(dot, `label="Q"`) {
		t.Error("ToDOT() should contain Q node")
	}
}

func TestPQTreeToDOTNodeIDs(t *testing.T) {
	tree := NewPQTree(3)

	dot := tree.ToDOT([]string{"a", "b", "c"})

	// Node IDs should be in format n0, n1, etc.
	if !strings.Contains(dot, "n0") {
		t.Error("ToDOT() should contain node ID n0")
	}
}

func TestPQTreeToDOTEdges(t *testing.T) {
	tree := NewPQTree(3)
	tree.Reduce([]int{0, 1})

	dot := tree.ToDOT([]string{"a", "b", "c"})

	// Should contain edges with ->
	if !strings.Contains(dot, "->") {
		t.Error("ToDOT() should contain edges")
	}
}
