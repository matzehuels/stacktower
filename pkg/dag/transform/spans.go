package transform

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/matzehuels/stacktower/pkg/dag"
)

func ResolveSpanOverlaps(d *dag.DAG) {
	usedIDs := nodeIDSet(d.Nodes())
	for _, row := range d.RowIDs() {
		if row > 0 {
			for insertSeparatorAt(d, row, usedIDs) {
			}
		}
	}
}

func insertSeparatorAt(d *dag.DAG, row int, usedIDs map[string]struct{}) bool {
	children := d.NodesInRow(row)
	if len(children) < 2 {
		return false
	}

	for _, child := range children {
		if child.IsSubdivider() {
			return false
		}
	}

	sorted := slices.Clone(children)
	slices.SortFunc(sorted, func(a, b *dag.Node) int { return cmp.Compare(a.ID, b.ID) })

	if ranges := findOverlappingSpans(d, sorted); len(ranges) > 0 {
		shiftRowsDown(d, row)
		for _, r := range ranges {
			insertSeparator(d, row, sorted, r, usedIDs)
		}
		return true
	}
	return false
}

type span struct{ lo, hi int }

func findOverlappingSpans(d *dag.DAG, children []*dag.Node) []span {
	if len(children) < 2 {
		return nil
	}

	childPos := dag.NodePosMap(children)
	overlapCounts := make([]int, len(children)-1)
	targetRow := children[0].Row

	for _, parent := range d.NodesInRow(targetRow - 1) {
		if !eligibleForSeparation(d, parent, targetRow) {
			continue
		}

		if indices := childPositions(d.Children(parent.ID), childPos); len(indices) >= 2 {
			minIdx, maxIdx := slices.Min(indices), slices.Max(indices)
			for i := minIdx; i < maxIdx; i++ {
				if canInsertBetween(children, i) {
					overlapCounts[i]++
				}
			}
		}
	}

	return collectRanges(overlapCounts)
}

func eligibleForSeparation(d *dag.DAG, parent *dag.Node, targetRow int) bool {
	children := d.ChildrenInRow(parent.ID, targetRow)
	if len(children) < 2 || len(children) != len(d.Children(parent.ID)) {
		return false
	}
	for _, childID := range children {
		if n, ok := d.Node(childID); ok && n.IsSubdivider() {
			return false
		}
	}
	return true
}

func childPositions(childIDs []string, posMap map[string]int) []int {
	var indices []int
	for _, id := range childIDs {
		if pos, ok := posMap[id]; ok {
			indices = append(indices, pos)
		}
	}
	return indices
}

func canInsertBetween(children []*dag.Node, i int) bool {
	if i < 0 || i+1 >= len(children) {
		return true
	}
	left, right := children[i], children[i+1]
	if !left.IsSubdivider() || !right.IsSubdivider() {
		return true
	}
	return left.MasterID == "" || left.MasterID != right.MasterID
}

func collectRanges(overlapCounts []int) []span {
	var ranges []span
	for i := 0; i < len(overlapCounts); i++ {
		if overlapCounts[i] >= 2 {
			start := i
			for i < len(overlapCounts) && overlapCounts[i] >= 2 {
				i++
			}
			ranges = append(ranges, span{start, i})
			i--
		}
	}
	return ranges
}

func shiftRowsDown(d *dag.DAG, fromRow int) {
	nodes := d.Nodes()
	newRows := make(map[string]int, len(nodes))
	for _, n := range nodes {
		row := n.Row
		if row >= fromRow {
			row++
		}
		newRows[n.ID] = row
	}
	d.SetRows(newRows)
}

func insertSeparator(d *dag.DAG, row int, children []*dag.Node, r span, usedIDs map[string]struct{}) {
	separatorID := uniqueID(row, children[r.lo].ID, children[r.hi].ID, usedIDs)
	if err := d.AddNode(dag.Node{
		ID:   separatorID,
		Row:  row,
		Kind: dag.NodeKindAuxiliary,
	}); err != nil {
		panic(err)
	}

	affectedChildren := make(map[string]struct{}, r.hi-r.lo+1)
	for i := r.lo; i <= r.hi; i++ {
		affectedChildren[children[i].ID] = struct{}{}
	}

	parents := make(map[string]struct{})
	for _, e := range d.Edges() {
		if src, ok := d.Node(e.From); ok && src.Row == row-1 {
			if _, affected := affectedChildren[e.To]; affected {
				parents[e.From] = struct{}{}
				d.RemoveEdge(e.From, e.To)
			}
		}
	}

	for parent := range parents {
		if err := d.AddEdge(dag.Edge{From: parent, To: separatorID}); err != nil {
			panic(err)
		}
	}

	for child := range affectedChildren {
		if err := d.AddEdge(dag.Edge{From: separatorID, To: child}); err != nil {
			panic(err)
		}
	}
}

func uniqueID(row int, firstChild, lastChild string, usedIDs map[string]struct{}) string {
	firstClean := strings.ReplaceAll(firstChild, "_", "")
	lastClean := strings.ReplaceAll(lastChild, "_", "")

	id := fmt.Sprintf("Sep_%d_%s_%s", row, firstClean, lastClean)
	if _, exists := usedIDs[id]; !exists {
		usedIDs[id] = struct{}{}
		return id
	}

	for i := 1; ; i++ {
		id = fmt.Sprintf("Sep_%d_%s_%s__%d", row, firstClean, lastClean, i)
		if _, exists := usedIDs[id]; !exists {
			usedIDs[id] = struct{}{}
			return id
		}
	}
}

func nodeIDSet(nodes []*dag.Node) map[string]struct{} {
	m := make(map[string]struct{}, len(nodes))
	for _, n := range nodes {
		m[n.ID] = struct{}{}
	}
	return m
}
