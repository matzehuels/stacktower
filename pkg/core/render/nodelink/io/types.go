package io

// LayoutData is the JSON-serializable representation of a nodelink layout.
// This structure stores computed node positions from graphviz.
type LayoutData struct {
	// Visualization type identifier (always "nodelink" for this package)
	VizType string `json:"viz_type"`

	// Frame dimensions
	Width  float64 `json:"width"`
	Height float64 `json:"height"`

	// Layout engine used (dot, neato, fdp, etc.)
	Engine string `json:"engine,omitempty"`

	// Render options (preserved for re-rendering)
	Style string `json:"style,omitempty"`

	// Computed positions
	Nodes []NodeData `json:"nodes"`
	Edges []EdgeData `json:"edges,omitempty"`
}

// NodeData represents a positioned node in the graph.
type NodeData struct {
	ID     string  `json:"id"`
	Label  string  `json:"label,omitempty"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`

	// Optional metadata from the original graph
	RepoURL     string `json:"repo_url,omitempty"`
	RepoStars   int    `json:"repo_stars,omitempty"`
	Description string `json:"description,omitempty"`
}

// EdgeData represents a directed edge between nodes.
type EdgeData struct {
	From   string      `json:"from"`
	To     string      `json:"to"`
	Points []PointData `json:"points,omitempty"` // Control points for curved edges
}

// PointData represents a point in 2D space.
type PointData struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// LayoutMeta contains render options extracted from a loaded layout.
type LayoutMeta struct {
	VizType string
	Engine  string
	Style   string
}
