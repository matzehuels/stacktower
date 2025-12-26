package io

import (
	"github.com/matzehuels/stacktower/pkg/render/tower/feature"
)

// VizType identifier for tower layouts.
const VizType = "tower"

// LayoutData is the JSON-serializable representation of a tower layout.
// This structure is used for both export and import operations.
type LayoutData struct {
	// Visualization type identifier (always "tower" for this package)
	VizType string `json:"viz_type,omitempty"`

	// Frame dimensions
	Width   float64 `json:"width"`
	Height  float64 `json:"height"`
	MarginX float64 `json:"margin_x"`
	MarginY float64 `json:"margin_y"`

	// Render options (preserved for re-rendering)
	Style     string `json:"style,omitempty"`
	Seed      uint64 `json:"seed,omitempty"`
	Randomize bool   `json:"randomize,omitempty"`
	Merged    bool   `json:"merged,omitempty"`

	// Layout structure
	Rows   map[int][]string `json:"rows,omitempty"`
	Blocks []BlockData      `json:"blocks"`
	Edges  []EdgeData       `json:"edges,omitempty"`

	// Optional features
	Nebraska []NebraskaData `json:"nebraska,omitempty"`
}

// BlockData represents a single block's position and metadata.
type BlockData struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`

	// Metadata (optional, from DAG)
	URL       string     `json:"url,omitempty"`
	Brittle   bool       `json:"brittle,omitempty"`
	Auxiliary bool       `json:"auxiliary,omitempty"`
	Synthetic bool       `json:"synthetic,omitempty"`
	Meta      *BlockMeta `json:"meta,omitempty"`
}

// BlockMeta contains rich metadata for a block (from GitHub, package registries, etc.).
type BlockMeta struct {
	Description string `json:"description,omitempty"`
	Stars       int    `json:"stars,omitempty"`
	LastCommit  string `json:"last_commit,omitempty"`
	LastRelease string `json:"last_release,omitempty"`
	Archived    bool   `json:"archived,omitempty"`
}

// EdgeData represents a dependency edge between blocks.
type EdgeData struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// NebraskaData contains maintainer ranking information.
type NebraskaData struct {
	Maintainer string            `json:"maintainer"`
	Score      float64           `json:"score"`
	Packages   []NebraskaPackage `json:"packages"`
}

// NebraskaPackage represents a package maintained by someone.
type NebraskaPackage struct {
	Package string `json:"package"`
	Role    string `json:"role"` // "owner", "lead", or "maintainer"
	URL     string `json:"url,omitempty"`
}

// LayoutMeta contains render options extracted from a loaded layout.
// Use this to configure sink renderers with the same options used when the layout was saved.
type LayoutMeta struct {
	// VizType is the visualization type (always "tower" for this package).
	VizType string

	// Style is the render style ("simple", "handdrawn").
	Style string

	// Seed is the random seed for reproducible rendering.
	Seed uint64

	// Randomize indicates if block widths were randomized.
	Randomize bool

	// Merged indicates if subdividers were merged.
	Merged bool

	// Nebraska contains maintainer ranking data.
	Nebraska []feature.NebraskaRanking

	// Edges contains dependency edge data.
	Edges []EdgeData
}
