package pipeline

import (
	"bytes"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	dagtransform "github.com/matzehuels/stacktower/pkg/core/dag/transform"
	nodelinkio "github.com/matzehuels/stacktower/pkg/core/render/nodelink/io"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/feature"
	towerio "github.com/matzehuels/stacktower/pkg/core/render/tower/io"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/layout"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/transform"
)

// LayoutResult contains the outputs of layout computation.
type LayoutResult struct {
	Layout     layout.Layout
	LayoutData []byte
	// Note: Nebraska rankings are embedded in LayoutData JSON, not stored separately
}

// ComputeLayout computes visual positions for a dependency graph.
// If opts.Normalize is true, normalizes the graph before computing layout.
// For tower visualizations, normalization is auto-applied if the graph hasn't
// been layered yet (all nodes at row 0), since tower layout requires row assignments.
// Returns the layout result containing positions, serialized data, and Nebraska rankings.
func ComputeLayout(g *dag.DAG, opts Options) (*LayoutResult, error) {
	// For tower viz, auto-normalize if graph isn't already layered.
	// This is essential because tower layout requires row assignments.
	// Check if graph needs normalization: if max row is 0 and there are edges,
	// the graph hasn't been layered yet.
	needsNormalization := opts.Normalize
	if opts.IsTower() && !opts.Normalize && g.EdgeCount() > 0 && g.MaxRow() == 0 {
		needsNormalization = true
	}

	if needsNormalization {
		dagtransform.Normalize(g)
	}

	if opts.IsNodelink() {
		return computeNodelinkLayout(g, opts)
	}
	return computeTowerLayout(g, opts)
}

// computeTowerLayout computes a tower-style layout.
func computeTowerLayout(g *dag.DAG, opts Options) (*LayoutResult, error) {
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

	// Serialize layout with Nebraska rankings embedded
	// Rankings are computed here while the graph has row assignments
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

	// Always compute and embed Nebraska rankings in layout JSON
	rankings := feature.RankNebraska(g, 10)
	if len(rankings) > 0 {
		writeOpts = append(writeOpts, towerio.WithNebraska(rankings))
	}

	layoutData, err := towerio.WriteLayout(l, writeOpts...)
	if err != nil {
		return nil, err
	}

	return &LayoutResult{
		Layout:     l,
		LayoutData: layoutData,
	}, nil
}

// computeNodelinkLayout computes a node-link (Graphviz) layout.
func computeNodelinkLayout(g *dag.DAG, opts Options) (*LayoutResult, error) {
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
		return nil, err
	}

	// Return empty layout since nodelink uses DOT string from JSON
	// Nebraska rankings are not applicable for nodelink visualizations
	return &LayoutResult{
		Layout:     layout.Layout{},
		LayoutData: layoutData,
	}, nil
}

// LayoutFromData deserializes a layout from JSON data.
// Returns layout, metadata (including Nebraska rankings), and any error.
func LayoutFromData(data []byte) (layout.Layout, towerio.LayoutMeta, error) {
	return towerio.ReadLayout(bytes.NewReader(data))
}
