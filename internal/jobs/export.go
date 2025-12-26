package jobs

import "fmt"

// VisualizePayload defines input parameters for visualize jobs.
//
// A visualize job generates visual output (SVG, PNG, PDF) from a computed layout
// and stores the results in storage.
//
// Result:
//
//	{
//	  "svg": "job-123/tower.svg",
//	  "png": "job-123/tower.png"
//	}
type VisualizePayload struct {
	// LayoutPath is the storage path to the input layout.json (required).
	LayoutPath string `json:"layout_path"`

	// VizType must match the layout type. Auto-detected if not specified.
	// Options: "tower", "nodelink"
	VizType string `json:"viz_type,omitempty"`

	// Formats specifies output formats to generate (required).
	// Options: "svg", "png", "pdf"
	Formats []string `json:"formats"`

	// --- Style options ---

	// Style selects the visual style.
	// Tower options: "simple", "handdrawn" (default)
	// Nodelink options: "simple" (default)
	Style string `json:"style,omitempty"`

	// ShowEdges renders dependency edges.
	ShowEdges bool `json:"show_edges,omitempty"`

	// --- Tower-specific options ---

	// Popups enables hover popups with package metadata.
	Popups bool `json:"popups,omitempty"`

	// Webhook is an optional callback URL.
	Webhook string `json:"webhook,omitempty"`
}

// Validate checks that required fields are present.
func (p *VisualizePayload) Validate() error {
	if p.LayoutPath == "" {
		return fmt.Errorf("layout_path is required")
	}
	if len(p.Formats) == 0 {
		return fmt.Errorf("at least one format is required")
	}
	for _, f := range p.Formats {
		switch f {
		case "svg", "png", "pdf":
			// valid
		default:
			return fmt.Errorf("unsupported format: %s (use svg, png, or pdf)", f)
		}
	}
	return nil
}

// SetDefaults applies default values to unset fields.
func (p *VisualizePayload) SetDefaults() {
	if p.Style == "" {
		p.Style = "handdrawn"
	}
}
