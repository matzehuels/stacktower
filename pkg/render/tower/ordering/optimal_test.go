package ordering

import (
	"slices"
	"testing"
	"time"

	"stacktower/pkg/dag"
)

func TestOptimalSearch_Empty(t *testing.T) {
	got := OptimalSearch{}.OrderRows(dag.New(nil))
	if got != nil {
		t.Errorf("want nil, got %v", got)
	}
}

func TestOptimalSearch_SingleNode(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "A", Row: 0})

	got := OptimalSearch{}.OrderRows(g)

	if !slices.Equal(got[0], []string{"A"}) {
		t.Errorf("want [A], got %v", got[0])
	}
}

func TestOptimalSearch_Diamond(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "A", Row: 0})
	g.AddNode(dag.Node{ID: "B", Row: 1})
	g.AddNode(dag.Node{ID: "C", Row: 1})
	g.AddNode(dag.Node{ID: "D", Row: 2})
	g.AddEdge(dag.Edge{From: "A", To: "B"})
	g.AddEdge(dag.Edge{From: "A", To: "C"})
	g.AddEdge(dag.Edge{From: "B", To: "D"})
	g.AddEdge(dag.Edge{From: "C", To: "D"})

	got := OptimalSearch{}.OrderRows(g)
	crossings := dag.CountCrossings(g, got)

	if crossings != 0 {
		t.Errorf("want 0 crossings, got %d with ordering %v", crossings, got)
	}
}

func TestOptimalSearch_CrossingReduction(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "A", Row: 0})
	g.AddNode(dag.Node{ID: "B", Row: 0})
	g.AddNode(dag.Node{ID: "C", Row: 1})
	g.AddNode(dag.Node{ID: "D", Row: 1})
	g.AddEdge(dag.Edge{From: "A", To: "D"})
	g.AddEdge(dag.Edge{From: "B", To: "C"})

	got := OptimalSearch{}.OrderRows(g)
	crossings := dag.CountCrossings(g, got)

	if crossings != 0 {
		t.Errorf("want 0 crossings, got %d with ordering %v", crossings, got)
	}
}

func TestOptimalSearch_RespectsChains(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "A", Row: 0})
	g.AddNode(dag.Node{ID: "A_sub_1", Row: 1, Kind: dag.NodeKindSubdivider, MasterID: "A"})
	g.AddNode(dag.Node{ID: "X", Row: 1})
	g.AddNode(dag.Node{ID: "B", Row: 2})
	g.AddEdge(dag.Edge{From: "A", To: "A_sub_1"})
	g.AddEdge(dag.Edge{From: "A", To: "X"})
	g.AddEdge(dag.Edge{From: "A_sub_1", To: "B"})
	g.AddEdge(dag.Edge{From: "X", To: "B"})

	got := OptimalSearch{}.OrderRows(g)

	if len(got[1]) != 2 {
		t.Errorf("row 1: want 2 nodes, got %d", len(got[1]))
	}

	crossings := dag.CountCrossings(g, got)
	if crossings != 0 {
		t.Errorf("want 0 crossings, got %d", crossings)
	}
}

func TestOptimalSearch_FindsOptimal(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "A", Row: 0})
	g.AddNode(dag.Node{ID: "B", Row: 0})
	g.AddNode(dag.Node{ID: "C", Row: 1})
	g.AddNode(dag.Node{ID: "D", Row: 1})
	g.AddEdge(dag.Edge{From: "A", To: "C"})
	g.AddEdge(dag.Edge{From: "A", To: "D"})
	g.AddEdge(dag.Edge{From: "B", To: "C"})
	g.AddEdge(dag.Edge{From: "B", To: "D"})

	optimal := OptimalSearch{}.OrderRows(g)
	barycentric := Barycentric{}.OrderRows(g)

	optScore := dag.CountCrossings(g, optimal)
	bcScore := dag.CountCrossings(g, barycentric)

	if optScore > bcScore {
		t.Errorf("optimal search should be optimal: opt=%d, bc=%d", optScore, bcScore)
	}
}

func TestOptimalSearch_Pruning(t *testing.T) {
	g := dag.New(nil)
	g.AddNode(dag.Node{ID: "A", Row: 0})
	g.AddNode(dag.Node{ID: "B", Row: 0})
	g.AddNode(dag.Node{ID: "C", Row: 0})
	g.AddNode(dag.Node{ID: "D", Row: 1})
	g.AddNode(dag.Node{ID: "E", Row: 1})
	g.AddNode(dag.Node{ID: "F", Row: 1})
	g.AddEdge(dag.Edge{From: "A", To: "F"})
	g.AddEdge(dag.Edge{From: "B", To: "E"})
	g.AddEdge(dag.Edge{From: "C", To: "D"})

	got := OptimalSearch{}.OrderRows(g)
	score := dag.CountCrossings(g, got)

	if score != 0 {
		t.Errorf("should find zero-crossing solution, got %d crossings", score)
	}
}

func TestOptimalSearch_Progress(t *testing.T) {
	g := dag.New(nil)
	for i := 0; i < 4; i++ {
		g.AddNode(dag.Node{ID: string(rune('A' + i)), Row: 0})
		g.AddNode(dag.Node{ID: string(rune('E' + i)), Row: 1})
		g.AddNode(dag.Node{ID: string(rune('I' + i)), Row: 2})
	}
	g.AddEdge(dag.Edge{From: "A", To: "H"})
	g.AddEdge(dag.Edge{From: "B", To: "G"})
	g.AddEdge(dag.Edge{From: "C", To: "F"})
	g.AddEdge(dag.Edge{From: "D", To: "E"})
	g.AddEdge(dag.Edge{From: "E", To: "L"})
	g.AddEdge(dag.Edge{From: "F", To: "K"})
	g.AddEdge(dag.Edge{From: "G", To: "J"})
	g.AddEdge(dag.Edge{From: "H", To: "I"})

	var lastExplored, lastPruned, lastScore int
	updates := 0

	opt := OptimalSearch{
		Progress: func(explored, pruned, score int) {
			lastExplored = explored
			lastPruned = pruned
			lastScore = score
			updates++
			t.Logf("Progress: explored=%d, pruned=%d, best=%d", explored, pruned, score)
		},
	}

	got := opt.OrderRows(g)
	finalScore := dag.CountCrossings(g, got)

	if updates == 0 {
		t.Error("expected progress updates, got none")
	}
	if lastExplored == 0 && lastPruned == 0 {
		t.Error("expected explorations or pruning to be tracked")
	}
	if lastScore != finalScore && lastScore >= 0 {
		t.Errorf("last reported score %d != final score %d", lastScore, finalScore)
	}

	t.Logf("Final: explored=%d, pruned=%d, score=%d, updates=%d", lastExplored, lastPruned, finalScore, updates)
}

func TestOptimalSearch_Timeout(t *testing.T) {
	g := dag.New(nil)
	for i := 0; i < 6; i++ {
		g.AddNode(dag.Node{ID: string(rune('A' + i)), Row: 0})
		g.AddNode(dag.Node{ID: string(rune('G' + i)), Row: 1})
	}

	for i := 0; i < 6; i++ {
		for j := 0; j < 6; j++ {
			g.AddEdge(dag.Edge{
				From: string(rune('A' + i)),
				To:   string(rune('G' + ((i + j) % 6))),
			})
		}
	}

	opt := OptimalSearch{
		Timeout: 100 * time.Millisecond,
	}

	got := opt.OrderRows(g)

	if got == nil {
		t.Error("expected fallback result after timeout, got nil")
	}

	t.Logf("Timed out as expected, returned fallback ordering")
}

func TestOptimalSearch_LargerGraph(t *testing.T) {
	g := dag.New(nil)

	// Create a 5x5 grid-like graph
	for row := 0; row < 5; row++ {
		for col := 0; col < 5; col++ {
			id := string(rune('A' + row*5 + col))
			g.AddNode(dag.Node{ID: id, Row: row})
		}
	}

	// Add edges creating crossings
	for row := 0; row < 4; row++ {
		for col := 0; col < 5; col++ {
			from := string(rune('A' + row*5 + col))
			// Connect to offset positions in next row
			to := string(rune('A' + (row+1)*5 + (4 - col)))
			g.AddEdge(dag.Edge{From: from, To: to})
		}
	}

	opt := OptimalSearch{
		Timeout: 2 * time.Second,
	}

	got := opt.OrderRows(g)
	if got == nil {
		t.Fatal("expected result, got nil")
	}

	score := dag.CountCrossings(g, got)
	t.Logf("Score: %d crossings for 5x5 graph", score)
}
