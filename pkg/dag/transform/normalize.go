package transform

import "stacktower/pkg/dag"

func Normalize(g *dag.DAG) *dag.DAG {
	TransitiveReduction(g)
	AssignLayers(g)
	Subdivide(g)
	ResolveSpanOverlaps(g)
	return g
}
