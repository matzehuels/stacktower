package ordering

import (
	"cmp"
	"slices"

	"github.com/matzehuels/stacktower/pkg/dag"
)

const defaultPasses = 24

type Barycentric struct {
	Passes int
}

func (b Barycentric) OrderRows(g *dag.DAG) map[int][]string {
	rows := g.RowIDs()
	if len(rows) == 0 {
		return nil
	}

	passes := b.Passes
	if passes <= 0 {
		passes = defaultPasses
	}

	rowNodes := make(map[int][]*dag.Node, len(rows))
	for _, r := range rows {
		rowNodes[r] = g.NodesInRow(r)
	}

	best := initOrders(g, rows, rowNodes)
	bestScore := dag.CountCrossings(g, best)
	if bestScore == 0 {
		return best
	}

	if orders, score := runPasses(g, rows, rowNodes, best, passes); score < bestScore {
		best, bestScore = orders, score
		if bestScore == 0 {
			return best
		}
	}

	if orders, score := runPasses(g, rows, rowNodes, reverseOrders(best, rows), passes); score < bestScore {
		return orders
	}
	return best
}

func runPasses(g *dag.DAG, rows []int, rowNodes map[int][]*dag.Node, init map[int][]string, passes int) (map[int][]string, int) {
	orders := copyOrders(init)
	best := copyOrders(orders)
	bestScore := dag.CountCrossings(g, orders)

	staleCount := 0
	for pass := 0; pass < passes && bestScore > 0; pass++ {
		prevScore := bestScore

		if pass%2 == 0 {
			for i := 1; i < len(rows); i++ {
				r := rows[i]
				orders[r] = wmedian(g, rowNodes[r], orders[r], orders[r-1], true)
				transpose(g, orders, r, r-1, true)
			}
		} else {
			for i := len(rows) - 2; i >= 0; i-- {
				r := rows[i]
				orders[r] = wmedian(g, rowNodes[r], orders[r], orders[r+1], false)
				transpose(g, orders, r, r+1, false)
			}
		}

		score := dag.CountCrossings(g, orders)
		if score < bestScore {
			best = copyOrders(orders)
			bestScore = score
			staleCount = 0
		} else {
			staleCount++
		}

		if staleCount >= 4 && score == prevScore {
			break
		}
	}
	return best, bestScore
}

type nodeEntry struct {
	id         string
	median     int
	hasMedian  bool
	currentPos int
}

func (e nodeEntry) sortKey() int {
	if e.hasMedian {
		return e.median
	}
	return e.currentPos
}

func wmedian(g *dag.DAG, nodes []*dag.Node, current, fixed []string, useParents bool) []string {
	if len(nodes) <= 1 {
		return dag.NodeIDs(nodes)
	}

	fixedPos := dag.PosMap(fixed)
	currentPos := dag.PosMap(current)
	entries := make([]nodeEntry, len(nodes))

	for i, n := range nodes {
		var neighbors []string
		if useParents {
			neighbors = g.Parents(n.ID)
		} else {
			neighbors = g.Children(n.ID)
		}

		pos := len(current)
		if p, ok := currentPos[n.ID]; ok {
			pos = p
		}

		medianPos, hasMedian := weightedMedian(neighbors, fixedPos)
		entries[i] = nodeEntry{n.ID, medianPos, hasMedian, pos}
	}

	slices.SortStableFunc(entries, func(a, b nodeEntry) int {
		if c := cmp.Compare(a.sortKey(), b.sortKey()); c != 0 {
			return c
		}
		if a.hasMedian && !b.hasMedian {
			return -1
		}
		if !a.hasMedian && b.hasMedian {
			return 1
		}
		return cmp.Compare(a.currentPos, b.currentPos)
	})

	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.id
	}
	return ids
}

func weightedMedian(neighbors []string, positions map[string]int) (int, bool) {
	var pos []int
	for _, n := range neighbors {
		if p, ok := positions[n]; ok {
			pos = append(pos, p)
		}
	}
	return medianPosition(pos)
}

func transpose(g *dag.DAG, orders map[int][]string, row, adjRow int, useParents bool) {
	order := orders[row]
	if len(order) < 2 {
		return
	}

	adjPos := dag.PosMap(orders[adjRow])
	for {
		swapped := false
		for i := 0; i < len(order)-1; i++ {
			left, right := order[i], order[i+1]

			if leftNode, leftOK := g.Node(left); leftOK {
				if rightNode, rightOK := g.Node(right); rightOK && leftNode.EffectiveID() == rightNode.EffectiveID() {
					continue
				}
			}

			if dag.CountPairCrossingsWithPos(g, right, left, adjPos, useParents) <
				dag.CountPairCrossingsWithPos(g, left, right, adjPos, useParents) {
				order[i], order[i+1] = right, left
				swapped = true
			}
		}
		if !swapped {
			break
		}
	}
}

func reverseOrders(orders map[int][]string, rows []int) map[int][]string {
	rev := make(map[int][]string, len(orders))
	for _, r := range rows {
		reversed := slices.Clone(orders[r])
		slices.Reverse(reversed)
		rev[r] = reversed
	}
	return rev
}

func initOrders(g *dag.DAG, rows []int, rowNodes map[int][]*dag.Node) map[int][]string {
	if len(rows) == 0 {
		return make(map[int][]string)
	}

	orders := make(map[int][]string, len(rows))
	orders[rows[0]] = dag.NodeIDs(rowNodes[rows[0]])
	slices.Sort(orders[rows[0]])

	for i := 1; i < len(rows); i++ {
		r := rows[i]
		if nodes := rowNodes[r]; len(nodes) > 0 {
			orders[r] = orderByMinParent(g, nodes, orders[r-1])
		}
	}
	return orders
}

func orderByMinParent(g *dag.DAG, nodes []*dag.Node, parentOrder []string) []string {
	parentPos := dag.PosMap(parentOrder)

	type entry struct {
		id     string
		minPos int
		avgPos float64
	}
	entries := make([]entry, len(nodes))
	for i, n := range nodes {
		parents := g.Parents(n.ID)
		minPos := len(parentOrder)
		sumPos := 0
		count := 0
		for _, parent := range parents {
			if pos, ok := parentPos[parent]; ok {
				if pos < minPos {
					minPos = pos
				}
				sumPos += pos
				count++
			}
		}
		avgPos := float64(minPos)
		if count > 0 {
			avgPos = float64(sumPos) / float64(count)
		}
		entries[i] = entry{n.ID, minPos, avgPos}
	}

	slices.SortStableFunc(entries, func(a, b entry) int {
		if c := cmp.Compare(a.minPos, b.minPos); c != 0 {
			return c
		}
		if c := cmp.Compare(a.avgPos, b.avgPos); c != 0 {
			return c
		}
		return cmp.Compare(a.id, b.id)
	})

	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.id
	}
	return ids
}

func copyOrders(orders map[int][]string) map[int][]string {
	c := make(map[int][]string, len(orders))
	for k, v := range orders {
		c[k] = slices.Clone(v)
	}
	return c
}
