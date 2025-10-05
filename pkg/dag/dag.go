package dag

import (
	"errors"
	"maps"
	"slices"
)

var (
	ErrInvalidNodeID       = errors.New("node ID must not be empty")
	ErrDuplicateNodeID     = errors.New("duplicate node ID")
	ErrUnknownSourceNode   = errors.New("unknown source node")
	ErrUnknownTargetNode   = errors.New("unknown target node")
	ErrInvalidEdgeEndpoint = errors.New("invalid edge endpoint")
	ErrNonConsecutiveRows  = errors.New("edges must connect consecutive rows")
	ErrGraphHasCycle       = errors.New("graph contains a cycle")
)

type Metadata map[string]any

type NodeKind int

const (
	NodeKindRegular    NodeKind = iota // Original graph nodes
	NodeKindSubdivider                 // Inserted to subdivide long edges
	NodeKindAuxiliary                  // Helper nodes for layout
)

type Node struct {
	ID   string
	Row  int
	Meta Metadata

	Kind     NodeKind
	MasterID string // Links synthetic nodes to their origin
}

func (n Node) IsSubdivider() bool { return n.Kind == NodeKindSubdivider }
func (n Node) IsAuxiliary() bool  { return n.Kind == NodeKindAuxiliary }
func (n Node) IsSynthetic() bool  { return n.Kind != NodeKindRegular }

func (n Node) EffectiveID() string {
	if n.MasterID != "" {
		return n.MasterID
	}
	return n.ID
}

type Edge struct {
	From string
	To   string
	Meta Metadata
}

type DAG struct {
	nodes    map[string]*Node
	edges    []Edge
	outgoing map[string][]string
	incoming map[string][]string
	rows     map[int][]*Node
	meta     Metadata
}

func New(meta Metadata) *DAG {
	if meta == nil {
		meta = Metadata{}
	}
	return &DAG{
		nodes:    make(map[string]*Node),
		outgoing: make(map[string][]string),
		incoming: make(map[string][]string),
		rows:     make(map[int][]*Node),
		meta:     meta,
	}
}

func (d *DAG) Meta() Metadata { return d.meta }

func (d *DAG) AddNode(n Node) error {
	if n.ID == "" {
		return ErrInvalidNodeID
	}
	if _, exists := d.nodes[n.ID]; exists {
		return ErrDuplicateNodeID
	}
	if n.Meta == nil {
		n.Meta = Metadata{}
	}
	node := &n
	d.nodes[node.ID] = node
	d.rows[node.Row] = append(d.rows[node.Row], node)
	return nil
}

func (d *DAG) SetRows(rows map[string]int) {
	d.rows = make(map[int][]*Node)
	for _, n := range d.nodes {
		if newRow, ok := rows[n.ID]; ok {
			n.Row = newRow
		}
		d.rows[n.Row] = append(d.rows[n.Row], n)
	}
}

func (d *DAG) AddEdge(e Edge) error {
	if _, ok := d.nodes[e.From]; !ok {
		return ErrUnknownSourceNode
	}
	if _, ok := d.nodes[e.To]; !ok {
		return ErrUnknownTargetNode
	}
	if e.Meta == nil {
		e.Meta = Metadata{}
	}
	d.edges = append(d.edges, e)
	d.outgoing[e.From] = append(d.outgoing[e.From], e.To)
	d.incoming[e.To] = append(d.incoming[e.To], e.From)
	return nil
}

func (d *DAG) RemoveEdge(from, to string) {
	d.edges = slices.DeleteFunc(d.edges, func(e Edge) bool { return e.From == from && e.To == to })
	d.outgoing[from] = slices.DeleteFunc(d.outgoing[from], func(s string) bool { return s == to })
	d.incoming[to] = slices.DeleteFunc(d.incoming[to], func(s string) bool { return s == from })
}

func (d *DAG) Nodes() []*Node {
	nodes := make([]*Node, 0, len(d.nodes))
	for _, n := range d.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

func (d *DAG) Edges() []Edge               { return slices.Clone(d.edges) }
func (d *DAG) NodeCount() int              { return len(d.nodes) }
func (d *DAG) EdgeCount() int              { return len(d.edges) }
func (d *DAG) Children(id string) []string { return d.outgoing[id] }
func (d *DAG) Parents(id string) []string  { return d.incoming[id] }
func (d *DAG) OutDegree(id string) int     { return len(d.outgoing[id]) }
func (d *DAG) InDegree(id string) int      { return len(d.incoming[id]) }

func (d *DAG) Node(id string) (*Node, bool) {
	n, ok := d.nodes[id]
	return n, ok
}

func (d *DAG) ChildrenInRow(id string, row int) []string {
	var result []string
	for _, c := range d.outgoing[id] {
		if n, ok := d.nodes[c]; ok && n.Row == row {
			result = append(result, c)
		}
	}
	return result
}

func (d *DAG) ParentsInRow(id string, row int) []string {
	var result []string
	for _, p := range d.incoming[id] {
		if n, ok := d.nodes[p]; ok && n.Row == row {
			result = append(result, p)
		}
	}
	return result
}

func (d *DAG) NodesInRow(row int) []*Node { return d.rows[row] }
func (d *DAG) RowCount() int              { return len(d.rows) }

func (d *DAG) RowIDs() []int {
	return slices.Sorted(maps.Keys(d.rows))
}

func (d *DAG) MaxRow() int {
	if len(d.rows) == 0 {
		return 0
	}
	rowIDs := d.RowIDs()
	return rowIDs[len(rowIDs)-1]
}

func (d *DAG) Sources() []*Node {
	var sources []*Node
	for _, n := range d.nodes {
		if len(d.incoming[n.ID]) == 0 {
			sources = append(sources, n)
		}
	}
	return sources
}

func (d *DAG) Sinks() []*Node {
	var sinks []*Node
	for _, n := range d.nodes {
		if len(d.outgoing[n.ID]) == 0 {
			sinks = append(sinks, n)
		}
	}
	return sinks
}

func (d *DAG) Validate() error {
	if err := d.validateEdgeConsistency(); err != nil {
		return err
	}
	return d.detectCycles()
}

func (d *DAG) validateEdgeConsistency() error {
	for _, e := range d.edges {
		src, okS := d.nodes[e.From]
		dst, okD := d.nodes[e.To]
		if !okS || !okD {
			return ErrInvalidEdgeEndpoint
		}
		if dst.Row != src.Row+1 {
			return ErrNonConsecutiveRows
		}
	}
	return nil
}

func (d *DAG) detectCycles() error {
	inDegree := make(map[string]int, len(d.nodes))
	for id := range d.nodes {
		inDegree[id] = len(d.incoming[id])
	}

	queue := make([]string, 0, len(d.nodes))
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	processed := 0
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		processed++

		for _, child := range d.outgoing[curr] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	if processed != len(d.nodes) {
		return ErrGraphHasCycle
	}
	return nil
}

func PosMap(ids []string) map[string]int {
	m := make(map[string]int, len(ids))
	for i, id := range ids {
		m[id] = i
	}
	return m
}

func NodePosMap(nodes []*Node) map[string]int {
	m := make(map[string]int, len(nodes))
	for i, n := range nodes {
		m[n.ID] = i
	}
	return m
}

func NodeIDs(nodes []*Node) []string {
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	return ids
}
