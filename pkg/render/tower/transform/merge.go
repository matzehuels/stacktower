package transform

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower"
)

func MergeSubdividers(layout tower.Layout, g *dag.DAG) tower.Layout {
	blocks := make(map[string]tower.Block)

	for master, members := range groupByMaster(g) {
		subgroups := groupByPosition(layout, members)
		for _, group := range subgroups {
			b := merge(group, master)
			key := master
			if len(subgroups) > 1 {
				key = fmt.Sprintf("%s@%.0f", master, b.Left)
			}
			blocks[key] = b
		}
	}

	return tower.Layout{
		FrameWidth:  layout.FrameWidth,
		FrameHeight: layout.FrameHeight,
		Blocks:      blocks,
		RowOrders:   filterSubdividers(layout.RowOrders, g),
		MarginX:     layout.MarginX,
		MarginY:     layout.MarginY,
	}
}

func groupByMaster(g *dag.DAG) map[string][]string {
	groups := make(map[string][]string)
	for _, n := range g.Nodes() {
		groups[n.EffectiveID()] = append(groups[n.EffectiveID()], n.ID)
	}
	return groups
}

func groupByPosition(layout tower.Layout, members []string) [][]tower.Block {
	type pos struct{ l, r int }
	groups := make(map[pos][]tower.Block)

	for _, id := range members {
		if b, ok := layout.Blocks[id]; ok {
			key := pos{int(b.Left + 0.5), int(b.Right + 0.5)}
			groups[key] = append(groups[key], b)
		}
	}

	result := make([][]tower.Block, 0, len(groups))
	for _, g := range groups {
		result = append(result, g)
	}
	return result
}

func merge(blocks []tower.Block, master string) tower.Block {
	if len(blocks) == 0 {
		return tower.Block{NodeID: master}
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
