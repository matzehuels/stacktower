package ordering

import (
	"cmp"
	"context"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/dag/perm"
)

const maxCandidates = 10000

type OptimalSearch struct {
	Progress func(explored, pruned, best int)
	Timeout  time.Duration
}

func (o OptimalSearch) OrderRows(g *dag.DAG) map[int][]string {
	rows := g.RowIDs()
	if len(rows) == 0 {
		return nil
	}

	timeout := o.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	initial := Barycentric{}.OrderRows(g)
	initialScore := dag.CountCrossings(g, initial)
	if initialScore == 0 {
		o.report(1, 0, 0)
		return initial
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	s := &solver{
		g:        g,
		fg:       newFastGraph(g, rows),
		rows:     rows,
		rowNodes: make(map[int][]*dag.Node, len(rows)),
		ctx:      ctx,
		cancel:   cancel,
	}
	s.bestScore.Store(int64(initialScore))
	s.bestPath.Store(toIndexPath(g, rows, initial))

	for _, r := range rows {
		s.rowNodes[r] = g.NodesInRow(r)
	}

	if o.Progress != nil {
		go s.monitor(o.Progress)
	}

	s.search()

	if o.Progress != nil {
		o.report(int(s.explored.Load()), int(s.pruned.Load()), int(s.bestScore.Load()))
	}

	return toStringOrder(s.rowNodes, s.rows, s.bestPath.Load().([][]int))
}

func (o OptimalSearch) report(explored, pruned, best int) {
	if o.Progress != nil {
		o.Progress(explored, pruned, best)
	}
}

type solver struct {
	g        *dag.DAG
	fg       *fastGraph
	rows     []int
	rowNodes map[int][]*dag.Node

	bestScore atomic.Int64
	bestPath  atomic.Value
	explored  atomic.Int64
	pruned    atomic.Int64

	ctx    context.Context
	cancel context.CancelFunc
}

func (s *solver) search() {
	workers := runtime.GOMAXPROCS(0)
	parallelRow := s.findParallelRow()

	prefix, prefixScore := s.buildPrefix(parallelRow)
	starts := s.generateStartPermutations(parallelRow, prefix, workers*100)

	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)

	for _, startPerm := range starts {
		if s.bestScore.Load() == 0 {
			break
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(start []int) {
			defer wg.Done()
			defer func() { <-sem }()

			path := make([][]int, len(s.rows))
			copy(path, prefix)
			path[parallelRow] = start

			score := prefixScore
			if parallelRow > 0 {
				ws := dag.NewCrossingWorkspace(s.fg.maxRowWidth)
				score += dag.CountCrossingsIdx(s.fg.edges[parallelRow-1], prefix[parallelRow-1], start, ws)
			}

			if score >= int(s.bestScore.Load()) {
				s.pruned.Add(1)
				return
			}

			s.dfs(parallelRow+1, score, path, dag.NewCrossingWorkspace(s.fg.maxRowWidth))
		}(startPerm)
	}

	wg.Wait()
}

func (s *solver) findParallelRow() int {
	for i, r := range s.rows {
		if len(s.rowNodes[r]) > 1 {
			return i
		}
	}
	return 0
}

func (s *solver) buildPrefix(parallelRow int) ([][]int, int) {
	prefix := make([][]int, len(s.rows))
	score := 0
	ws := dag.NewCrossingWorkspace(s.fg.maxRowWidth)

	for depth := 0; depth < parallelRow; depth++ {
		order := perm.Seq(len(s.rowNodes[s.rows[depth]]))
		prefix[depth] = order
		if depth > 0 {
			score += dag.CountCrossingsIdx(s.fg.edges[depth-1], prefix[depth-1], order, ws)
		}
	}
	return prefix, score
}

func (s *solver) generateStartPermutations(parallelRow int, prefix [][]int, workerLimit int) [][]int {
	parallelNodes := s.rowNodes[s.rows[parallelRow]]
	n := len(parallelNodes)

	var starts [][]int
	if parallelRow == 0 {
		if n <= 8 {
			starts = perm.Generate(n, -1)
		} else {
			starts = perm.Generate(n, workerLimit)
		}
	} else {
		prevNodes := s.rowNodes[s.rows[parallelRow-1]]
		starts = s.generateC1PCandidates(parallelRow, parallelNodes, prefix[parallelRow-1], prevNodes)
		if len(starts) > workerLimit {
			starts = starts[:workerLimit]
		}

		prevPos := make(map[string]int, len(prefix[parallelRow-1]))
		for i, idx := range prefix[parallelRow-1] {
			prevPos[prevNodes[idx].ID] = i
		}
		sortByBarycenter(starts, s.g, parallelNodes, prevPos)
	}

	return starts
}

func (s *solver) dfs(depth, score int, path [][]int, ws *dag.CrossingWorkspace) {
	if depth&0x0F == 0 && s.ctx.Err() != nil {
		return
	}

	if score >= int(s.bestScore.Load()) {
		s.pruned.Add(1)
		return
	}

	if depth == len(s.rows) {
		s.updateBest(path, score)
		return
	}

	rowID := s.rows[depth]
	nodes := s.rowNodes[rowID]
	if len(nodes) == 0 {
		path[depth] = nil
		s.dfs(depth+1, score, path, ws)
		return
	}

	prevOrder := path[depth-1]
	prevNodes := s.rowNodes[s.rows[depth-1]]
	prevPos := make(map[string]int, len(prevOrder))
	for i, idx := range prevOrder {
		prevPos[prevNodes[idx].ID] = i
	}

	candidates := s.generateC1PCandidates(depth, nodes, prevOrder, prevNodes)
	sortByBarycenter(candidates, s.g, nodes, prevPos)

	for _, candidate := range candidates {
		newScore := score + dag.CountCrossingsIdx(s.fg.edges[depth-1], prevOrder, candidate, ws)
		if newScore >= int(s.bestScore.Load()) {
			s.pruned.Add(1)
			continue
		}

		path[depth] = candidate
		s.dfs(depth+1, newScore, path, ws)

		if s.bestScore.Load() == 0 {
			return
		}
	}
}

func (s *solver) generateC1PCandidates(depth int, nodes []*dag.Node, prevOrder []int, prevNodes []*dag.Node) [][]int {
	n := len(nodes)
	if n <= 1 {
		return [][]int{perm.Seq(n)}
	}

	nodeIdx := buildNodeIndex(nodes)
	tree := perm.NewPQTree(n)

	if !s.applyParentConstraints(tree, nodeIdx, depth, prevOrder, prevNodes) {
		return s.fallbackPermutations(n)
	}
	if !s.applyChildConstraints(tree, nodeIdx, depth) {
		return s.fallbackPermutations(n)
	}

	limit := maxCandidates
	if n <= 8 {
		limit = tree.ValidCount()
	}

	perms := tree.Enumerate(limit)
	if len(perms) == 0 {
		return s.fallbackPermutations(n)
	}
	return perms
}

func (s *solver) applyParentConstraints(tree *perm.PQTree, nodeIdx map[string]int, depth int, prevOrder []int, prevNodes []*dag.Node) bool {
	row := s.rows[depth]
	for _, idx := range prevOrder {
		children := s.g.ChildrenInRow(prevNodes[idx].ID, row)
		if constraint := idsToIndices(children, nodeIdx); len(constraint) >= 2 {
			if !tree.Reduce(constraint) {
				return false
			}
		}
	}
	return true
}

func (s *solver) applyChildConstraints(tree *perm.PQTree, nodeIdx map[string]int, depth int) bool {
	if depth >= len(s.rows)-1 {
		return true
	}
	row := s.rows[depth]
	for _, child := range s.rowNodes[s.rows[depth+1]] {
		parents := s.g.ParentsInRow(child.ID, row)
		if constraint := idsToIndices(parents, nodeIdx); len(constraint) >= 2 {
			if !tree.Reduce(constraint) {
				return false
			}
		}
	}
	return true
}

func (s *solver) fallbackPermutations(n int) [][]int {
	if n <= 8 {
		return perm.Generate(n, -1)
	}
	return perm.Generate(n, maxCandidates)
}

func (s *solver) updateBest(path [][]int, score int) {
	s.explored.Add(1)

	for {
		current := int(s.bestScore.Load())
		if score >= current {
			return
		}
		if s.bestScore.CompareAndSwap(int64(current), int64(score)) {
			cloned := make([][]int, len(path))
			for i, p := range path {
				cloned[i] = slices.Clone(p)
			}
			s.bestPath.Store(cloned)
			if score == 0 {
				s.cancel()
			}
			return
		}
	}
}

func (s *solver) monitor(fn func(int, int, int)) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			fn(int(s.explored.Load()), int(s.pruned.Load()), int(s.bestScore.Load()))
		}
	}
}

type fastGraph struct {
	edges       [][][]int
	maxRowWidth int
}

func newFastGraph(g *dag.DAG, rows []int) *fastGraph {
	rowNodes := make(map[int][]*dag.Node, len(rows))
	maxWidth := 0
	for _, r := range rows {
		nodes := g.NodesInRow(r)
		rowNodes[r] = nodes
		if len(nodes) > maxWidth {
			maxWidth = len(nodes)
		}
	}

	fg := &fastGraph{
		edges:       make([][][]int, len(rows)-1),
		maxRowWidth: maxWidth,
	}

	for i := 0; i < len(rows)-1; i++ {
		upper := rowNodes[rows[i]]
		lower := rowNodes[rows[i+1]]

		lowerIdx := make(map[string]int, len(lower))
		for j, n := range lower {
			lowerIdx[n.ID] = j
		}

		fg.edges[i] = make([][]int, len(upper))
		for j, node := range upper {
			children := g.ChildrenInRow(node.ID, rows[i+1])
			targets := make([]int, 0, len(children))
			for _, child := range children {
				if idx, ok := lowerIdx[child]; ok {
					targets = append(targets, idx)
				}
			}
			slices.Sort(targets)
			fg.edges[i][j] = targets
		}
	}
	return fg
}

func sortByBarycenter(perms [][]int, g *dag.DAG, nodes []*dag.Node, prevPos map[string]int) {
	type scored struct {
		perm  []int
		score float64
	}
	s := make([]scored, len(perms))
	for i, p := range perms {
		s[i] = scored{p, barycenterDeviationIndices(g, nodes, p, prevPos, true)}
	}
	slices.SortFunc(s, func(a, b scored) int {
		return cmp.Compare(a.score, b.score)
	})
	for i, x := range s {
		perms[i] = x.perm
	}
}

func toIndexPath(g *dag.DAG, rows []int, order map[int][]string) [][]int {
	path := make([][]int, len(rows))
	for i, r := range rows {
		nodes := g.NodesInRow(r)
		nodeIdx := make(map[string]int, len(nodes))
		for j, n := range nodes {
			nodeIdx[n.ID] = j
		}
		indices := make([]int, len(nodes))
		for j, id := range order[r] {
			indices[j] = nodeIdx[id]
		}
		path[i] = indices
	}
	return path
}

func toStringOrder(rowNodes map[int][]*dag.Node, rows []int, path [][]int) map[int][]string {
	result := make(map[int][]string, len(rows))
	for i, r := range rows {
		if i >= len(path) || path[i] == nil {
			continue
		}
		nodes := rowNodes[r]
		ids := make([]string, len(path[i]))
		for j, idx := range path[i] {
			ids[j] = nodes[idx].ID
		}
		result[r] = ids
	}
	return result
}

func buildNodeIndex(nodes []*dag.Node) map[string]int {
	idx := make(map[string]int, len(nodes))
	for i, n := range nodes {
		idx[n.ID] = i
	}
	return idx
}

func idsToIndices(ids []string, idx map[string]int) []int {
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if i, ok := idx[id]; ok {
			result = append(result, i)
		}
	}
	return result
}
