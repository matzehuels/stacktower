package transform

import "github.com/matzehuels/stacktower/pkg/core/dag"

// BreakCycles removes back-edges from the graph to ensure it is a valid
// directed acyclic graph (DAG).
//
// BreakCycles uses depth-first search with white/gray/black coloring to detect
// cycles. When a gray node is encountered (indicating a back-edge that would
// complete a cycle), that edge is marked for removal. The function returns the
// number of edges removed.
//
// # Algorithm
//
// The DFS starts from all source nodes (nodes with in-degree 0), then visits
// any remaining unvisited nodes to handle disconnected components. A node is:
//   - white: not yet visited
//   - gray: currently being visited (on the DFS stack)
//   - black: fully processed (all descendants visited)
//
// Any edge pointing to a gray node creates a cycle and is removed.
//
// # Edge Selection
//
// When multiple edges could break a cycle, the choice is deterministic but not
// guaranteed to minimize the total number removed across all cycles. For
// minimal cycle-breaking, consider using a feedback arc set algorithm instead.
//
// # Nil Handling
//
// BreakCycles panics if g is nil. If g is empty (zero nodes), it returns 0.
//
// # Performance
//
// Time complexity is O(V + E) where V is nodes and E is edges. Space
// complexity is O(V) for the color map and recursion stack.
func BreakCycles(g *dag.DAG) int {
	const (
		white = iota
		gray
		black
	)

	color := make(map[string]int)
	var backEdges [][2]string

	var dfs func(node string)
	dfs = func(node string) {
		color[node] = gray
		for _, child := range g.Children(node) {
			switch color[child] {
			case white:
				dfs(child)
			case gray:
				backEdges = append(backEdges, [2]string{node, child})
			}
		}
		color[node] = black
	}

	for _, n := range g.Sources() {
		if color[n.ID] == white {
			dfs(n.ID)
		}
	}

	for _, n := range g.Nodes() {
		if color[n.ID] == white {
			dfs(n.ID)
		}
	}

	for _, e := range backEdges {
		g.RemoveEdge(e[0], e[1])
	}
	return len(backEdges)
}
