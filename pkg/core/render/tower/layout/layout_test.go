package layout

import (
	"math"
	"testing"

	"github.com/matzehuels/stacktower/pkg/core/dag"
)

func TestBlock(t *testing.T) {
	tests := []struct {
		name   string
		block  Block
		width  float64
		height float64
	}{
		{"standard", Block{Left: 0, Right: 10, Bottom: 0, Top: 20}, 10.0, 20.0},
		{"zero size", Block{Left: 5, Right: 5, Bottom: 10, Top: 10}, 0.0, 0.0},
		{"offset", Block{Left: 100, Right: 150, Bottom: 50, Top: 100}, 50.0, 50.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.block.Width(); got != tt.width {
				t.Errorf("width: want %v, got %v", tt.width, got)
			}
			if got := tt.block.Height(); got != tt.height {
				t.Errorf("height: want %v, got %v", tt.height, got)
			}
		})
	}
}

func TestComputeWidths(t *testing.T) {
	tests := []struct {
		name      string
		nodes     []dag.Node
		edges     []dag.Edge
		rowOrders map[int][]string
		frame     float64
		want      map[string]float64
	}{
		{
			name:      "Empty",
			nodes:     nil,
			edges:     nil,
			rowOrders: map[int][]string{},
			frame:     100,
			want:      map[string]float64{},
		},
		{
			name:      "SingleRow",
			nodes:     []dag.Node{{ID: "a", Row: 0}, {ID: "b", Row: 0}, {ID: "c", Row: 0}},
			rowOrders: map[int][]string{0: {"a", "b", "c"}},
			frame:     120,
			want:      map[string]float64{"a": 40, "b": 40, "c": 40},
		},
		{
			name:      "Chain",
			nodes:     []dag.Node{{ID: "a", Row: 0}, {ID: "b", Row: 1}},
			edges:     []dag.Edge{{From: "a", To: "b"}},
			rowOrders: map[int][]string{0: {"a"}, 1: {"b"}},
			frame:     100,
			want:      map[string]float64{"a": 100, "b": 100},
		},
		{
			name:      "FanOut",
			nodes:     []dag.Node{{ID: "a", Row: 0}, {ID: "b", Row: 1}, {ID: "c", Row: 1}},
			edges:     []dag.Edge{{From: "a", To: "b"}, {From: "a", To: "c"}},
			rowOrders: map[int][]string{0: {"a"}, 1: {"b", "c"}},
			frame:     100,
			want:      map[string]float64{"a": 100, "b": 50, "c": 50},
		},
		{
			name:      "FanIn",
			nodes:     []dag.Node{{ID: "a", Row: 0}, {ID: "b", Row: 0}, {ID: "c", Row: 1}},
			edges:     []dag.Edge{{From: "a", To: "c"}, {From: "b", To: "c"}},
			rowOrders: map[int][]string{0: {"a", "b"}, 1: {"c"}},
			frame:     100,
			want:      map[string]float64{"a": 50, "b": 50, "c": 100},
		},
		{
			name: "Diamond",
			nodes: []dag.Node{
				{ID: "a", Row: 0}, {ID: "b", Row: 1},
				{ID: "c", Row: 1}, {ID: "d", Row: 2},
			},
			edges: []dag.Edge{
				{From: "a", To: "b"}, {From: "a", To: "c"},
				{From: "b", To: "d"}, {From: "c", To: "d"},
			},
			rowOrders: map[int][]string{0: {"a"}, 1: {"b", "c"}, 2: {"d"}},
			frame:     100,
			want:      map[string]float64{"a": 100, "b": 50, "c": 50, "d": 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := dag.New(nil)
			for _, n := range tt.nodes {
				_ = g.AddNode(n)
			}
			for _, e := range tt.edges {
				_ = g.AddEdge(e)
			}

			got := ComputeWidths(g, tt.rowOrders, tt.frame)

			if len(got) != len(tt.want) {
				t.Fatalf("width count: want %d, got %d", len(tt.want), len(got))
			}

			for id, want := range tt.want {
				if got[id] != want {
					t.Errorf("node %s: want %.1f, got %.1f", id, want, got[id])
				}
			}
		})
	}
}

func TestComputeWidths_Normalization(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "a", Row: 0})
	_ = g.AddNode(dag.Node{ID: "b", Row: 0})
	_ = g.AddNode(dag.Node{ID: "c", Row: 1})
	_ = g.AddEdge(dag.Edge{From: "a", To: "c"})

	rowOrders := map[int][]string{
		0: {"a", "b"},
		1: {"c"},
	}

	frameWidth := 100.0
	widths := ComputeWidths(g, rowOrders, frameWidth)

	for row, order := range rowOrders {
		total := 0.0
		for _, id := range order {
			total += widths[id]
		}
		if math.Abs(total-frameWidth) > eps {
			t.Errorf("row %d: want total %.1f, got %.1f", row, frameWidth, total)
		}
	}
}

func TestBuild_SimpleDiamond(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A", Row: 0})
	_ = g.AddNode(dag.Node{ID: "B", Row: 1})
	_ = g.AddNode(dag.Node{ID: "C", Row: 1})
	_ = g.AddNode(dag.Node{ID: "D", Row: 2})
	_ = g.AddEdge(dag.Edge{From: "A", To: "B"})
	_ = g.AddEdge(dag.Edge{From: "A", To: "C"})
	_ = g.AddEdge(dag.Edge{From: "B", To: "D"})
	_ = g.AddEdge(dag.Edge{From: "C", To: "D"})

	layout := Build(g, 100, 90)

	if layout.FrameWidth != 100 {
		t.Errorf("FrameWidth: want 100, got %v", layout.FrameWidth)
	}
	if layout.FrameHeight != 90 {
		t.Errorf("FrameHeight: want 90, got %v", layout.FrameHeight)
	}

	if len(layout.Blocks) != 4 {
		t.Fatalf("want 4 blocks, got %d", len(layout.Blocks))
	}

	if len(layout.RowOrders) != 3 {
		t.Fatalf("want 3 rows, got %d", len(layout.RowOrders))
	}

	// Row 0: A
	if len(layout.RowOrders[0]) != 1 || layout.RowOrders[0][0] != "A" {
		t.Errorf("row 0 order: want [A], got %v", layout.RowOrders[0])
	}

	// Row 1: B, C (alphabetical)
	if len(layout.RowOrders[1]) != 2 {
		t.Fatalf("row 1: want 2 nodes, got %d", len(layout.RowOrders[1]))
	}
	if layout.RowOrders[1][0] != "B" || layout.RowOrders[1][1] != "C" {
		t.Errorf("row 1 order: want [B C], got %v", layout.RowOrders[1])
	}

	// Row 2: D
	if len(layout.RowOrders[2]) != 1 || layout.RowOrders[2][0] != "D" {
		t.Errorf("row 2 order: want [D], got %v", layout.RowOrders[2])
	}

	// Check row width totals (should equal effective width, not frame width)
	effectiveWidth := layout.FrameWidth - 2*layout.MarginX
	for row, order := range layout.RowOrders {
		total := 0.0
		for _, id := range order {
			total += layout.Blocks[id].Width()
		}
		if math.Abs(total-effectiveWidth) > eps {
			t.Errorf("row %d width: want %.1f, got %.1f", row, effectiveWidth, total)
		}
	}

	// Check vertical stacking (root at top, sink at bottom)
	margin := layout.MarginY
	if layout.Blocks["A"].Bottom != margin {
		t.Errorf("row 0 (A) should be at top with margin: got bottom=%v, want %v",
			layout.Blocks["A"].Bottom, margin)
	}
	if layout.Blocks["D"].Top != layout.FrameHeight-margin {
		t.Errorf("row 2 (D) should be at bottom with margin: got top=%v, want %v",
			layout.Blocks["D"].Top, layout.FrameHeight-margin)
	}

	// Check horizontal margins
	if layout.Blocks["A"].Left != layout.MarginX {
		t.Errorf("blocks should start at left margin: got %v, want %v",
			layout.Blocks["A"].Left, layout.MarginX)
	}
}

func TestBuild_WithAuxiliary(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A", Row: 0})
	_ = g.AddNode(dag.Node{ID: "aux", Row: 1, Kind: dag.NodeKindAuxiliary})
	_ = g.AddNode(dag.Node{ID: "B", Row: 2})

	layout := Build(g, 100, 100)

	auxBlock := layout.Blocks["aux"]
	regBlock := layout.Blocks["A"]

	// Auxiliary should be 1/5 height of regular node
	ratio := auxBlock.Height() / regBlock.Height()
	if math.Abs(ratio-0.2) > 0.01 {
		t.Errorf("auxiliary height ratio: want 0.2, got %.3f", ratio)
	}
}

func TestBuild_WithSubdividers(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A", Row: 0})
	_ = g.AddNode(dag.Node{ID: "sep", Row: 1, Kind: dag.NodeKindSubdivider})
	_ = g.AddNode(dag.Node{ID: "B", Row: 2})

	layout := Build(g, 100, 100)

	sepBlock := layout.Blocks["sep"]
	regBlock := layout.Blocks["A"]

	// Subdividers (separators) should have same height as regular nodes
	ratio := sepBlock.Height() / regBlock.Height()
	if math.Abs(ratio-1.0) > 0.01 {
		t.Errorf("subdivider height ratio: want 1.0, got %.3f", ratio)
	}
}

func TestBuild_WithMixedAuxiliary(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A", Row: 0})
	_ = g.AddNode(dag.Node{ID: "B", Row: 0})
	_ = g.AddNode(dag.Node{ID: "sep", Row: 0, Kind: dag.NodeKindAuxiliary})
	_ = g.AddNode(dag.Node{ID: "C", Row: 1})

	layout := Build(g, 100, 100)

	sepBlock := layout.Blocks["sep"]
	regBlockA := layout.Blocks["A"]
	regBlockB := layout.Blocks["B"]
	regBlockC := layout.Blocks["C"]

	// When auxiliary nodes are mixed with regular nodes in same row,
	// the entire row should have normal height (not shrunk)
	if math.Abs(sepBlock.Height()-regBlockA.Height()) > 0.01 {
		t.Errorf("mixed row: auxiliary and regular should have same height, got %.3f and %.3f",
			sepBlock.Height(), regBlockA.Height())
	}

	if math.Abs(regBlockA.Height()-regBlockC.Height()) > 0.01 {
		t.Errorf("row 0 and row 1 should have same height when row 0 is mixed, got %.3f and %.3f",
			regBlockA.Height(), regBlockC.Height())
	}

	// All nodes in the same row should have the same height
	if math.Abs(regBlockA.Height()-regBlockB.Height()) > 0.01 {
		t.Errorf("all nodes in row 0 should have same height, got A=%.3f, B=%.3f",
			regBlockA.Height(), regBlockB.Height())
	}
}

func TestBuild_WithSeparatorRow(t *testing.T) {
	// Test the real-world case: separators in their own row between parents and children
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "P1", Row: 0})
	_ = g.AddNode(dag.Node{ID: "P2", Row: 0})
	_ = g.AddNode(dag.Node{ID: "Sep", Row: 1, Kind: dag.NodeKindAuxiliary})
	_ = g.AddNode(dag.Node{ID: "C1", Row: 2})
	_ = g.AddNode(dag.Node{ID: "C2", Row: 2})

	layout := Build(g, 100, 100)

	sepBlock := layout.Blocks["Sep"]
	p1Block := layout.Blocks["P1"]
	c1Block := layout.Blocks["C1"]

	// Separator row should be shorter (20% of normal)
	ratio := sepBlock.Height() / p1Block.Height()
	if math.Abs(ratio-0.2) > 0.01 {
		t.Errorf("separator row height ratio: want 0.2, got %.3f", ratio)
	}

	// Parent and child rows should have same (normal) height
	if math.Abs(p1Block.Height()-c1Block.Height()) > 0.01 {
		t.Errorf("parent and child rows should have same height, got %.3f and %.3f",
			p1Block.Height(), c1Block.Height())
	}
}

func TestBuild_RootAtTop(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A", Row: 0}) // root (top)
	_ = g.AddNode(dag.Node{ID: "B", Row: 1}) // dependency (bottom)

	layout := Build(g, 100, 100)

	// Row 0 (root) should be at top with margin
	if layout.Blocks["A"].Bottom != layout.MarginY {
		t.Errorf("row 0 should be at top with margin, got bottom=%v, want %v",
			layout.Blocks["A"].Bottom, layout.MarginY)
	}

	// Row 1 (dependency) should be below row 0
	if layout.Blocks["B"].Bottom < layout.Blocks["A"].Top {
		t.Errorf("row 1 should be below row 0, got B.Bottom=%v, A.Top=%v",
			layout.Blocks["B"].Bottom, layout.Blocks["A"].Top)
	}
}

func TestBuild_WithOptions(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A", Row: 0})
	_ = g.AddNode(dag.Node{ID: "B", Row: 1})

	layout := Build(g, 200, 150,
		WithAuxiliaryRatio(0.1),
	)

	if layout.FrameWidth != 200 {
		t.Errorf("FrameWidth: want 200, got %v", layout.FrameWidth)
	}
	if layout.FrameHeight != 150 {
		t.Errorf("FrameHeight: want 150, got %v", layout.FrameHeight)
	}

	if len(layout.Blocks) != 2 {
		t.Fatalf("want 2 blocks, got %d", len(layout.Blocks))
	}
}

func TestBuild_WithMargins(t *testing.T) {
	g := dag.New(nil)
	_ = g.AddNode(dag.Node{ID: "A", Row: 0})
	_ = g.AddNode(dag.Node{ID: "B", Row: 1})

	// Test with default margin (5%)
	layout := Build(g, 100, 100)

	expectedMarginX := 100 * 0.05
	expectedMarginY := 100 * 0.05

	if layout.MarginX != expectedMarginX {
		t.Errorf("MarginX: want %.1f, got %.1f", expectedMarginX, layout.MarginX)
	}
	if layout.MarginY != expectedMarginY {
		t.Errorf("MarginY: want %.1f, got %.1f", expectedMarginY, layout.MarginY)
	}

	// Blocks should not touch frame edges
	for _, block := range layout.Blocks {
		if block.Left < layout.MarginX {
			t.Errorf("Block %s left %.1f < margin %.1f", block.NodeID, block.Left, layout.MarginX)
		}
		if block.Right > layout.FrameWidth-layout.MarginX {
			t.Errorf("Block %s right %.1f > frame-margin %.1f",
				block.NodeID, block.Right, layout.FrameWidth-layout.MarginX)
		}
		if block.Bottom < layout.MarginY {
			t.Errorf("Block %s bottom %.1f < margin %.1f", block.NodeID, block.Bottom, layout.MarginY)
		}
		if block.Top > layout.FrameHeight-layout.MarginY {
			t.Errorf("Block %s top %.1f > frame-margin %.1f",
				block.NodeID, block.Top, layout.FrameHeight-layout.MarginY)
		}
	}

	// Test with custom margin (10%)
	layoutCustom := Build(g, 100, 100, WithMarginRatio(0.1))

	if layoutCustom.MarginX != 10 {
		t.Errorf("Custom MarginX: want 10, got %.1f", layoutCustom.MarginX)
	}
	if layoutCustom.MarginY != 10 {
		t.Errorf("Custom MarginY: want 10, got %.1f", layoutCustom.MarginY)
	}
}
