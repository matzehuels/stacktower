package transform

import "github.com/matzehuels/stacktower/pkg/core/dag"

// TransitiveReduction removes redundant edges from the graph.
//
// TransitiveReduction removes any edge (u, v) where there exists an alternate
// path from u to v through at least one intermediate node. For example, if
// edges A→B, B→C, and A→C all exist, then A→C is redundant and is removed
// because A reaches C via B.
//
// This simplifies visualization by showing only direct dependencies, which is
// critical for tower layouts where transitive edges create impossible geometry
// (a block cannot rest on both adjacent and distant floors simultaneously).
//
// # Algorithm
//
// TransitiveReduction computes full transitive closure using DFS-based
// reachability, then removes any edge (u, v) where u can reach v through an
// intermediate node w (where u→w and w reaches v).
//
// # Nil Handling
//
// TransitiveReduction panics if g is nil. If g is empty (zero nodes), the
// function returns immediately without error.
//
// # Performance
//
// Time complexity is O(V²·E) in the worst case, where V is the number of nodes
// and E is the number of edges. For sparse graphs (typical dependency graphs
// with limited fan-out), performance approaches O(V·E).
//
// Space complexity is O(V²) for the reachability matrix. For large dense
// graphs (thousands of nodes with high connectivity), this may consume
// significant memory.
//
// # Edge Metadata
//
// TransitiveReduction preserves edge metadata for all non-redundant edges.
// Metadata on removed edges is discarded.
func TransitiveReduction(g *dag.DAG) {
	nodes := g.Nodes()
	if len(nodes) == 0 {
		return
	}

	nodeIndex := dag.NodePosMap(nodes)
	adjacency := make([][]int, len(nodes))
	for _, e := range g.Edges() {
		if src, ok := nodeIndex[e.From]; ok {
			if dst, ok := nodeIndex[e.To]; ok {
				adjacency[src] = append(adjacency[src], dst)
			}
		}
	}

	reachability := computeReachability(adjacency)

	for _, e := range g.Edges() {
		src, dst := nodeIndex[e.From], nodeIndex[e.To]
		for _, intermediate := range adjacency[src] {
			if intermediate != dst && reachability[intermediate][dst] {
				g.RemoveEdge(e.From, e.To)
				break
			}
		}
	}
}

func computeReachability(adjacency [][]int) [][]bool {
	n := len(adjacency)
	reachable := make([][]bool, n)
	for i := range reachable {
		reachable[i] = make([]bool, n)
	}

	var dfs func(source, current int)
	dfs = func(source, current int) {
		if reachable[source][current] {
			return
		}
		reachable[source][current] = true
		for _, next := range adjacency[current] {
			dfs(source, next)
		}
	}

	for i := range reachable {
		dfs(i, i)
	}
	return reachable
}
