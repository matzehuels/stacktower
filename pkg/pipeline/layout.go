package pipeline

import (
	"bytes"

	"github.com/matzehuels/stacktower/pkg/dag"
	dagtransform "github.com/matzehuels/stacktower/pkg/dag/transform"
	"github.com/matzehuels/stacktower/pkg/render/nodelink"
	towerio "github.com/matzehuels/stacktower/pkg/render/tower/io"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
	"github.com/matzehuels/stacktower/pkg/render/tower/ordering"
	"github.com/matzehuels/stacktower/pkg/render/tower/transform"
)

// LayoutOptions contains options for layout computation.
type LayoutOptions struct {
	VizType   string
	Width     float64
	Height    float64
	Ordering  string
	Merge     bool
	Randomize bool
	Seed      uint64
	Orderer   ordering.Orderer
}

// ComputeLayout computes visual positions for a dependency graph.
// If opts.Normalize is true, normalizes the graph before computing layout.
// Returns the layout, serialized layout data, and any error.
func ComputeLayout(g *dag.DAG, opts Options) (layout.Layout, []byte, error) {
	// Normalize graph if requested
	if opts.Normalize {
		dagtransform.Normalize(g)
	}

	if opts.IsNodelink() {
		return computeNodelinkLayout(g)
	}
	return computeTowerLayout(g, opts)
}

// computeTowerLayout computes a tower-style layout.
func computeTowerLayout(g *dag.DAG, opts Options) (layout.Layout, []byte, error) {
	// Build layout options
	var layoutOpts []layout.Option
	if opts.Orderer != nil {
		layoutOpts = append(layoutOpts, layout.WithOrderer(opts.Orderer))
	}

	// Compute base layout
	l := layout.Build(g, opts.Width, opts.Height, layoutOpts...)

	// Apply transforms
	if opts.Merge {
		l = transform.MergeSubdividers(l, g)
	}
	if opts.Randomize {
		l = transform.Randomize(l, g, opts.Seed, nil)
	}

	// Serialize layout
	writeOpts := []towerio.WriteOption{towerio.WithGraph(g)}
	if opts.Merge {
		writeOpts = append(writeOpts, towerio.WithMerged())
	}
	if opts.Randomize {
		writeOpts = append(writeOpts, towerio.WithRandomize(opts.Seed))
	}
	if opts.Style != "" {
		writeOpts = append(writeOpts, towerio.WithStyle(opts.Style))
	}

	layoutData, err := towerio.WriteLayout(l, writeOpts...)
	if err != nil {
		return layout.Layout{}, nil, err
	}

	return l, layoutData, nil
}

// computeNodelinkLayout computes a node-link (Graphviz) layout.
func computeNodelinkLayout(g *dag.DAG) (layout.Layout, []byte, error) {
	dot := nodelink.ToDOT(g, nodelink.Options{Detailed: false})
	// Return empty layout since nodelink uses DOT string directly
	return layout.Layout{}, []byte(dot), nil
}

// LayoutFromData deserializes a layout from JSON data.
func LayoutFromData(data []byte) (layout.Layout, towerio.LayoutMeta, error) {
	return towerio.ReadLayout(bytes.NewReader(data))
}
