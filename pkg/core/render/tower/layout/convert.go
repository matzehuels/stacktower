package layout

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/deps/metadata"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/feature"
	"github.com/matzehuels/stacktower/pkg/dto"
)

// ToDTO converts an internal tower layout to the serialization format (dto.Layout).
//
// Use this when you need to serialize the layout for:
//   - JSON file output (via dto.WriteLayoutFile)
//   - API responses
//   - Caching
//
// The DAG is optional but recommended for metadata enrichment (URLs, brittle flags, etc.).
func (l Layout) ToDTO(g *dag.DAG) (dto.Layout, error) {
	result := dto.Layout{
		VizType:   dto.VizTypeTower,
		Width:     l.FrameWidth,
		Height:    l.FrameHeight,
		MarginX:   l.MarginX,
		MarginY:   l.MarginY,
		Style:     l.Style,
		Seed:      l.Seed,
		Randomize: l.Randomize,
		Merged:    l.Merged,
		Rows:      l.RowOrders,
		Blocks:    buildBlocks(l, g),
		Nodes:     buildNodes(g),
	}

	if g != nil {
		result.Edges = buildEdges(l, g, l.Merged)
	}

	if len(l.Nebraska) > 0 {
		result.Nebraska = buildNebraska(l.Nebraska)
	}

	return result, nil
}

// FromDTO converts a serialized layout (dto.Layout) to an internal tower layout.
//
// Use this when you need to render from a previously serialized layout:
//   - Loading from JSON file (via dto.ReadLayoutFile)
//   - Receiving from API/cache
//
// Returns an error if the DTO is not a tower layout (VizType must be "tower" or empty).
func FromDTO(d dto.Layout) (Layout, error) {
	if d.VizType != "" && d.VizType != dto.VizTypeTower {
		return Layout{}, fmt.Errorf("invalid viz_type for tower layout: %q", d.VizType)
	}

	l := Layout{
		FrameWidth:  d.Width,
		FrameHeight: d.Height,
		MarginX:     d.MarginX,
		MarginY:     d.MarginY,
		RowOrders:   d.Rows,
		Blocks:      make(map[string]Block, len(d.Blocks)),
		Style:       d.Style,
		Seed:        d.Seed,
		Randomize:   d.Randomize,
		Merged:      d.Merged,
		Nebraska:    convertNebraska(d.Nebraska),
	}

	for _, b := range d.Blocks {
		l.Blocks[b.ID] = Block{
			NodeID: b.Label,
			Left:   b.X,
			Right:  b.X + b.Width,
			Bottom: b.Y,
			Top:    b.Y + b.Height,
		}
	}

	return l, nil
}

// =============================================================================
// Block Building Helpers
// =============================================================================

func buildBlocks(l Layout, g *dag.DAG) []dto.Block {
	blocks := make([]dto.Block, 0, len(l.Blocks))
	for id, b := range l.Blocks {
		bd := dto.Block{
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
					bd.URL, _ = n.Meta[metadata.RepoURL].(string)
					bd.Brittle = feature.IsBrittle(n)
					bd.Meta = extractMeta(n)
				}
			}
		}
		blocks = append(blocks, bd)
	}
	return blocks
}

func buildNodes(g *dag.DAG) []dto.Node {
	if g == nil {
		return nil
	}
	graph := dto.FromDAG(g)
	return graph.Nodes
}

func extractMeta(n *dag.Node) *dto.BlockMeta {
	if n.Meta == nil {
		return nil
	}
	m := &dto.BlockMeta{
		Stars: feature.AsInt(n.Meta[metadata.RepoStars]),
	}
	m.LastCommit, _ = n.Meta[metadata.RepoLastCommit].(string)
	m.LastRelease, _ = n.Meta[metadata.RepoLastRelease].(string)
	m.Archived, _ = n.Meta[metadata.RepoArchived].(bool)

	if desc, ok := n.Meta[metadata.RepoDescription].(string); ok && desc != "" {
		m.Description = desc
	}

	if m.Description == "" && m.Stars == 0 && m.LastCommit == "" && m.LastRelease == "" && !m.Archived {
		return nil
	}
	return m
}

// =============================================================================
// Edge Building Helpers
// =============================================================================

func buildEdges(l Layout, g *dag.DAG, merged bool) []dto.Edge {
	if merged {
		return buildMergedEdges(l, g)
	}
	edges := make([]dto.Edge, 0)
	for _, e := range g.Edges() {
		if _, ok := l.Blocks[e.From]; !ok {
			continue
		}
		if _, ok := l.Blocks[e.To]; !ok {
			continue
		}
		edges = append(edges, dto.Edge{From: e.From, To: e.To})
	}
	return edges
}

func buildMergedEdges(l Layout, g *dag.DAG) []dto.Edge {
	masterOf := func(id string) string {
		if n, ok := g.Node(id); ok && n.MasterID != "" {
			return n.MasterID
		}
		return id
	}

	type edgeKey struct{ from, to string }
	seen := make(map[edgeKey]struct{})
	var edges []dto.Edge

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
		edges = append(edges, dto.Edge{From: fromMaster, To: toMaster})
	}
	return edges
}

// =============================================================================
// Nebraska Helpers
// =============================================================================

func buildNebraska(rankings []feature.NebraskaRanking) []dto.NebraskaRanking {
	result := make([]dto.NebraskaRanking, len(rankings))
	for i, r := range rankings {
		pkgs := make([]dto.NebraskaPackage, len(r.Packages))
		for j, p := range r.Packages {
			pkgs[j] = dto.NebraskaPackage{Package: p.Package, Role: string(p.Role), URL: p.URL}
		}
		result[i] = dto.NebraskaRanking{
			Maintainer: r.Maintainer,
			Score:      r.Score,
			Packages:   pkgs,
		}
	}
	return result
}

func convertNebraska(data []dto.NebraskaRanking) []feature.NebraskaRanking {
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
