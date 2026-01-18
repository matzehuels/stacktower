package pipeline

import (
	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/nodelink"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/feature"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/layout"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/transform"
	"github.com/matzehuels/stacktower/pkg/dto"
)

// =============================================================================
// Layout DTO Generation
// =============================================================================

// GenerateLayoutDTO generates a complete layout DTO for any visualization type.
// This is the unified entry point for generating serializable layout data.
//
// Both tower and nodelink layouts include:
//   - Graph structure (nodes, edges, rows)
//   - Nebraska rankings (maintainer data)
//   - Visualization-specific data (blocks for tower, DOT for nodelink)
func GenerateLayoutDTO(g *dag.DAG, opts Options) (dto.Layout, error) {
	if opts.IsNodelink() {
		return generateNodelinkDTO(g, opts)
	}
	return generateTowerDTO(g, opts)
}

// =============================================================================
// Tower
// =============================================================================

// generateTowerDTO generates a complete tower layout DTO.
// Computes positions, applies transforms, and includes Nebraska rankings.
func generateTowerDTO(g *dag.DAG, opts Options) (dto.Layout, error) {
	// Ensure graph has row assignments
	workGraph := g
	if g.MaxRow() == 0 && g.EdgeCount() > 0 {
		workGraph = g.Clone()
		layout.EnsureLayered(workGraph)
	}

	// Build layout options
	var layoutOpts []layout.Option
	if opts.Orderer != nil {
		layoutOpts = append(layoutOpts, layout.WithOrderer(opts.Orderer))
	}

	// Compute base layout
	l := layout.Build(workGraph, opts.Width, opts.Height, layoutOpts...)

	// Apply transforms
	if opts.Merge {
		l = transform.MergeSubdividers(l, workGraph)
	}
	if opts.Randomize {
		l = transform.Randomize(l, workGraph, opts.Seed, nil)
	}

	// Set metadata
	l.Style = opts.Style
	l.Seed = opts.Seed
	l.Randomize = opts.Randomize
	l.Merged = opts.Merge

	// Compute Nebraska rankings
	l.Nebraska = feature.RankNebraska(workGraph, 10)

	// Convert to DTO
	return l.ToDTO(workGraph)
}

// =============================================================================
// Nodelink
// =============================================================================

// generateNodelinkDTO generates a complete nodelink layout DTO.
// Includes DOT string for Graphviz and Nebraska rankings.
func generateNodelinkDTO(g *dag.DAG, opts Options) (dto.Layout, error) {
	// Generate DOT representation
	dot := nodelink.ToDOT(g, nodelink.Options{Detailed: false})

	// Build base layout DTO
	layoutDTO, err := nodelink.ToDTO(dot, g, nodelink.Options{Detailed: false}, opts.Width, opts.Height, opts.Style)
	if err != nil {
		return layoutDTO, err
	}

	// Add Nebraska rankings
	layoutDTO.Nebraska = nebraskaToDTO(feature.RankNebraska(g, 10))

	return layoutDTO, nil
}

// =============================================================================
// Helpers
// =============================================================================

// nebraskaToDTO converts internal Nebraska rankings to DTO format.
func nebraskaToDTO(rankings []feature.NebraskaRanking) []dto.NebraskaRanking {
	result := make([]dto.NebraskaRanking, len(rankings))
	for i, r := range rankings {
		pkgs := make([]dto.NebraskaPackage, len(r.Packages))
		for j, p := range r.Packages {
			pkgs[j] = dto.NebraskaPackage{
				Package: p.Package,
				Role:    string(p.Role),
				URL:     p.URL,
			}
		}
		result[i] = dto.NebraskaRanking{
			Maintainer: r.Maintainer,
			Score:      r.Score,
			Packages:   pkgs,
		}
	}
	return result
}
