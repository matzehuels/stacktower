package queue

// Type represents the kind of work to perform.
type Type string

const (
	// TypeParse resolves dependencies for a package.
	TypeParse Type = "parse"

	// TypeLayout computes visualization positions from a graph.
	TypeLayout Type = "layout"

	// TypeVisualize generates visual output (SVG, PNG, PDF) from a layout.
	TypeVisualize Type = "visualize"

	// TypeRender runs the full pipeline (parse -> layout -> visualize).
	TypeRender Type = "render"
)

// String returns the string representation of the job type.
func (t Type) String() string {
	return string(t)
}
