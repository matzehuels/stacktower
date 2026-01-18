package nodelink

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/dto"
)

// ToDTO creates a serializable nodelink layout (dto.Layout) from a DOT string.
//
// Unlike tower layouts, nodelink layouts don't compute positions internally - Graphviz
// does that during rendering. This function packages the DOT string and graph metadata
// into the unified serialization format.
//
// Use this when you need to:
//   - Export nodelink layout to JSON file
//   - Cache the layout for later rendering
//   - Return layout data from an API
func ToDTO(dot string, g *dag.DAG, opts Options, width, height float64, style string) (dto.Layout, error) {
	result := dto.Layout{
		VizType: dto.VizTypeNodelink,
		DOT:     dot,
		Width:   width,
		Height:  height,
		Engine:  "dot",
		Style:   style,
	}

	if g != nil {
		// Extract structured nodes using the unified conversion
		graph := dto.FromDAG(g)
		result.Nodes = graph.Nodes
		result.Edges = graph.Edges

		// Build row assignments
		rows := make(map[int][]string)
		for _, n := range g.Nodes() {
			rows[n.Row] = append(rows[n.Row], n.ID)
		}
		result.Rows = rows
	}

	return result, nil
}

// FromDTO extracts the DOT string from a serialized nodelink layout.
//
// Returns an error if the DTO is not a nodelink layout or is missing the DOT string.
func FromDTO(d dto.Layout) (string, error) {
	if d.VizType != "" && d.VizType != dto.VizTypeNodelink {
		return "", fmt.Errorf("invalid viz_type for nodelink layout: %q", d.VizType)
	}

	if d.DOT == "" {
		return "", fmt.Errorf("nodelink layout must contain DOT string")
	}

	return d.DOT, nil
}
