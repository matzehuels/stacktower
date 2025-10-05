package dag

import "testing"

func TestNew(t *testing.T) {
	g := New(nil)
	if g.NodeCount() != 0 {
		t.Errorf("NodeCount() = %d, want 0", g.NodeCount())
	}
	if g.EdgeCount() != 0 {
		t.Errorf("EdgeCount() = %d, want 0", g.EdgeCount())
	}
}

func TestAddNode(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *DAG
		node    Node
		wantErr error
		wantCnt int
	}{
		{
			name:    "valid node",
			setup:   func() *DAG { return New(nil) },
			node:    Node{ID: "a", Row: 0},
			wantErr: nil,
			wantCnt: 1,
		},
		{
			name:    "empty ID",
			setup:   func() *DAG { return New(nil) },
			node:    Node{ID: "", Row: 0},
			wantErr: ErrInvalidNodeID,
			wantCnt: 0,
		},
		{
			name: "duplicate ID",
			setup: func() *DAG {
				g := New(nil)
				g.AddNode(Node{ID: "a", Row: 0})
				return g
			},
			node:    Node{ID: "a", Row: 0},
			wantErr: ErrDuplicateNodeID,
			wantCnt: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.setup()
			err := g.AddNode(tt.node)
			if err != tt.wantErr {
				t.Errorf("AddNode() error = %v, want %v", err, tt.wantErr)
			}
			if g.NodeCount() != tt.wantCnt {
				t.Errorf("NodeCount() = %d, want %d", g.NodeCount(), tt.wantCnt)
			}
		})
	}
}

func TestAddEdge(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *DAG
		edge    Edge
		wantErr error
		wantCnt int
	}{
		{
			name: "valid edge",
			setup: func() *DAG {
				g := New(nil)
				g.AddNode(Node{ID: "a", Row: 0})
				g.AddNode(Node{ID: "b", Row: 1})
				return g
			},
			edge:    Edge{From: "a", To: "b"},
			wantErr: nil,
			wantCnt: 1,
		},
		{
			name: "unknown source",
			setup: func() *DAG {
				g := New(nil)
				g.AddNode(Node{ID: "b", Row: 1})
				return g
			},
			edge:    Edge{From: "a", To: "b"},
			wantErr: ErrUnknownSourceNode,
			wantCnt: 0,
		},
		{
			name: "unknown target",
			setup: func() *DAG {
				g := New(nil)
				g.AddNode(Node{ID: "a", Row: 0})
				return g
			},
			edge:    Edge{From: "a", To: "b"},
			wantErr: ErrUnknownTargetNode,
			wantCnt: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.setup()
			err := g.AddEdge(tt.edge)
			if err != tt.wantErr {
				t.Errorf("AddEdge() error = %v, want %v", err, tt.wantErr)
			}
			if g.EdgeCount() != tt.wantCnt {
				t.Errorf("EdgeCount() = %d, want %d", g.EdgeCount(), tt.wantCnt)
			}
		})
	}
}

func TestChildren(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "a", Row: 0})
	g.AddNode(Node{ID: "b", Row: 1})
	g.AddNode(Node{ID: "c", Row: 1})
	g.AddEdge(Edge{From: "a", To: "b"})
	g.AddEdge(Edge{From: "a", To: "c"})

	children := g.Children("a")
	if len(children) != 2 {
		t.Errorf("Children() count = %d, want 2", len(children))
	}
}

func TestParents(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "a", Row: 0})
	g.AddNode(Node{ID: "b", Row: 0})
	g.AddNode(Node{ID: "c", Row: 1})
	g.AddEdge(Edge{From: "a", To: "c"})
	g.AddEdge(Edge{From: "b", To: "c"})

	parents := g.Parents("c")
	if len(parents) != 2 {
		t.Errorf("Parents() count = %d, want 2", len(parents))
	}
}

func TestNodesInRow(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "a", Row: 0})
	g.AddNode(Node{ID: "b", Row: 0})
	g.AddNode(Node{ID: "c", Row: 1})

	if got := len(g.NodesInRow(0)); got != 2 {
		t.Errorf("NodesInRow(0) count = %d, want 2", got)
	}
	if got := len(g.NodesInRow(1)); got != 1 {
		t.Errorf("NodesInRow(1) count = %d, want 1", got)
	}
}

func TestRowIDs(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "a", Row: 2})
	g.AddNode(Node{ID: "b", Row: 0})
	g.AddNode(Node{ID: "c", Row: 1})

	rows := g.RowIDs()
	want := []int{0, 1, 2}
	if len(rows) != len(want) {
		t.Fatalf("RowIDs() count = %d, want %d", len(rows), len(want))
	}
	for i, r := range rows {
		if r != want[i] {
			t.Errorf("RowIDs()[%d] = %d, want %d", i, r, want[i])
		}
	}
}

func TestSources(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "a", Row: 0})
	g.AddNode(Node{ID: "b", Row: 0})
	g.AddNode(Node{ID: "c", Row: 1})
	g.AddEdge(Edge{From: "a", To: "c"})

	sources := g.Sources()
	if len(sources) != 2 {
		t.Errorf("Sources() count = %d, want 2", len(sources))
	}
}

func TestSinks(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "a", Row: 0})
	g.AddNode(Node{ID: "b", Row: 1})
	g.AddNode(Node{ID: "c", Row: 1})
	g.AddEdge(Edge{From: "a", To: "b"})

	sinks := g.Sinks()
	if len(sinks) != 2 {
		t.Errorf("Sinks() count = %d, want 2", len(sinks))
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *DAG
		wantErr error
	}{
		{
			name: "valid DAG",
			setup: func() *DAG {
				g := New(nil)
				g.AddNode(Node{ID: "a", Row: 0})
				g.AddNode(Node{ID: "b", Row: 1})
				g.AddNode(Node{ID: "c", Row: 2})
				g.AddEdge(Edge{From: "a", To: "b"})
				g.AddEdge(Edge{From: "b", To: "c"})
				return g
			},
			wantErr: nil,
		},
		{
			name: "non-consecutive rows",
			setup: func() *DAG {
				g := New(nil)
				g.AddNode(Node{ID: "a", Row: 0})
				g.AddNode(Node{ID: "b", Row: 2})
				g.AddEdge(Edge{From: "a", To: "b"})
				return g
			},
			wantErr: ErrNonConsecutiveRows,
		},
		{
			name: "cycle detected",
			setup: func() *DAG {
				g := New(nil)
				g.AddNode(Node{ID: "a", Row: 0})
				g.AddNode(Node{ID: "b", Row: 1})
				g.AddNode(Node{ID: "c", Row: 2})
				g.AddEdge(Edge{From: "a", To: "b"})
				g.AddEdge(Edge{From: "b", To: "c"})
				g.incoming["a"] = []string{"c"}
				g.outgoing["c"] = append(g.outgoing["c"], "a")
				return g
			},
			wantErr: ErrGraphHasCycle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := tt.setup()
			if err := g.Validate(); err != tt.wantErr {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNodeKind(t *testing.T) {
	tests := []struct {
		kind         NodeKind
		isSubdivider bool
		isAuxiliary  bool
		isSynthetic  bool
	}{
		{NodeKindRegular, false, false, false},
		{NodeKindSubdivider, true, false, true},
		{NodeKindAuxiliary, false, true, true},
	}

	for _, tt := range tests {
		n := Node{Kind: tt.kind}
		if got := n.IsSubdivider(); got != tt.isSubdivider {
			t.Errorf("Node{Kind: %d}.IsSubdivider() = %v, want %v", tt.kind, got, tt.isSubdivider)
		}
		if got := n.IsAuxiliary(); got != tt.isAuxiliary {
			t.Errorf("Node{Kind: %d}.IsAuxiliary() = %v, want %v", tt.kind, got, tt.isAuxiliary)
		}
		if got := n.IsSynthetic(); got != tt.isSynthetic {
			t.Errorf("Node{Kind: %d}.IsSynthetic() = %v, want %v", tt.kind, got, tt.isSynthetic)
		}
	}
}

func TestNodes(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "a"})
	g.AddNode(Node{ID: "b"})
	g.AddNode(Node{ID: "c"})

	nodes := g.Nodes()
	if len(nodes) != 3 {
		t.Errorf("Nodes() count = %d, want 3", len(nodes))
	}

	ids := make(map[string]bool)
	for _, n := range nodes {
		ids[n.ID] = true
	}
	for _, want := range []string{"a", "b", "c"} {
		if !ids[want] {
			t.Errorf("Nodes() missing node %q", want)
		}
	}
}

func TestNode(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "test", Row: 5, Meta: Metadata{"key": "value"}})

	node, ok := g.Node("test")
	if !ok {
		t.Fatal("Node(test) not found")
	}
	if node.ID != "test" {
		t.Errorf("Node.ID = %q, want %q", node.ID, "test")
	}
	if node.Row != 5 {
		t.Errorf("Node.Row = %d, want 5", node.Row)
	}
	if node.Meta["key"] != "value" {
		t.Error("Node.Meta not preserved")
	}

	if _, ok := g.Node("nonexistent"); ok {
		t.Error("Node(nonexistent) should return false")
	}
}

func TestEdges(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "a"})
	g.AddNode(Node{ID: "b"})
	g.AddNode(Node{ID: "c"})
	g.AddEdge(Edge{From: "a", To: "b"})
	g.AddEdge(Edge{From: "b", To: "c"})

	edges := g.Edges()
	if len(edges) != 2 {
		t.Errorf("Edges() count = %d, want 2", len(edges))
	}
}

func TestRemoveEdge(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "a"})
	g.AddNode(Node{ID: "b"})
	g.AddNode(Node{ID: "c"})
	g.AddEdge(Edge{From: "a", To: "b"})
	g.AddEdge(Edge{From: "b", To: "c"})

	g.RemoveEdge("a", "b")

	if got := g.EdgeCount(); got != 1 {
		t.Errorf("EdgeCount() = %d after removal, want 1", got)
	}
	if got := len(g.Children("a")); got != 0 {
		t.Errorf("Children(a) count = %d after removal, want 0", got)
	}
}

func TestOutDegree(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "a"})
	g.AddNode(Node{ID: "b"})
	g.AddNode(Node{ID: "c"})
	g.AddEdge(Edge{From: "a", To: "b"})
	g.AddEdge(Edge{From: "a", To: "c"})

	if got := g.OutDegree("a"); got != 2 {
		t.Errorf("OutDegree(a) = %d, want 2", got)
	}
	if got := g.OutDegree("b"); got != 0 {
		t.Errorf("OutDegree(b) = %d, want 0", got)
	}
}

func TestChildrenInRow(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "a", Row: 0})
	g.AddNode(Node{ID: "b", Row: 1})
	g.AddNode(Node{ID: "c", Row: 2})
	g.AddNode(Node{ID: "d", Row: 1})
	g.AddEdge(Edge{From: "a", To: "b"})
	g.AddEdge(Edge{From: "a", To: "c"})
	g.AddEdge(Edge{From: "a", To: "d"})

	tests := []struct {
		row  int
		want int
	}{
		{1, 2},
		{2, 1},
		{3, 0},
	}

	for _, tt := range tests {
		if got := len(g.ChildrenInRow("a", tt.row)); got != tt.want {
			t.Errorf("ChildrenInRow(a, %d) count = %d, want %d", tt.row, got, tt.want)
		}
	}
}

func TestParentsInRow(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "a", Row: 0})
	g.AddNode(Node{ID: "b", Row: 1})
	g.AddNode(Node{ID: "c", Row: 0})
	g.AddNode(Node{ID: "d", Row: 2})
	g.AddEdge(Edge{From: "a", To: "d"})
	g.AddEdge(Edge{From: "b", To: "d"})
	g.AddEdge(Edge{From: "c", To: "d"})

	tests := []struct {
		row  int
		want int
	}{
		{0, 2},
		{1, 1},
		{2, 0},
	}

	for _, tt := range tests {
		if got := len(g.ParentsInRow("d", tt.row)); got != tt.want {
			t.Errorf("ParentsInRow(d, %d) count = %d, want %d", tt.row, got, tt.want)
		}
	}
}

func TestSetRows(t *testing.T) {
	g := New(nil)
	g.AddNode(Node{ID: "a"})
	g.AddNode(Node{ID: "b"})

	g.SetRows(map[string]int{
		"a": 0,
		"b": 1,
	})

	nodeA, _ := g.Node("a")
	nodeB, _ := g.Node("b")

	if nodeA.Row != 0 {
		t.Errorf("Node(a).Row = %d, want 0", nodeA.Row)
	}
	if nodeB.Row != 1 {
		t.Errorf("Node(b).Row = %d, want 1", nodeB.Row)
	}
}

func TestMeta(t *testing.T) {
	meta := Metadata{"graph": "test"}
	g := New(meta)

	result := g.Meta()
	if result["graph"] != "test" {
		t.Errorf("Meta()[graph] = %v, want %q", result["graph"], "test")
	}
}
