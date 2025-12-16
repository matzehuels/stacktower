package transform

import "github.com/matzehuels/stacktower/pkg/dag"

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
