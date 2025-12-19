package transform

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
)

// MergeSubdividers combines subdivider blocks into continuous vertical columns.
// Subdivider nodes (created by [dag/transform.Subdivide] to break long edges)
// are grouped by their MasterID and horizontal position, then merged into
// single blocks spanning from top to bottom.
//
// This creates cleaner visuals where a package's vertical "column" is rendered
// as one continuous block rather than separate segments per row.
//
// The returned layout has subdivider nodes removed from RowOrders and replaced
// with merged blocks keyed by their master ID.
func MergeSubdividers(l layout.Layout, g *dag.DAG) layout.Layout {
	blocks := make(map[string]layout.Block)

	for master, members := range groupByMaster(g) {
		subgroups := groupByPosition(l, members)
		for _, group := range subgroups {
			b := merge(group, master)
			key := master
			if len(subgroups) > 1 {
				key = fmt.Sprintf("%s@%.0f", master, b.Left)
			}
			blocks[key] = b
		}
	}

	return layout.Layout{
		FrameWidth:  l.FrameWidth,
		FrameHeight: l.FrameHeight,
		Blocks:      blocks,
		RowOrders:   filterSubdividers(l.RowOrders, g),
		MarginX:     l.MarginX,
		MarginY:     l.MarginY,
	}
}

func groupByMaster(g *dag.DAG) map[string][]string {
	groups := make(map[string][]string)
	for _, n := range g.Nodes() {
		groups[n.EffectiveID()] = append(groups[n.EffectiveID()], n.ID)
	}
	return groups
}

func groupByPosition(l layout.Layout, members []string) [][]layout.Block {
	type pos struct{ l, r int }
	groups := make(map[pos][]layout.Block)

	for _, id := range members {
		if b, ok := l.Blocks[id]; ok {
			key := pos{int(b.Left + 0.5), int(b.Right + 0.5)}
			groups[key] = append(groups[key], b)
		}
	}

	result := make([][]layout.Block, 0, len(groups))
	for _, g := range groups {
		result = append(result, g)
	}
	return result
}

func merge(blocks []layout.Block, master string) layout.Block {
	if len(blocks) == 0 {
		return layout.Block{NodeID: master}
	}
	result := blocks[0]
	for _, b := range blocks[1:] {
		result.Bottom = min(result.Bottom, b.Bottom)
		result.Top = max(result.Top, b.Top)
	}
	result.NodeID = master
	return result
}

func filterSubdividers(orders map[int][]string, g *dag.DAG) map[int][]string {
	result := make(map[int][]string, len(orders))
	for row, ids := range orders {
		var filtered []string
		for _, id := range ids {
			if n, ok := g.Node(id); ok && !n.IsSubdivider() {
				filtered = append(filtered, id)
			}
		}
		if len(filtered) > 0 {
			result[row] = filtered
		}
	}
	return result
}
