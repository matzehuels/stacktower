package transform

import (
	"stacktower/pkg/dag"
	"stacktower/pkg/render/tower"
)

func MergeSubdividers(layout tower.Layout, g *dag.DAG) tower.Layout {
	groups := buildMasterGroups(g)
	blocks := make(map[string]tower.Block, len(groups))
	for master, members := range groups {
		blocks[master] = mergeBlocks(layout, master, members)
	}

	return tower.Layout{
		FrameWidth:  layout.FrameWidth,
		FrameHeight: layout.FrameHeight,
		Blocks:      blocks,
		RowOrders:   filterRowOrders(layout.RowOrders, g),
		MarginX:     layout.MarginX,
		MarginY:     layout.MarginY,
	}
}

func buildMasterGroups(g *dag.DAG) map[string][]string {
	groups := make(map[string][]string)
	for _, n := range g.Nodes() {
		master := n.EffectiveID()
		groups[master] = append(groups[master], n.ID)
	}
	return groups
}

func mergeBlocks(layout tower.Layout, master string, members []string) tower.Block {
	var blocks []tower.Block
	for _, id := range members {
		if b, ok := layout.Blocks[id]; ok {
			blocks = append(blocks, b)
		}
	}

	if len(blocks) == 0 {
		return tower.Block{NodeID: master}
	}

	result := blocks[0]
	for _, b := range blocks[1:] {
		result.Bottom = min(result.Bottom, b.Bottom)
		result.Top = max(result.Top, b.Top)
		result.Left = min(result.Left, b.Left)
		result.Right = max(result.Right, b.Right)
	}
	result.NodeID = master
	return result
}

func filterRowOrders(orders map[int][]string, g *dag.DAG) map[int][]string {
	result := make(map[int][]string, len(orders))
	for row, ids := range orders {
		var filtered []string
		for _, id := range ids {
			n, ok := g.Node(id)
			if !ok || n.IsSubdivider() {
				continue
			}
			filtered = append(filtered, id)
		}
		if len(filtered) > 0 {
			result[row] = filtered
		}
	}
	return result
}
