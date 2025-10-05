package transform

import "stacktower/pkg/dag"

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
