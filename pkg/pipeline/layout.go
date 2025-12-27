package pipeline

import (
	"bytes"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	dagtransform "github.com/matzehuels/stacktower/pkg/core/dag/transform"
	nodelinkio "github.com/matzehuels/stacktower/pkg/core/render/nodelink/io"
	towerio "github.com/matzehuels/stacktower/pkg/core/render/tower/io"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/layout"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/transform"
)

// ComputeLayout computes visual positions for a dependency graph.
// If opts.Normalize is true, normalizes the graph before computing layout.
// Returns the layout, serialized layout data, and any error.
func ComputeLayout(g *dag.DAG, opts Options) (layout.Layout, []byte, error) {
	// Normalize graph if requested
	if opts.Normalize {
		dagtransform.Normalize(g)
	}

	if opts.IsNodelink() {
		return computeNodelinkLayout(g, opts)
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
func computeNodelinkLayout(g *dag.DAG, opts Options) (layout.Layout, []byte, error) {
	// Build write options
	writeOpts := []nodelinkio.WriteOption{
		nodelinkio.WithGraph(g),
		nodelinkio.WithDimensions(opts.Width, opts.Height),
		nodelinkio.WithEngine("dot"),
	}
	if opts.Style != "" {
		writeOpts = append(writeOpts, nodelinkio.WithStyle(opts.Style))
	}

	// Serialize layout (includes DOT generation)
	layoutData, err := nodelinkio.WriteLayout(writeOpts...)
	if err != nil {
		return layout.Layout{}, nil, err
	}

	// Return empty layout since nodelink uses DOT string from JSON
	return layout.Layout{}, layoutData, nil
}

// LayoutFromData deserializes a layout from JSON data.
func LayoutFromData(data []byte) (layout.Layout, towerio.LayoutMeta, error) {
	return towerio.ReadLayout(bytes.NewReader(data))
}
