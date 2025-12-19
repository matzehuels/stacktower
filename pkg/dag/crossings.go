package dag

import (
	"maps"
	"slices"
)

// CrossingWorkspace provides reusable buffers for crossing calculations.
// Create with NewCrossingWorkspace and reuse across multiple calls.
type CrossingWorkspace struct {
	ft  []int
	pos []int
}

// NewCrossingWorkspace creates a workspace for counting crossings.
// maxWidth should be the maximum number of nodes in any row.
func NewCrossingWorkspace(maxWidth int) *CrossingWorkspace {
	return &CrossingWorkspace{
		ft:  make([]int, maxWidth+2),
		pos: make([]int, maxWidth+2),
	}
}

// CountCrossings returns the total edge crossings for the given row orderings.
func CountCrossings(g *DAG, orders map[int][]string) int {
	rows := slices.Sorted(maps.Keys(orders))
	crossings := 0
	for i := 0; i < len(rows)-1; i++ {
		r := rows[i]
		crossings += CountLayerCrossings(g, orders[r], orders[r+1])
	}
	return crossings
}

// CountLayerCrossings counts edge crossings between two adjacent rows.
// Uses a Fenwick tree for O(E log V) performance.
func CountLayerCrossings(g *DAG, upper, lower []string) int {
	if len(upper) == 0 || len(lower) == 0 {
		return 0
	}

	lowerPos := PosMap(lower)

	type edge struct{ upper, lower int }
	edges := make([]edge, 0, len(upper)*2)
	for i, nodeID := range upper {
		for _, child := range g.Children(nodeID) {
			if pos, ok := lowerPos[child]; ok {
				edges = append(edges, edge{i, pos})
			}
		}
	}
	if len(edges) < 2 {
		return 0
	}

	slices.SortFunc(edges, func(a, b edge) int {
		if a.upper != b.upper {
			return a.upper - b.upper
		}
		return a.lower - b.lower
	})

	fenwick := make([]int, len(lower)+1)
	crossings, total := 0, 0
	for _, e := range edges {
		lessOrEqual := 0
		for q := e.lower + 1; q > 0; q -= q & (-q) {
			lessOrEqual += fenwick[q]
		}
		crossings += total - lessOrEqual

		total++
		for idx := e.lower + 1; idx < len(fenwick); idx += idx & (-idx) {
			fenwick[idx]++
		}
	}
	return crossings
}

// CountCrossingsIdx counts crossings using index-based edges and permutations.
// This is an optimized version for the branch-and-bound search that avoids
// string lookups. edges[i] contains the indices of children for upper node i.
func CountCrossingsIdx(edges [][]int, upperPerm, lowerPerm []int, ws *CrossingWorkspace) int {
	if len(upperPerm) == 0 || len(lowerPerm) == 0 {
		return 0
	}

	for pos, origIdx := range lowerPerm {
		ws.pos[origIdx] = pos
	}

	limit := len(lowerPerm) + 1
	for i := 0; i < limit; i++ {
		ws.ft[i] = 0
	}

	crossings, total := 0, 0
	for _, upperIdx := range upperPerm {
		targets := edges[upperIdx]
		for _, targetIdx := range targets {
			targetPos := ws.pos[targetIdx]
			lessOrEqual := 0
			for q := targetPos + 1; q > 0; q -= q & (-q) {
				lessOrEqual += ws.ft[q]
			}
			crossings += total - lessOrEqual
		}

		for _, targetIdx := range targets {
			targetPos := ws.pos[targetIdx]
			total++
			for idx := targetPos + 1; idx < limit; idx += idx & (-idx) {
				ws.ft[idx]++
			}
		}
	}
	return crossings
}

// CountPairCrossings counts how many crossings swapping left and right would cause.
// If useParents is true, considers edges to the row above; otherwise, to the row below.
func CountPairCrossings(g *DAG, left, right string, adjOrder []string, useParents bool) int {
	return CountPairCrossingsWithPos(g, left, right, PosMap(adjOrder), useParents)
}

// CountPairCrossingsWithPos is like CountPairCrossings but takes a precomputed position map.
func CountPairCrossingsWithPos(g *DAG, left, right string, adjPos map[string]int, useParents bool) int {
	var lnbr, rnbr []string
	if useParents {
		lnbr = g.Parents(left)
		rnbr = g.Parents(right)
	} else {
		lnbr = g.Children(left)
		rnbr = g.Children(right)
	}

	crossings := 0
	for _, ln := range lnbr {
		lp, ok := adjPos[ln]
		if !ok {
			continue
		}
		for _, rn := range rnbr {
			if rp, ok := adjPos[rn]; ok && lp > rp {
				crossings++
			}
		}
	}
	return crossings
}
