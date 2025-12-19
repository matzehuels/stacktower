package transform

import (
	"maps"
	"math/rand/v2"
	"slices"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
)

// Options configures the randomization behavior for [Randomize].
type Options struct {
	// WidthShrink is the maximum shrink factor applied to blocks (0-1).
	// Higher values create more width variation. Default: 0.85.
	WidthShrink float64

	// MinBlockWidth is the minimum allowed block width in pixels.
	// Blocks will not shrink below this size. Default: 30.
	MinBlockWidth float64

	// MinGap is the minimum gap between adjacent blocks in pixels. Default: 5.
	MinGap float64

	// MinOverlap is the minimum horizontal overlap required between connected
	// blocks (parent-child pairs). Blocks are expanded if needed. Default: 10.
	MinOverlap float64
}

var defaultOpts = Options{
	WidthShrink:   0.85,
	MinBlockWidth: 30.0,
	MinGap:        5.0,
	MinOverlap:    10.0,
}

// Randomize applies controlled random variation to block widths.
// It creates a checkerboard pattern by shrinking alternating rows, which
// mimics hand-drawn diagrams and adds visual interest.
//
// The seed ensures reproducible randomness—the same seed produces identical
// layouts. Pass nil for opts to use defaults.
//
// After shrinking, the function ensures connected blocks maintain minimum
// overlap so dependency edges remain visually clear.
func Randomize(l layout.Layout, g *dag.DAG, seed uint64, opts *Options) layout.Layout {
	if opts == nil {
		opts = &defaultOpts
	}
	if shrink := max(0.0, min(opts.WidthShrink, 1.0)); shrink == 0 {
		return l
	}

	blocks := maps.Clone(l.Blocks)
	rows := sortedRows(l.RowOrders)
	rng := rand.New(rand.NewPCG(seed, seed^0xdeadbeef))

	shrinkCheckerboard(l.RowOrders, blocks, rows, rng, opts)
	ensureMinimumOverlap(g, blocks, opts.MinOverlap)

	l.Blocks = blocks
	return l
}

func shrinkCheckerboard(orders map[int][]string, blocks map[string]layout.Block, rows []int, rng *rand.Rand, opts *Options) {
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

func ensureMinimumOverlap(g *dag.DAG, blocks map[string]layout.Block, minOverlap float64) {
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
