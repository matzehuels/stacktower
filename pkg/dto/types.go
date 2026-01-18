package dto

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/deps/metadata"
)

// =============================================================================
// Constants - Single Source of Truth
// =============================================================================

// Visualization types.
const (
	VizTypeTower    = "tower"
	VizTypeNodelink = "nodelink"
)

// Visual styles for rendering.
const (
	StyleSimple    = "simple"
	StyleHanddrawn = "handdrawn"
)

// Node kinds.
const (
	KindSubdivider = "subdivider"
	KindAuxiliary  = "auxiliary"
)

// =============================================================================
// Graph - Dependency Graph Serialization
// =============================================================================

// Graph is the canonical serialization format for dependency graphs.
// Used for API responses, storage, caching, and cross-tool compatibility.
//
// The format is human-readable and designed for round-trip fidelity:
// import → transform → export → re-import produces identical results.
type Graph struct {
	Nodes []Node `json:"nodes" bson:"nodes"`
	Edges []Edge `json:"edges" bson:"edges"`
}

// =============================================================================
// Node - Unified Node Type
// =============================================================================

// Node is the unified node type for all serialization contexts.
// Used in both Graph and Layout types for consistency.
type Node struct {
	ID       string         `json:"id" bson:"id"`
	Label    string         `json:"label,omitempty" bson:"label,omitempty"`     // Display label (defaults to ID)
	Row      int            `json:"row,omitempty" bson:"row,omitempty"`         // Layer/rank assignment
	Kind     string         `json:"kind,omitempty" bson:"kind,omitempty"`       // "subdivider", "auxiliary", or empty
	Brittle  bool           `json:"brittle,omitempty" bson:"brittle,omitempty"` // At-risk package flag
	MasterID string         `json:"master_id,omitempty" bson:"master_id,omitempty"`
	URL      string         `json:"url,omitempty" bson:"url,omitempty"` // Repository URL
	Meta     map[string]any `json:"meta,omitempty" bson:"meta,omitempty"`
}

// IsSubdivider returns true if this is a subdivider node.
func (n *Node) IsSubdivider() bool { return n.Kind == KindSubdivider }

// IsAuxiliary returns true if this is an auxiliary dependency.
func (n *Node) IsAuxiliary() bool { return n.Kind == KindAuxiliary }

// DisplayLabel returns the label if set, otherwise the ID.
func (n *Node) DisplayLabel() string {
	if n.Label != "" {
		return n.Label
	}
	return n.ID
}

// =============================================================================
// Edge - Directed Dependency
// =============================================================================

// Edge represents a directed edge in the dependency graph.
type Edge struct {
	From string `json:"from" bson:"from"`
	To   string `json:"to" bson:"to"`
}

// =============================================================================
// DAG ↔ Graph Conversion
// =============================================================================

// FromDAG converts a DAG to its serialization format.
// Nodes are sorted by ID for deterministic output.
// Extracts repository URL and computes brittle flag from metadata.
func FromDAG(g *dag.DAG) Graph {
	nodes := g.Nodes()
	slices.SortFunc(nodes, func(a, b *dag.Node) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})

	out := Graph{
		Nodes: make([]Node, len(nodes)),
		Edges: make([]Edge, len(g.Edges())),
	}

	for i, n := range nodes {
		out.Nodes[i] = nodeFromDAG(n)
	}

	for i, e := range g.Edges() {
		out.Edges[i] = Edge{From: e.From, To: e.To}
	}

	return out
}

// ToDAG converts a Graph to a DAG.
// Returns an error if the structure violates DAG constraints.
func ToDAG(gj Graph) (*dag.DAG, error) {
	d := dag.New(nil)

	for _, nj := range gj.Nodes {
		n := dag.Node{
			ID:       nj.ID,
			Row:      nj.Row,
			Meta:     nj.Meta,
			Kind:     stringToDAGKind(nj.Kind),
			MasterID: nj.MasterID,
		}
		if n.Meta == nil {
			n.Meta = dag.Metadata{}
		}
		if err := d.AddNode(n); err != nil {
			return nil, fmt.Errorf("add node %s: %w", nj.ID, err)
		}
	}

	for _, ej := range gj.Edges {
		if err := d.AddEdge(dag.Edge{From: ej.From, To: ej.To}); err != nil {
			return nil, fmt.Errorf("add edge %s→%s: %w", ej.From, ej.To, err)
		}
	}

	return d, nil
}

// UnmarshalGraph deserializes JSON bytes to a Graph.
func UnmarshalGraph(data []byte) (Graph, error) {
	var g Graph
	if err := json.Unmarshal(data, &g); err != nil {
		return Graph{}, err
	}
	return g, nil
}

// =============================================================================
// Internal Helpers
// =============================================================================

// nodeFromDAG converts a dag.Node to a serialization Node.
// This is the single point of conversion for all DAG→Node operations.
// Label is omitted when it equals ID (the default).
func nodeFromDAG(n *dag.Node) Node {
	node := Node{
		ID:       n.ID,
		Row:      n.Row,
		MasterID: n.MasterID,
		Meta:     n.Meta,
		Kind:     dagKindToString(n.Kind),
		Brittle:  isBrittle(n),
	}

	// Extract URL from metadata
	if n.Meta != nil {
		if url, ok := n.Meta[metadata.RepoURL].(string); ok {
			node.URL = url
		}
	}

	return node
}

func dagKindToString(k dag.NodeKind) string {
	switch k {
	case dag.NodeKindSubdivider:
		return KindSubdivider
	case dag.NodeKindAuxiliary:
		return KindAuxiliary
	default:
		return ""
	}
}

func stringToDAGKind(s string) dag.NodeKind {
	switch s {
	case KindSubdivider:
		return dag.NodeKindSubdivider
	case KindAuxiliary:
		return dag.NodeKindAuxiliary
	default:
		return dag.NodeKindRegular
	}
}

// isBrittle computes the brittle flag from node metadata.
// A package is brittle if it has few maintainers and low star count.
func isBrittle(n *dag.Node) bool {
	if n.Meta == nil {
		return false
	}

	// Check maintainers count
	maintainers, _ := n.Meta[metadata.RepoMaintainers].([]string)
	hasFewMaintainers := len(maintainers) > 0 && len(maintainers) <= 2

	// Check star count
	stars, _ := n.Meta[metadata.RepoStars].(int)
	hasLowStars := stars > 0 && stars < 100

	return hasFewMaintainers || hasLowStars
}

