package io

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/matzehuels/stacktower/pkg/render/tower/feature"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
)

// ReadLayout deserializes a layout from JSON.
// It returns the reconstructed Layout, metadata about render options, and any error.
//
// The returned LayoutMeta contains options that were stored with the layout:
//   - Style: the render style used (e.g., "handdrawn", "simple")
//   - Seed: randomization seed for reproducible rendering
//   - Randomize: whether block widths were randomized
//   - Merged: whether subdividers were merged
//   - Nebraska: maintainer ranking data
//   - Edges: dependency edges (for edge rendering)
//
// Example:
//
//	reader, _ := storage.Retrieve(ctx, "job-123/layout.json")
//	l, meta, err := ReadLayout(reader)
//	if err != nil {
//	    return err
//	}
//	// Use meta to configure rendering
//	opts := []sink.SVGOption{}
//	if meta.Merged {
//	    opts = append(opts, sink.WithMerged())
//	}
//	svg := sink.RenderSVG(l, opts...)
func ReadLayout(r io.Reader) (layout.Layout, LayoutMeta, error) {
	var data LayoutData
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return layout.Layout{}, LayoutMeta{}, fmt.Errorf("decode layout: %w", err)
	}

	l := layout.Layout{
		FrameWidth:  data.Width,
		FrameHeight: data.Height,
		MarginX:     data.MarginX,
		MarginY:     data.MarginY,
		RowOrders:   data.Rows,
		Blocks:      make(map[string]layout.Block, len(data.Blocks)),
	}

	for _, b := range data.Blocks {
		l.Blocks[b.ID] = layout.Block{
			NodeID: b.Label,
			Left:   b.X,
			Right:  b.X + b.Width,
			Bottom: b.Y,
			Top:    b.Y + b.Height,
		}
	}

	vizType := data.VizType
	if vizType == "" {
		vizType = VizType // Default to "tower"
	}

	meta := LayoutMeta{
		VizType:   vizType,
		Style:     data.Style,
		Seed:      data.Seed,
		Randomize: data.Randomize,
		Merged:    data.Merged,
		Edges:     data.Edges,
		Nebraska:  convertNebraska(data.Nebraska),
	}

	return l, meta, nil
}

// ReadLayoutFrom reads a layout from a file path.
func ReadLayoutFrom(path string) (layout.Layout, LayoutMeta, error) {
	f, err := os.Open(path)
	if err != nil {
		return layout.Layout{}, LayoutMeta{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return ReadLayout(f)
}

// BlocksData returns the raw block data from a layout JSON, preserving all metadata.
// This is useful when you need access to block metadata (URLs, descriptions, etc.)
// without the full DAG.
func ReadBlocksData(r io.Reader) ([]BlockData, error) {
	var data LayoutData
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode layout: %w", err)
	}
	return data.Blocks, nil
}

func convertNebraska(data []NebraskaData) []feature.NebraskaRanking {
	if len(data) == 0 {
		return nil
	}
	result := make([]feature.NebraskaRanking, len(data))
	for i, d := range data {
		pkgs := make([]feature.PackageRole, len(d.Packages))
		for j, p := range d.Packages {
			pkgs[j] = feature.PackageRole{
				Package: p.Package,
				Role:    feature.Role(p.Role),
				URL:     p.URL,
			}
		}
		result[i] = feature.NebraskaRanking{
			Maintainer: d.Maintainer,
			Score:      d.Score,
			Packages:   pkgs,
		}
	}
	return result
}

// GetBlockMeta returns a map of block ID to BlockData for easy lookup.
// This is useful for enriching renderers with metadata from a stored layout.
func (l *LayoutData) GetBlockMeta() map[string]BlockData {
	m := make(map[string]BlockData, len(l.Blocks))
	for _, b := range l.Blocks {
		m[b.ID] = b
	}
	return m
}
