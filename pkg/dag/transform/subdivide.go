package transform

import (
	"fmt"

	"stacktower/pkg/dag"
)

func Subdivide(g *dag.DAG) {
	gen := newIDGen(g.Nodes())
	subdivideLongEdges(g, gen)
	extendSinksToBottom(g, gen)
}

func subdivideLongEdges(g *dag.DAG, gen *idGen) {
	var toRemove []dag.Edge
	for _, e := range g.Edges() {
		src, srcOK := g.Node(e.From)
		dst, dstOK := g.Node(e.To)
		if !srcOK || !dstOK || dst.Row <= src.Row+1 {
			continue
		}

		toRemove = append(toRemove, e)
		prevID := src.ID
		for row := src.Row + 1; row < dst.Row; row++ {
			prevID = addSubdivider(g, gen, prevID, src.ID, row)
		}
		if err := g.AddEdge(dag.Edge{From: prevID, To: dst.ID, Meta: e.Meta}); err != nil {
			panic(err)
		}
	}

	for _, e := range toRemove {
		g.RemoveEdge(e.From, e.To)
	}
}

func addSubdivider(g *dag.DAG, gen *idGen, from, master string, row int) string {
	id := gen.next(master, row)
	if err := g.AddNode(dag.Node{
		ID:       id,
		Row:      row,
		Kind:     dag.NodeKindSubdivider,
		MasterID: master,
	}); err != nil {
		panic(err)
	}
	if err := g.AddEdge(dag.Edge{From: from, To: id}); err != nil {
		panic(err)
	}
	return id
}

func extendSinksToBottom(g *dag.DAG, gen *idGen) {
	maxRow := g.MaxRow()
	for _, n := range g.Nodes() {
		if g.OutDegree(n.ID) > 0 || n.Row >= maxRow {
			continue
		}
		prevID := n.ID
		for row := n.Row + 1; row <= maxRow; row++ {
			prevID = addSubdivider(g, gen, prevID, n.EffectiveID(), row)
		}
	}
}

type idGen struct {
	used map[string]struct{}
}

func newIDGen(nodes []*dag.Node) *idGen {
	m := make(map[string]struct{}, len(nodes)*2)
	for _, n := range nodes {
		m[n.ID] = struct{}{}
	}
	return &idGen{used: m}
}

func (gen *idGen) next(base string, row int) string {
	id := fmt.Sprintf("%s_sub_%d", base, row)
	if _, exists := gen.used[id]; !exists {
		gen.used[id] = struct{}{}
		return id
	}

	for i := 1; ; i++ {
		id = fmt.Sprintf("%s_sub_%d__%d", base, row, i)
		if _, exists := gen.used[id]; !exists {
			gen.used[id] = struct{}{}
			return id
		}
	}
}
