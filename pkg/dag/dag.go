package dag

import (
	"errors"
	"maps"
	"slices"
)

var (
	// ErrInvalidNodeID is returned by [DAG.AddNode] when the node ID is empty.
	ErrInvalidNodeID = errors.New("node ID must not be empty")

	// ErrDuplicateNodeID is returned by [DAG.AddNode] when a node with the
	// same ID already exists in the graph.
	ErrDuplicateNodeID = errors.New("duplicate node ID")

	// ErrUnknownSourceNode is returned by [DAG.AddEdge] when the From node
	// does not exist, or by [DAG.RenameNode] when the old ID is not found.
	ErrUnknownSourceNode = errors.New("unknown source node")

	// ErrUnknownTargetNode is returned by [DAG.AddEdge] when the To node
	// does not exist in the graph.
	ErrUnknownTargetNode = errors.New("unknown target node")

	// ErrInvalidEdgeEndpoint is returned by [DAG.Validate] when an edge
	// references a node that doesn't exist.
	ErrInvalidEdgeEndpoint = errors.New("invalid edge endpoint")

	// ErrNonConsecutiveRows is returned by [DAG.Validate] when an edge
	// connects nodes that are not in adjacent rows (From.Row+1 != To.Row).
	ErrNonConsecutiveRows = errors.New("edges must connect consecutive rows")

	// ErrGraphHasCycle is returned by [DAG.Validate] when a cycle is detected.
	// This indicates the graph is not a valid DAG.
	ErrGraphHasCycle = errors.New("graph contains a cycle")
)

// Metadata stores arbitrary key-value pairs attached to nodes or the graph.
type Metadata map[string]any

// NodeKind distinguishes between original and synthetic nodes.
type NodeKind int

const (
	NodeKindRegular    NodeKind = iota // Original graph nodes
	NodeKindSubdivider                 // Inserted to subdivide long edges
	NodeKindAuxiliary                  // Helper nodes for layout
)

// Node represents a vertex in the dependency graph.
type Node struct {
	ID   string   // Unique identifier (also used as display label)
	Row  int      // Layer assignment (0 = root/top)
	Meta Metadata // Arbitrary key-value metadata

	Kind     NodeKind // Node type (regular, subdivider, auxiliary)
	MasterID string   // Links synthetic nodes to their origin
}

// IsSubdivider reports whether the node was inserted to break a long edge.
func (n Node) IsSubdivider() bool { return n.Kind == NodeKindSubdivider }

// IsAuxiliary reports whether the node is a helper for layout (e.g., separator beam).
func (n Node) IsAuxiliary() bool { return n.Kind == NodeKindAuxiliary }

// IsSynthetic reports whether the node was created during graph transformation
// (subdivider or auxiliary), as opposed to an original graph vertex.
func (n Node) IsSynthetic() bool { return n.Kind != NodeKindRegular }

// EffectiveID returns MasterID if set (for subdividers), otherwise the node's ID.
// This allows subdivider chains to be treated as a single logical entity.
func (n Node) EffectiveID() string {
	if n.MasterID != "" {
		return n.MasterID
	}
	return n.ID
}

// Edge represents a directed connection between two nodes.
type Edge struct {
	From string   // Source node ID
	To   string   // Target node ID
	Meta Metadata // Arbitrary key-value metadata
}

// DAG is a directed acyclic graph optimized for row-based layered layouts.
type DAG struct {
	nodes    map[string]*Node
	edges    []Edge
	outgoing map[string][]string
	incoming map[string][]string
	rows     map[int][]*Node
	meta     Metadata
}

// New creates an empty DAG with optional graph-level metadata.
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

// Meta returns the graph-level metadata.
func (d *DAG) Meta() Metadata { return d.meta }

// AddNode adds a node to the graph. Returns an error if the node ID is empty
// or already exists. The node is automatically indexed by its Row.
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

// SetRows updates the row assignments for nodes and rebuilds the row index.
// Nodes not in the map retain their current row assignment.
func (d *DAG) SetRows(rows map[string]int) {
	d.rows = make(map[int][]*Node)
	for _, n := range d.nodes {
		if newRow, ok := rows[n.ID]; ok {
			n.Row = newRow
		}
		d.rows[n.Row] = append(d.rows[n.Row], n)
	}
}

// AddEdge adds a directed edge between two existing nodes.
// Returns an error if either endpoint does not exist.
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

// RemoveEdge removes the edge from→to if it exists. No error is returned
// if the edge does not exist.
func (d *DAG) RemoveEdge(from, to string) {
	d.edges = slices.DeleteFunc(d.edges, func(e Edge) bool { return e.From == from && e.To == to })
	d.outgoing[from] = slices.DeleteFunc(d.outgoing[from], func(s string) bool { return s == to })
	d.incoming[to] = slices.DeleteFunc(d.incoming[to], func(s string) bool { return s == from })
}

// RenameNode changes a node's ID, updating all edges and indices.
// Returns an error if oldID doesn't exist or newID is empty/duplicate.
func (d *DAG) RenameNode(oldID, newID string) error {
	if newID == "" {
		return ErrInvalidNodeID
	}
	node, ok := d.nodes[oldID]
	if !ok {
		return ErrUnknownSourceNode
	}
	if _, exists := d.nodes[newID]; exists {
		return ErrDuplicateNodeID
	}

	node.ID = newID
	delete(d.nodes, oldID)
	d.nodes[newID] = node

	for i := range d.edges {
		if d.edges[i].From == oldID {
			d.edges[i].From = newID
		}
		if d.edges[i].To == oldID {
			d.edges[i].To = newID
		}
	}

	d.outgoing[newID] = d.outgoing[oldID]
	delete(d.outgoing, oldID)
	for id, targets := range d.outgoing {
		for i, t := range targets {
			if t == oldID {
				d.outgoing[id][i] = newID
			}
		}
	}

	d.incoming[newID] = d.incoming[oldID]
	delete(d.incoming, oldID)
	for id, sources := range d.incoming {
		for i, s := range sources {
			if s == oldID {
				d.incoming[id][i] = newID
			}
		}
	}

	return nil
}

// Nodes returns all nodes in the graph. The order is not guaranteed.
func (d *DAG) Nodes() []*Node {
	nodes := make([]*Node, 0, len(d.nodes))
	for _, n := range d.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// Edges returns a copy of all edges in the graph.
func (d *DAG) Edges() []Edge { return slices.Clone(d.edges) }

// NodeCount returns the number of nodes in the graph.
func (d *DAG) NodeCount() int { return len(d.nodes) }

// EdgeCount returns the number of edges in the graph.
func (d *DAG) EdgeCount() int { return len(d.edges) }

// Children returns the IDs of nodes that this node has edges to (dependencies).
func (d *DAG) Children(id string) []string { return d.outgoing[id] }

// Parents returns the IDs of nodes that have edges to this node (dependents).
func (d *DAG) Parents(id string) []string { return d.incoming[id] }

// OutDegree returns the number of outgoing edges from the node.
func (d *DAG) OutDegree(id string) int { return len(d.outgoing[id]) }

// InDegree returns the number of incoming edges to the node.
func (d *DAG) InDegree(id string) int { return len(d.incoming[id]) }

// Node returns the node with the given ID and true, or nil and false if not found.
func (d *DAG) Node(id string) (*Node, bool) {
	n, ok := d.nodes[id]
	return n, ok
}

// ChildrenInRow returns children of the node that are in the specified row.
func (d *DAG) ChildrenInRow(id string, row int) []string {
	var result []string
	for _, c := range d.outgoing[id] {
		if n, ok := d.nodes[c]; ok && n.Row == row {
			result = append(result, c)
		}
	}
	return result
}

// ParentsInRow returns parents of the node that are in the specified row.
func (d *DAG) ParentsInRow(id string, row int) []string {
	var result []string
	for _, p := range d.incoming[id] {
		if n, ok := d.nodes[p]; ok && n.Row == row {
			result = append(result, p)
		}
	}
	return result
}

// NodesInRow returns all nodes assigned to the given row.
func (d *DAG) NodesInRow(row int) []*Node { return d.rows[row] }

// RowCount returns the number of distinct rows (layers) in the graph.
func (d *DAG) RowCount() int { return len(d.rows) }

// RowIDs returns all row indices in sorted order.
func (d *DAG) RowIDs() []int {
	return slices.Sorted(maps.Keys(d.rows))
}

// MaxRow returns the highest row index, or 0 if the graph is empty.
func (d *DAG) MaxRow() int {
	if len(d.rows) == 0 {
		return 0
	}
	rowIDs := d.RowIDs()
	return rowIDs[len(rowIDs)-1]
}

// Sources returns nodes with no incoming edges (roots/entry points).
func (d *DAG) Sources() []*Node {
	var sources []*Node
	for _, n := range d.nodes {
		if len(d.incoming[n.ID]) == 0 {
			sources = append(sources, n)
		}
	}
	return sources
}

// Sinks returns nodes with no outgoing edges (leaves/terminals).
func (d *DAG) Sinks() []*Node {
	var sinks []*Node
	for _, n := range d.nodes {
		if len(d.outgoing[n.ID]) == 0 {
			sinks = append(sinks, n)
		}
	}
	return sinks
}

// Validate checks graph integrity: edges must connect consecutive rows
// and the graph must be acyclic. Returns nil if valid.
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
	const (
		white = iota
		gray
		black
	)

	color := make(map[string]int, len(d.nodes))
	var hasCycle bool

	var dfs func(id string)
	dfs = func(id string) {
		color[id] = gray
		for _, child := range d.outgoing[id] {
			switch color[child] {
			case white:
				dfs(child)
			case gray:
				hasCycle = true
				return
			}
		}
		color[id] = black
	}

	for id := range d.nodes {
		if color[id] == white {
			dfs(id)
			if hasCycle {
				return ErrGraphHasCycle
			}
		}
	}
	return nil
}

// PosMap creates a position lookup map from a slice of node IDs.
func PosMap(ids []string) map[string]int {
	m := make(map[string]int, len(ids))
	for i, id := range ids {
		m[id] = i
	}
	return m
}

// NodePosMap creates a position lookup map from a slice of nodes.
func NodePosMap(nodes []*Node) map[string]int {
	m := make(map[string]int, len(nodes))
	for i, n := range nodes {
		m[n.ID] = i
	}
	return m
}

// NodeIDs extracts the ID from each node in a slice.
func NodeIDs(nodes []*Node) []string {
	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	return ids
}
