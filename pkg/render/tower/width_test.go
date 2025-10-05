package tower

import (
	"math"
	"testing"

	"stacktower/pkg/dag"
)

func TestComputeWidths_FlowPropagation(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A", Row: 0})
	_ = g.AddNode(dag.Node{ID: "A_sub_1", Row: 1, Kind: dag.NodeKindSubdivider, MasterID: "A"})
	_ = g.AddNode(dag.Node{ID: "A_sub_2", Row: 2, Kind: dag.NodeKindSubdivider, MasterID: "A"})
	_ = g.AddNode(dag.Node{ID: "B", Row: 0})
	_ = g.AddNode(dag.Node{ID: "C", Row: 3})
	_ = g.AddEdge(dag.Edge{From: "A", To: "A_sub_1"})
	_ = g.AddEdge(dag.Edge{From: "A_sub_1", To: "A_sub_2"})
	_ = g.AddEdge(dag.Edge{From: "A_sub_2", To: "C"})
	_ = g.AddEdge(dag.Edge{From: "B", To: "C"})

	orders := map[int][]string{
		0: {"A", "B"},
		1: {"A_sub_1"},
		2: {"A_sub_2"},
		3: {"C"},
	}

	widths := ComputeWidths(g, orders, 800.0)

	if math.Abs(widths["A"]-400.0) > 1e-9 {
		t.Errorf("A should have width 400.0 (half of frame), got %.2f", widths["A"])
	}

	if math.Abs(widths["A_sub_1"]-800.0) > 1e-9 {
		t.Errorf("A_sub_1 should fill row (800.0), got %.2f", widths["A_sub_1"])
	}

	if math.Abs(widths["A_sub_2"]-800.0) > 1e-9 {
		t.Errorf("A_sub_2 should fill row (800.0), got %.2f", widths["A_sub_2"])
	}

	if math.Abs(widths["C"]-800.0) > 1e-9 {
		t.Errorf("C should fill row (800.0), got %.2f", widths["C"])
	}
}

func TestComputeWidths_MultipleChains(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A", Row: 0})
	_ = g.AddNode(dag.Node{ID: "B", Row: 0})
	_ = g.AddNode(dag.Node{ID: "A_sub_1", Row: 1, Kind: dag.NodeKindSubdivider, MasterID: "A"})
	_ = g.AddNode(dag.Node{ID: "B_sub_1", Row: 1, Kind: dag.NodeKindSubdivider, MasterID: "B"})
	_ = g.AddEdge(dag.Edge{From: "A", To: "A_sub_1"})
	_ = g.AddEdge(dag.Edge{From: "B", To: "B_sub_1"})

	orders := map[int][]string{
		0: {"A", "B"},
		1: {"A_sub_1", "B_sub_1"},
	}

	widths := ComputeWidths(g, orders, 1000.0)

	if math.Abs(widths["A"]-widths["A_sub_1"]) > 1e-9 {
		t.Errorf("A chain: master %.2f != subdivider %.2f", widths["A"], widths["A_sub_1"])
	}

	if math.Abs(widths["B"]-widths["B_sub_1"]) > 1e-9 {
		t.Errorf("B chain: master %.2f != subdivider %.2f", widths["B"], widths["B_sub_1"])
	}

	if math.Abs(widths["A"]-500.0) > 1e-9 {
		t.Errorf("A should have width 500.0, got %.2f", widths["A"])
	}

	if math.Abs(widths["B"]-500.0) > 1e-9 {
		t.Errorf("B should have width 500.0, got %.2f", widths["B"])
	}
}
