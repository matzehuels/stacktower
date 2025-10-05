package ordering

import (
	"slices"

	"stacktower/pkg/dag"
)

func medianPosition(pos []int) (int, bool) {
	if len(pos) == 0 {
		return 0, false
	}
	sorted := slices.Clone(pos)
	slices.Sort(sorted)
	n := len(sorted)
	if n&1 == 0 {
		return sorted[n/2-1], true
	}
	return sorted[n/2], true
}

func barycenterDeviationIndices(g *dag.DAG, nodes []*dag.Node, indices []int, adjPos map[string]int, useParents bool) float64 {
	deviation := 0.0
	for i, idx := range indices {
		if idx >= len(nodes) {
			continue
		}
		node := nodes[idx]

		var neighbors []string
		if useParents {
			neighbors = g.Parents(node.EffectiveID())
		} else {
			neighbors = g.Children(node.EffectiveID())
		}

		sum, count := 0, 0
		for _, neighbor := range neighbors {
			if pos, ok := adjPos[neighbor]; ok {
				sum += pos
				count++
			}
		}

		if count > 0 {
			barycenter := float64(sum) / float64(count)
			delta := float64(i) - barycenter
			deviation += delta * delta
		}
	}
	return deviation
}
