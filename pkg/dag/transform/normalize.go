package transform

import "github.com/matzehuels/stacktower/pkg/dag"

func Normalize(g *dag.DAG) *dag.DAG {
	TransitiveReduction(g)
	AssignLayers(g)
	Subdivide(g)
	ResolveSpanOverlaps(g)
	return g
}
