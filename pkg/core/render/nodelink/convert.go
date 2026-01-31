package nodelink

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/feature"
	"github.com/matzehuels/stacktower/pkg/graph"
)

// Export creates a serializable nodelink layout from a DOT string.
//
// Unlike tower layouts, nodelink layouts don't compute positions internally—Graphviz
// does that during rendering. This function packages the DOT string and graph metadata
// into the unified serialization format.
//
// Use this when you need to:
//   - Export nodelink layout to JSON file
//   - Cache the layout for later rendering
//   - Return layout data from an API
func Export(dot string, g *dag.DAG, opts Options, width, height float64, style string) (graph.Layout, error) {
	result := graph.Layout{
		VizType: graph.VizTypeNodelink,
		DOT:     dot,
		Width:   width,
		Height:  height,
		Engine:  "dot",
		Style:   style,
	}

	if g != nil {
		// Extract structured nodes using the unified conversion
		serialized := graph.FromDAG(g)
		// Enrich nodes with computed brittle flag
		for i := range serialized.Nodes {
			if n, ok := g.Node(serialized.Nodes[i].ID); ok {
				serialized.Nodes[i].Brittle = feature.IsBrittle(n)
			}
		}
		result.Nodes = serialized.Nodes
		result.Edges = serialized.Edges

		// Build row assignments
		rows := make(map[int][]string)
		for _, n := range g.Nodes() {
			rows[n.Row] = append(rows[n.Row], n.ID)
		}
		result.Rows = rows
	}

	return result, nil
}

// Parse extracts the DOT string from a serialized nodelink layout.
//
// Returns an error if the layout is not a nodelink type or is missing the DOT string.
func Parse(layout graph.Layout) (string, error) {
	if layout.VizType != "" && layout.VizType != graph.VizTypeNodelink {
		return "", fmt.Errorf("invalid viz_type for nodelink layout: %q", layout.VizType)
	}

	if layout.DOT == "" {
		return "", fmt.Errorf("nodelink layout must contain DOT string")
	}

	return layout.DOT, nil
}
