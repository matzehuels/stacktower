package jobs

import "fmt"

// VizType constants for visualization types.
const (
	VizTypeTower    = "tower"
	VizTypeNodelink = "nodelink"
)

// LayoutPayload defines input parameters for layout jobs.
//
// A layout job computes visualization positions from a dependency graph
// and stores the result as layout.json in storage.
//
// Result:
//
//	{
//	  "layout_path": "job-123/layout.json",
//	  "viz_type": "tower",
//	  "blocks": 50
//	}
type LayoutPayload struct {
	// GraphPath is the storage path to the input graph.json (required).
	GraphPath string `json:"graph_path"`

	// VizType selects the visualization algorithm.
	// Options: "tower" (default), "nodelink"
	VizType string `json:"viz_type,omitempty"`

	// Width of the output frame in pixels.
	// Default: 800
	Width float64 `json:"width,omitempty"`

	// Height of the output frame in pixels.
	// Default: 600
	Height float64 `json:"height,omitempty"`

	// --- Tower-specific options ---

	// Ordering algorithm for block arrangement.
	// Options: "barycentric" (default), "optimal"
	Ordering string `json:"ordering,omitempty"`

	// Randomize applies random variation to block widths for visual interest.
	Randomize bool `json:"randomize,omitempty"`

	// Merge combines subdivider blocks that share the same parent.
	Merge bool `json:"merge,omitempty"`

	// Normalize applies DAG normalization before layout (if not done in parse).
	Normalize bool `json:"normalize,omitempty"`

	// Seed for random number generation (for reproducible randomization).
	// Default: 42
	Seed uint64 `json:"seed,omitempty"`

	// --- Nodelink-specific options ---

	// Engine selects the graphviz layout engine.
	// Options: "dot" (default), "neato", "fdp", "sfdp", "circo", "twopi"
	Engine string `json:"engine,omitempty"`

	// Webhook is an optional callback URL.
	Webhook string `json:"webhook,omitempty"`
}

// ValidateAndSetDefaults checks required fields and applies defaults.
func (p *LayoutPayload) ValidateAndSetDefaults() error {
	if p.GraphPath == "" {
		return fmt.Errorf("graph_path is required")
	}

	if p.VizType == "" {
		p.VizType = VizTypeTower
	}
	if p.Width == 0 {
		p.Width = 800
	}
	if p.Height == 0 {
		p.Height = 600
	}
	if p.Seed == 0 {
		p.Seed = 42
	}
	if p.Ordering == "" {
		p.Ordering = "barycentric"
	}
	if p.Engine == "" {
		p.Engine = "dot"
	}
	return nil
}

// IsTower returns true if this is a tower visualization.
func (p *LayoutPayload) IsTower() bool {
	return p.VizType == "" || p.VizType == VizTypeTower
}

// IsNodelink returns true if this is a nodelink visualization.
func (p *LayoutPayload) IsNodelink() bool {
	return p.VizType == VizTypeNodelink
}
