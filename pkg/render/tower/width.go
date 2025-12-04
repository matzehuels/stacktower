package tower

import (
	"math"

	"github.com/matzehuels/stacktower/pkg/dag"
)

const eps = 1e-9

func ComputeWidths(g *dag.DAG, orders map[int][]string, frameWidth float64) map[string]float64 {
	rows := g.RowIDs()
	if len(rows) == 0 {
		return nil
	}

	widths := make(map[string]float64, g.NodeCount())

	if topRow := orders[0]; len(topRow) > 0 {
		unit := frameWidth / float64(len(topRow))
		for _, id := range topRow {
			widths[id] = unit
		}
	}

	maxRow := rows[len(rows)-1]
	for r := 0; r < maxRow; r++ {
		currRow := orders[r+1]
		if len(currRow) == 0 {
			continue
		}

		for _, id := range currRow {
			widths[id] = 0.0
		}

		for _, parent := range orders[r] {
			kids := g.ChildrenInRow(parent, r+1)
			if n := len(kids); n > 0 {
				share := widths[parent] / float64(n)
				for _, kid := range kids {
					widths[kid] += share
				}
			}
		}

		var sum float64
		for _, id := range currRow {
			sum += widths[id]
		}

		if sum > eps && math.Abs(sum-frameWidth) > eps {
			scale := frameWidth / sum
			for _, id := range currRow {
				widths[id] *= scale
			}
		}
	}
	return widths
}

func ComputeWidthsBottomUp(g *dag.DAG, orders map[int][]string, frameWidth float64) map[string]float64 {
	rows := g.RowIDs()
	if len(rows) == 0 {
		return nil
	}

	widths := make(map[string]float64, g.NodeCount())
	maxRow := rows[len(rows)-1]

	// Start from bottom: sinks get equal width
	if bottomRow := orders[maxRow]; len(bottomRow) > 0 {
		unit := frameWidth / float64(len(bottomRow))
		for _, id := range bottomRow {
			widths[id] = unit
		}
	}

	// Propagate upward: parent width = sum of children's contributions
	for r := maxRow - 1; r >= 0; r-- {
		currRow := orders[r]
		if len(currRow) == 0 {
			continue
		}

		// Each parent gets width from its children
		for _, id := range currRow {
			widths[id] = 0.0
		}

		for _, parent := range currRow {
			kids := g.ChildrenInRow(parent, r+1)
			if len(kids) == 0 {
				continue
			}
			// Parent gets the sum of its share from each child
			// Each child divides its width among its parents
			for _, kid := range kids {
				parents := g.ParentsInRow(kid, r)
				if len(parents) > 0 {
					widths[parent] += widths[kid] / float64(len(parents))
				}
			}
		}

		// Normalize row to fill frame width
		var sum float64
		for _, id := range currRow {
			sum += widths[id]
		}

		if sum > eps && math.Abs(sum-frameWidth) > eps {
			scale := frameWidth / sum
			for _, id := range currRow {
				widths[id] *= scale
			}
		}
	}

	return widths
}
