package transform

import (
	"maps"
	"math/rand/v2"
	"slices"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower"
)

type Options struct {
	WidthShrink   float64
	MinBlockWidth float64
	MinGap        float64
	MinOverlap    float64
}

var defaultOpts = Options{
	WidthShrink:   0.85,
	MinBlockWidth: 30.0,
	MinGap:        5.0,
	MinOverlap:    10.0,
}

func Randomize(layout tower.Layout, g *dag.DAG, seed uint64, opts *Options) tower.Layout {
	if opts == nil {
		opts = &defaultOpts
	}
	if shrink := max(0.0, min(opts.WidthShrink, 1.0)); shrink == 0 {
		return layout
	}

	blocks := maps.Clone(layout.Blocks)
	rows := sortedRows(layout.RowOrders)
	rng := rand.New(rand.NewPCG(seed, seed^0xdeadbeef))

	shrinkCheckerboard(layout.RowOrders, blocks, rows, rng, opts)
	ensureMinimumOverlap(g, blocks, opts.MinOverlap)

	layout.Blocks = blocks
	return layout
}

func shrinkCheckerboard(orders map[int][]string, blocks map[string]tower.Block, rows []int, rng *rand.Rand, opts *Options) {
	shrink := max(0, min(opts.WidthShrink, 1))
	for rowIdx, row := range rows {
		if rowIdx == 0 {
			continue
		}
		for _, nodeID := range orders[row] {
			node := blocks[nodeID]
			center := (node.Left + node.Right) / 2
			width := node.Right - node.Left - 2*opts.MinGap
			if rowIdx%2 == 1 {
				width *= 1 - rng.Float64()*shrink
			}
			width = max(width, opts.MinBlockWidth)
			node.Left = center - width/2
			node.Right = center + width/2
			blocks[nodeID] = node
		}
	}
}

func sortedRows(orders map[int][]string) []int {
	rows := slices.Collect(maps.Keys(orders))
	slices.Sort(rows)
	return rows
}

func ensureMinimumOverlap(g *dag.DAG, blocks map[string]tower.Block, minOverlap float64) {
	edges := g.Edges()

	for range 10 {
		changed := false
		for _, edge := range edges {
			parent, okP := blocks[edge.From]
			child, okC := blocks[edge.To]
			if !okP || !okC || calcOverlap(parent.Left, parent.Right, child.Left, child.Right) >= minOverlap {
				continue
			}
			changed = true

			if (parent.Left+parent.Right)/2 < (child.Left+child.Right)/2 {
				parent.Right = max(parent.Right, child.Left+minOverlap)
				child.Left = min(child.Left, parent.Right-minOverlap)
			} else {
				parent.Left = min(parent.Left, child.Right-minOverlap)
				child.Right = max(child.Right, parent.Left+minOverlap)
			}
			blocks[edge.From] = parent
			blocks[edge.To] = child
		}
		if !changed {
			break
		}
	}
}

func calcOverlap(a1, a2, b1, b2 float64) float64 {
	return max(0, min(a2, b2)-max(a1, b1))
}
