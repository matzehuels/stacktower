package transform

import (
	"math"
	"testing"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/layout"
)

func TestRandomize_Deterministic(t *testing.T) {
	layout := buildTestLayout()

	result1 := Randomize(layout, buildTestDAG(), 12345, nil)
	result2 := Randomize(layout, buildTestDAG(), 12345, nil)

	for id := range layout.Blocks {
		b1, b2 := result1.Blocks[id], result2.Blocks[id]
		if b1.Left != b2.Left || b1.Right != b2.Right {
			t.Errorf("block %s not deterministic: (%.2f, %.2f) vs (%.2f, %.2f)",
				id, b1.Left, b1.Right, b2.Left, b2.Right)
		}
	}
}

func TestRandomize_WidthShrinks(t *testing.T) {
	layout := buildTestLayout()
	g := buildTestDAG()
	result := Randomize(layout, g, 42, nil)

	slotWidth := layout.Blocks["A"].Width()

	blockA := result.Blocks["A"]
	if blockA.Width() != slotWidth {
		t.Errorf("root block A should not shrink, got width %.2f, want %.2f",
			blockA.Width(), slotWidth)
	}

	blockB := result.Blocks["B"]
	blockC := result.Blocks["C"]

	if blockB.Width() >= slotWidth {
		t.Errorf("block B should shrink, got width %.2f == slot %.2f", blockB.Width(), slotWidth)
	}
	if blockC.Width() >= slotWidth {
		t.Errorf("block C should shrink, got width %.2f == slot %.2f", blockC.Width(), slotWidth)
	}

	for id, original := range layout.Blocks {
		if id == "A" {
			continue
		}
		randomized := result.Blocks[id]
		if randomized.Width() > original.Width()+0.01 {
			t.Errorf("block %s width increased: %.2f > %.2f", id, randomized.Width(), original.Width())
		}
		shrinkRatio := (original.Width() - randomized.Width()) / original.Width()
		if shrinkRatio > 0.60 {
			t.Errorf("block %s shrink ratio = %.2f%%, want <= 60%%", id, shrinkRatio*100)
		}
	}
}

func TestRandomize_StaysInBounds(t *testing.T) {
	layout := buildTestLayout()
	result := Randomize(layout, buildTestDAG(), 99, nil)

	for id, original := range layout.Blocks {
		b := result.Blocks[id]
		if b.Left < original.Left || b.Right > original.Right {
			t.Errorf("block %s bounds [%.2f, %.2f] outside slot [%.2f, %.2f]",
				id, b.Left, b.Right, original.Left, original.Right)
		}
	}
}

func TestRandomize_PreservesAllBlocks(t *testing.T) {
	layout := buildTestLayout()
	result := Randomize(layout, buildTestDAG(), 123, nil)

	if got, want := len(result.Blocks), len(layout.Blocks); got != want {
		t.Fatalf("block count = %d, want %d", got, want)
	}

	for id := range layout.Blocks {
		if _, ok := result.Blocks[id]; !ok {
			t.Errorf("missing block %s", id)
		}
	}
}

func TestRandomize_ZeroVariation(t *testing.T) {
	layout := buildTestLayout()
	result := Randomize(layout, buildTestDAG(), 42, &Options{
		WidthShrink: 0,
	})

	for id, original := range layout.Blocks {
		randomized := result.Blocks[id]
		if randomized.Left != original.Left || randomized.Right != original.Right {
			t.Errorf("block %s changed with zero variation: (%.2f, %.2f) vs (%.2f, %.2f)",
				id, randomized.Left, randomized.Right, original.Left, original.Right)
		}
	}
}

func TestRandomize_CustomParameters(t *testing.T) {
	layout := buildTestLayout()
	g := buildTestDAG()
	result := Randomize(layout, g, 77, &Options{
		WidthShrink:   0.25,
		MinBlockWidth: 20.0,
	})

	slotWidth := layout.Blocks["A"].Width()

	blockA := result.Blocks["A"]
	if blockA.Width() != slotWidth {
		t.Errorf("root block A should not shrink, got width %.2f, want %.2f",
			blockA.Width(), slotWidth)
	}

	for id, original := range layout.Blocks {
		if id == "A" {
			continue
		}
		randomized := result.Blocks[id]
		shrinkRatio := (original.Width() - randomized.Width()) / original.Width()
		if shrinkRatio > 0.25 {
			t.Errorf("block %s shrink ratio = %.2f%%, want <= 25%%", id, shrinkRatio*100)
		}
	}
}

func TestRandomize_PreservesVertical(t *testing.T) {
	layout := buildTestLayout()
	result := Randomize(layout, buildTestDAG(), 11, nil)

	for id, original := range layout.Blocks {
		randomized := result.Blocks[id]
		if randomized.Bottom != original.Bottom || randomized.Top != original.Top {
			t.Errorf("block %s vertical = (%.2f, %.2f), want (%.2f, %.2f)",
				id, randomized.Bottom, randomized.Top, original.Bottom, original.Top)
		}
	}
}

func TestRandomize_PreservesLayoutMetadata(t *testing.T) {
	layout := buildTestLayout()
	result := Randomize(layout, buildTestDAG(), 55, nil)

	if got, want := result.FrameWidth, layout.FrameWidth; got != want {
		t.Errorf("FrameWidth = %.2f, want %.2f", got, want)
	}
	if got, want := result.FrameHeight, layout.FrameHeight; got != want {
		t.Errorf("FrameHeight = %.2f, want %.2f", got, want)
	}
	if got, want := result.MarginX, layout.MarginX; got != want {
		t.Errorf("MarginX = %.2f, want %.2f", got, want)
	}
	if got, want := result.MarginY, layout.MarginY; got != want {
		t.Errorf("MarginY = %.2f, want %.2f", got, want)
	}
}

func TestRandomize_NoHorizontalOverlap(t *testing.T) {
	layout := buildMultiColumnLayout()
	result := Randomize(layout, buildMultiColumnDAG(), 123, nil)

	for row, ids := range result.RowOrders {
		for i := 0; i < len(ids)-1; i++ {
			curr, next := result.Blocks[ids[i]], result.Blocks[ids[i+1]]
			if curr.Right > next.Left {
				t.Errorf("row %d: block %s (right=%.2f) overlaps %s (left=%.2f)",
					row, ids[i], curr.Right, ids[i+1], next.Left)
			}
		}
	}
}

func TestRandomize_ClampsToValidRange(t *testing.T) {
	layout := buildTestLayout()
	result := Randomize(layout, buildTestDAG(), 42, &Options{
		WidthShrink:   5.0,
		MinBlockWidth: 20.0,
	})

	for id := range layout.Blocks {
		if _, ok := result.Blocks[id]; !ok {
			t.Errorf("missing block %s after out-of-range parameters", id)
		}
	}
}

func TestRandomize_MinimumOverlap(t *testing.T) {
	layout := buildTestLayout()
	g := buildTestDAG()
	minOverlap := 40.0

	result := Randomize(layout, g, 999, &Options{
		WidthShrink:   0.8,
		MinBlockWidth: 20.0,
		MinGap:        5.0,
		MinOverlap:    minOverlap,
	})

	for _, edge := range g.Edges() {
		parent := result.Blocks[edge.From]
		child := result.Blocks[edge.To]

		overlap := math.Max(0, math.Min(parent.Right, child.Right)-math.Max(parent.Left, child.Left))
		if overlap < minOverlap-0.01 {
			t.Errorf("edge %s->%s: overlap %.2f < min %.2f",
				edge.From, edge.To, overlap, minOverlap)
		}
	}
}

func TestRandomize_MinimumOverlapComplex(t *testing.T) {
	layout := buildMultiColumnLayout()
	g := buildMultiColumnDAG()
	minOverlap := 50.0

	result := Randomize(layout, g, 777, &Options{
		WidthShrink:   0.9,
		MinBlockWidth: 20.0,
		MinGap:        5.0,
		MinOverlap:    minOverlap,
	})

	for _, edge := range g.Edges() {
		parent := result.Blocks[edge.From]
		child := result.Blocks[edge.To]

		overlap := math.Max(0, math.Min(parent.Right, child.Right)-math.Max(parent.Left, child.Left))
		if overlap < minOverlap-0.01 {
			t.Errorf("edge %s->%s: overlap %.2f < min %.2f (parent=[%.2f,%.2f], child=[%.2f,%.2f])",
				edge.From, edge.To, overlap, minOverlap,
				parent.Left, parent.Right, child.Left, child.Right)
		}
	}
}

func buildMultiColumnLayout() layout.Layout {
	return layout.Layout{
		FrameWidth:  800,
		FrameHeight: 400,
		MarginX:     20,
		MarginY:     20,
		Blocks: map[string]layout.Block{
			"A": {NodeID: "A", Left: 20, Right: 420, Bottom: 20, Top: 220},
			"B": {NodeID: "B", Left: 420, Right: 780, Bottom: 20, Top: 220},
			"C": {NodeID: "C", Left: 20, Right: 420, Bottom: 220, Top: 420},
			"D": {NodeID: "D", Left: 420, Right: 780, Bottom: 220, Top: 420},
		},
		RowOrders: map[int][]string{
			0: {"A", "B"},
			1: {"C", "D"},
		},
	}
}

func buildTestLayout() layout.Layout {
	return layout.Layout{
		FrameWidth:  800,
		FrameHeight: 600,
		MarginX:     20,
		MarginY:     20,
		Blocks: map[string]layout.Block{
			"A": {NodeID: "A", Left: 20, Right: 780, Bottom: 20, Top: 220},
			"B": {NodeID: "B", Left: 20, Right: 780, Bottom: 220, Top: 420},
			"C": {NodeID: "C", Left: 20, Right: 780, Bottom: 420, Top: 620},
		},
		RowOrders: map[int][]string{
			0: {"A"},
			1: {"B"},
			2: {"C"},
		},
	}
}

func buildTestDAG() *dag.DAG {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A", Row: 0})
	_ = g.AddNode(dag.Node{ID: "B", Row: 1})
	_ = g.AddNode(dag.Node{ID: "C", Row: 2})
	_ = g.AddEdge(dag.Edge{From: "A", To: "B"})
	_ = g.AddEdge(dag.Edge{From: "B", To: "C"})
	return g
}

func buildMultiColumnDAG() *dag.DAG {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A", Row: 0})
	_ = g.AddNode(dag.Node{ID: "B", Row: 0})
	_ = g.AddNode(dag.Node{ID: "C", Row: 1})
	_ = g.AddNode(dag.Node{ID: "D", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "A", To: "C"})
	_ = g.AddEdge(dag.Edge{From: "B", To: "D"})
	return g
}
