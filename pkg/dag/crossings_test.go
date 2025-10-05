package dag

import "testing"

func TestCountLayerCrossings(t *testing.T) {
	tests := []struct {
		name  string
		setup func() (*DAG, []string, []string)
		want  int
	}{
		{
			name: "no crossings",
			setup: func() (*DAG, []string, []string) {
				g := New(nil)
				g.AddNode(Node{ID: "A", Row: 0})
				g.AddNode(Node{ID: "B", Row: 0})
				g.AddNode(Node{ID: "C", Row: 1})
				g.AddNode(Node{ID: "D", Row: 1})
				g.AddEdge(Edge{From: "A", To: "C"})
				g.AddEdge(Edge{From: "B", To: "D"})
				return g, []string{"A", "B"}, []string{"C", "D"}
			},
			want: 0,
		},
		{
			name: "one crossing",
			setup: func() (*DAG, []string, []string) {
				g := New(nil)
				g.AddNode(Node{ID: "A", Row: 0})
				g.AddNode(Node{ID: "B", Row: 0})
				g.AddNode(Node{ID: "C", Row: 1})
				g.AddNode(Node{ID: "D", Row: 1})
				g.AddEdge(Edge{From: "A", To: "D"})
				g.AddEdge(Edge{From: "B", To: "C"})
				return g, []string{"A", "B"}, []string{"C", "D"}
			},
			want: 1,
		},
		{
			name: "multiple crossings",
			setup: func() (*DAG, []string, []string) {
				g := New(nil)
				g.AddNode(Node{ID: "A", Row: 0})
				g.AddNode(Node{ID: "B", Row: 0})
				g.AddNode(Node{ID: "C", Row: 0})
				g.AddNode(Node{ID: "X", Row: 1})
				g.AddNode(Node{ID: "Y", Row: 1})
				g.AddNode(Node{ID: "Z", Row: 1})
				g.AddEdge(Edge{From: "A", To: "Z"})
				g.AddEdge(Edge{From: "B", To: "Y"})
				g.AddEdge(Edge{From: "C", To: "X"})
				return g, []string{"A", "B", "C"}, []string{"X", "Y", "Z"}
			},
			want: 3,
		},
		{
			name: "K23 complete bipartite",
			setup: func() (*DAG, []string, []string) {
				g := New(nil)
				g.AddNode(Node{ID: "A", Row: 0})
				g.AddNode(Node{ID: "B", Row: 0})
				g.AddNode(Node{ID: "C", Row: 1})
				g.AddNode(Node{ID: "D", Row: 1})
				g.AddNode(Node{ID: "E", Row: 1})
				g.AddEdge(Edge{From: "A", To: "C"})
				g.AddEdge(Edge{From: "A", To: "D"})
				g.AddEdge(Edge{From: "A", To: "E"})
				g.AddEdge(Edge{From: "B", To: "C"})
				g.AddEdge(Edge{From: "B", To: "D"})
				g.AddEdge(Edge{From: "B", To: "E"})
				return g, []string{"A", "B"}, []string{"C", "D", "E"}
			},
			want: 3,
		},
		{
			name: "empty layers",
			setup: func() (*DAG, []string, []string) {
				return New(nil), nil, nil
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, upper, lower := tt.setup()
			if got := CountLayerCrossings(g, upper, lower); got != tt.want {
				t.Errorf("CountLayerCrossings() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCountCrossings(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "A", Row: 0})
	g.AddNode(Node{ID: "B", Row: 0})
	g.AddNode(Node{ID: "C", Row: 1})
	g.AddNode(Node{ID: "D", Row: 1})
	g.AddNode(Node{ID: "E", Row: 2})
	g.AddNode(Node{ID: "F", Row: 2})
	g.AddEdge(Edge{From: "A", To: "D"})
	g.AddEdge(Edge{From: "B", To: "C"})
	g.AddEdge(Edge{From: "C", To: "F"})
	g.AddEdge(Edge{From: "D", To: "E"})

	orders := map[int][]string{
		0: {"A", "B"},
		1: {"C", "D"},
		2: {"E", "F"},
	}

	if got := CountCrossings(g, orders); got != 2 {
		t.Errorf("CountCrossings() = %d, want 2", got)
	}
}

func TestCountPairCrossings(t *testing.T) {
	tests := []struct {
		name  string
		setup func() (*DAG, string, string, []string, bool)
		want  int
	}{
		{
			name: "parents",
			setup: func() (*DAG, string, string, []string, bool) {
				g := New(nil)
				g.AddNode(Node{ID: "P1", Row: 0})
				g.AddNode(Node{ID: "P2", Row: 0})
				g.AddNode(Node{ID: "A", Row: 1})
				g.AddNode(Node{ID: "B", Row: 1})
				g.AddEdge(Edge{From: "P1", To: "B"})
				g.AddEdge(Edge{From: "P2", To: "A"})
				return g, "A", "B", []string{"P1", "P2"}, true
			},
			want: 1,
		},
		{
			name: "children",
			setup: func() (*DAG, string, string, []string, bool) {
				g := New(nil)
				g.AddNode(Node{ID: "A", Row: 0})
				g.AddNode(Node{ID: "B", Row: 0})
				g.AddNode(Node{ID: "C1", Row: 1})
				g.AddNode(Node{ID: "C2", Row: 1})
				g.AddEdge(Edge{From: "A", To: "C2"})
				g.AddEdge(Edge{From: "B", To: "C1"})
				return g, "A", "B", []string{"C1", "C2"}, false
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, left, right, adjOrder, useParents := tt.setup()
			if got := CountPairCrossings(g, left, right, adjOrder, useParents); got != tt.want {
				t.Errorf("CountPairCrossings() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPosMap(t *testing.T) {
	tests := []struct {
		name string
		ids  []string
		want map[string]int
	}{
		{
			name: "basic",
			ids:  []string{"A", "B", "C"},
			want: map[string]int{"A": 0, "B": 1, "C": 2},
		},
		{
			name: "empty",
			ids:  nil,
			want: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PosMap(tt.ids)
			if len(got) != len(tt.want) {
				t.Fatalf("PosMap() length = %d, want %d", len(got), len(tt.want))
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("PosMap()[%q] = %d, want %d", k, got[k], v)
				}
			}
		})
	}
}
