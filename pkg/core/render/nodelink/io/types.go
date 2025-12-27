package io

// LayoutData is the JSON-serializable representation of a nodelink layout.
// It stores the DOT graph definition for rendering with Graphviz.
type LayoutData struct {
	// Visualization type identifier (always "nodelink" for this package)
	VizType string `json:"viz_type"`

	// The DOT graph definition (Graphviz format)
	DOT string `json:"dot"`

	// Frame dimensions (informational, actual size determined by graphviz)
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`

	// Layout engine used (dot, neato, fdp, etc.)
	Engine string `json:"engine,omitempty"`

	// Render options (preserved for re-rendering)
	Style string `json:"style,omitempty"`

	// Graph statistics
	NodeCount int `json:"node_count,omitempty"`
	EdgeCount int `json:"edge_count,omitempty"`
}

// LayoutMeta contains render options extracted from a loaded layout.
type LayoutMeta struct {
	VizType string
	Engine  string
	Style   string
}
