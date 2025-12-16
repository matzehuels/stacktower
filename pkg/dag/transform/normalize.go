package transform

import "github.com/matzehuels/stacktower/pkg/dag"

func Normalize(g *dag.DAG) *dag.DAG {
	BreakCycles(g)
	TransitiveReduction(g)
	AssignLayers(g)
	Subdivide(g)
	ResolveSpanOverlaps(g)
	return g
}
