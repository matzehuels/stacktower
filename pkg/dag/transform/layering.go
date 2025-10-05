package transform

import "stacktower/pkg/dag"

func AssignLayers(g *dag.DAG) {
	nodes := g.Nodes()
	inDegree := make(map[string]int, len(nodes))
	rows := make(map[string]int, len(nodes))
	queue := make([]string, 0, len(nodes))

	for _, n := range nodes {
		degree := g.InDegree(n.ID)
		inDegree[n.ID] = degree
		if degree == 0 {
			queue = append(queue, n.ID)
		}
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, child := range g.Children(curr) {
			if row := rows[curr] + 1; row > rows[child] {
				rows[child] = row
			}
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	g.SetRows(rows)
}
