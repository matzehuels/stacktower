package io

import (
	"encoding/json"
	"io"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/feature"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
)

// WriteOption configures layout export.
type WriteOption func(*writeConfig)

type writeConfig struct {
	graph     *dag.DAG
	merged    bool
	randomize bool
	seed      uint64
	style     string
	nebraska  []feature.NebraskaRanking
}

// WithGraph attaches the DAG for metadata enrichment (URLs, brittle flags,
// auxiliary/synthetic flags, edges). Without this, blocks will have minimal metadata.
func WithGraph(g *dag.DAG) WriteOption { return func(c *writeConfig) { c.graph = g } }

// WithMerged marks that the layout uses merged subdividers.
func WithMerged() WriteOption { return func(c *writeConfig) { c.merged = true } }

// WithRandomize records the randomization seed, enabling reproducible re-rendering.
func WithRandomize(seed uint64) WriteOption {
	return func(c *writeConfig) { c.randomize = true; c.seed = seed }
}

// WithStyle records the style name (e.g., "simple", "handdrawn") for re-rendering.
func WithStyle(s string) WriteOption { return func(c *writeConfig) { c.style = s } }

// WithNebraska includes Nebraska ranking data in the output.
func WithNebraska(rankings []feature.NebraskaRanking) WriteOption {
	return func(c *writeConfig) { c.nebraska = rankings }
}

// WriteLayout serializes a layout to JSON with optional metadata enrichment.
// The output can be read back with ReadLayout to reconstruct the layout.
//
// Example:
//
//	data, err := WriteLayout(l, WithGraph(g), WithStyle("handdrawn"), WithRandomize(42))
//	if err != nil {
//	    return err
//	}
//	storage.Store(ctx, jobID, "layout.json", bytes.NewReader(data))
func WriteLayout(l layout.Layout, opts ...WriteOption) ([]byte, error) {
	cfg := writeConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	data := LayoutData{
		VizType:   VizType,
		Width:     l.FrameWidth,
		Height:    l.FrameHeight,
		MarginX:   l.MarginX,
		MarginY:   l.MarginY,
		Style:     cfg.style,
		Seed:      cfg.seed,
		Randomize: cfg.randomize,
		Merged:    cfg.merged,
		Rows:      l.RowOrders,
		Blocks:    buildBlocks(l, cfg.graph),
	}

	if cfg.graph != nil {
		data.Edges = buildEdges(l, cfg.graph, cfg.merged)
	}

	if len(cfg.nebraska) > 0 {
		data.Nebraska = buildNebraska(cfg.nebraska)
	}

	return json.MarshalIndent(data, "", "  ")
}

// WriteLayoutTo writes a layout to the given writer.
func WriteLayoutTo(w io.Writer, l layout.Layout, opts ...WriteOption) error {
	data, err := WriteLayout(l, opts...)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func buildBlocks(l layout.Layout, g *dag.DAG) []BlockData {
	blocks := make([]BlockData, 0, len(l.Blocks))
	for id, b := range l.Blocks {
		bd := BlockData{
			ID:     id,
			Label:  b.NodeID,
			X:      b.Left,
			Y:      b.Bottom,
			Width:  b.Width(),
			Height: b.Height(),
		}
		if g != nil {
			if n, ok := g.Node(id); ok {
				bd.Auxiliary = n.IsAuxiliary()
				bd.Synthetic = n.IsSynthetic()
				if n.Meta != nil {
					bd.URL, _ = n.Meta["repo_url"].(string)
					bd.Brittle = feature.IsBrittle(n)
					bd.Meta = extractMeta(n)
				}
			}
		}
		blocks = append(blocks, bd)
	}
	return blocks
}

func extractMeta(n *dag.Node) *BlockMeta {
	if n.Meta == nil {
		return nil
	}
	m := &BlockMeta{
		Stars: feature.AsInt(n.Meta["repo_stars"]),
	}
	m.LastCommit, _ = n.Meta["repo_last_commit"].(string)
	m.LastRelease, _ = n.Meta["repo_last_release"].(string)
	m.Archived, _ = n.Meta["repo_archived"].(bool)

	if desc, ok := n.Meta["description"].(string); ok && desc != "" {
		m.Description = desc
	} else if summary, ok := n.Meta["summary"].(string); ok && summary != "" {
		m.Description = summary
	}

	if m.Description == "" && m.Stars == 0 && m.LastCommit == "" && m.LastRelease == "" && !m.Archived {
		return nil
	}
	return m
}

func buildEdges(l layout.Layout, g *dag.DAG, merged bool) []EdgeData {
	if merged {
		return buildMergedEdges(l, g)
	}
	edges := make([]EdgeData, 0)
	for _, e := range g.Edges() {
		if _, ok := l.Blocks[e.From]; !ok {
			continue
		}
		if _, ok := l.Blocks[e.To]; !ok {
			continue
		}
		edges = append(edges, EdgeData{From: e.From, To: e.To})
	}
	return edges
}

func buildMergedEdges(l layout.Layout, g *dag.DAG) []EdgeData {
	masterOf := func(id string) string {
		if n, ok := g.Node(id); ok && n.MasterID != "" {
			return n.MasterID
		}
		return id
	}

	type edgeKey struct{ from, to string }
	seen := make(map[edgeKey]struct{})
	var edges []EdgeData

	for _, e := range g.Edges() {
		fromMaster, toMaster := masterOf(e.From), masterOf(e.To)
		if fromMaster == toMaster {
			continue
		}
		key := edgeKey{fromMaster, toMaster}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		edges = append(edges, EdgeData{From: fromMaster, To: toMaster})
	}
	return edges
}

func buildNebraska(rankings []feature.NebraskaRanking) []NebraskaData {
	result := make([]NebraskaData, len(rankings))
	for i, r := range rankings {
		pkgs := make([]NebraskaPackage, len(r.Packages))
		for j, p := range r.Packages {
			pkgs[j] = NebraskaPackage{Package: p.Package, Role: string(p.Role), URL: p.URL}
		}
		result[i] = NebraskaData{
			Maintainer: r.Maintainer,
			Score:      r.Score,
			Packages:   pkgs,
		}
	}
	return result
}
