package transform

import (
	"maps"
	"math/rand/v2"
	"slices"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/layout"
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
	ensureMinimumOverlap(g, blocks, l.RowOrders, opts.MinOverlap)

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

func ensureMinimumOverlap(g *dag.DAG, blocks map[string]layout.Block, rowOrders map[int][]string, minOverlap float64) {
	edges := g.Edges()

	// Build a map of block ID to row for collision checking
	blockRow := make(map[string]int)
	for row, ids := range rowOrders {
		for _, id := range ids {
			blockRow[id] = row
		}
	}

	for range 10 {
		changed := false
		for _, edge := range edges {
			parent, okP := blocks[edge.From]
			child, okC := blocks[edge.To]
			if !okP || !okC {
				continue
			}

			currentOverlap := calcOverlap(parent.Left, parent.Right, child.Left, child.Right)
			if currentOverlap >= minOverlap {
				continue
			}

			// Calculate proposed expansions for both parent and child
			newParent, newChild := parent, child
			parentCenter := (parent.Left + parent.Right) / 2
			childCenter := (child.Left + child.Right) / 2

			if parentCenter < childCenter {
				// Parent is left of child: parent expands right, child expands left
				newParent.Right = max(parent.Right, child.Left+minOverlap)
				newChild.Left = min(child.Left, parent.Right-minOverlap)
			} else {
				// Parent is right of child: parent expands left, child expands right
				newParent.Left = min(parent.Left, child.Right-minOverlap)
				newChild.Right = max(child.Right, parent.Left+minOverlap)
			}

			// Check collisions independently and apply what we can
			parentCollides := wouldCollide(edge.From, newParent, blockRow, rowOrders, blocks)
			childCollides := wouldCollide(edge.To, newChild, blockRow, rowOrders, blocks)

			if !parentCollides && !childCollides {
				// Both can expand
				blocks[edge.From] = newParent
				blocks[edge.To] = newChild
				changed = true
			} else if !parentCollides {
				// Only parent can expand - make parent cover the child
				if parentCenter < childCenter {
					newParent.Right = child.Right + minOverlap
				} else {
					newParent.Left = child.Left - minOverlap
				}
				if !wouldCollide(edge.From, newParent, blockRow, rowOrders, blocks) {
					blocks[edge.From] = newParent
					changed = true
				}
			} else if !childCollides {
				// Only child can expand - make child reach the parent
				if parentCenter < childCenter {
					newChild.Left = parent.Left - minOverlap
				} else {
					newChild.Right = parent.Right + minOverlap
				}
				if !wouldCollide(edge.To, newChild, blockRow, rowOrders, blocks) {
					blocks[edge.To] = newChild
					changed = true
				}
			}
			// If both collide, skip this edge
		}
		if !changed {
			break
		}
	}
}

// wouldCollide checks if expanding a block to newBounds would collide with
// other blocks in the same row or pillar blocks that span through this row.
func wouldCollide(id string, newBounds layout.Block, blockRow map[string]int, rowOrders map[int][]string, blocks map[string]layout.Block) bool {
	row, ok := blockRow[id]
	if !ok {
		return false
	}

	// Check against neighbors in the same row
	for _, neighborID := range rowOrders[row] {
		if neighborID == id {
			continue
		}
		neighbor, ok := blocks[neighborID]
		if !ok {
			continue
		}
		// Check for overlap (with small tolerance for floating point)
		if newBounds.Right > neighbor.Left+1 && newBounds.Left < neighbor.Right-1 {
			return true
		}
	}

	// Check against pillar blocks (merged subdividers) that span through this row.
	// These blocks are in different rows in rowOrders but visually overlap this row.
	for blockID, block := range blocks {
		if blockID == id {
			continue
		}
		// Skip blocks that are in rowOrders for this row (already checked above)
		if blockRow[blockID] == row {
			continue
		}
		// Check if this block vertically spans through newBounds' row
		if block.Top > newBounds.Bottom && block.Bottom < newBounds.Top {
			// Horizontal overlap check
			if newBounds.Right > block.Left+1 && newBounds.Left < block.Right-1 {
				return true
			}
		}
	}
	return false
}

func calcOverlap(a1, a2, b1, b2 float64) float64 {
	return max(0, min(a2, b2)-max(a1, b1))
}
