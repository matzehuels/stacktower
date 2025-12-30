package queue

// Type represents the kind of work to perform.
type Type string

const (
	// TypeParse resolves dependencies for a package.
	TypeParse Type = "parse"

	// TypeLayout computes visualization positions from a graph.
	TypeLayout Type = "layout"

	// TypeRender runs the full pipeline (parse -> layout -> visualize).
	TypeRender Type = "render"
)

// String returns the string representation of the job type.
func (t Type) String() string {
	return string(t)
}

// SupportedJobTypes is the canonical list of job types that workers process.
// This is the single source of truth - use this instead of repeating the list.
var SupportedJobTypes = []Type{
	TypeParse,
	TypeLayout,
	TypeRender,
}

// SupportedJobTypeStrings returns the job types as strings for queue operations.
func SupportedJobTypeStrings() []string {
	result := make([]string, len(SupportedJobTypes))
	for i, t := range SupportedJobTypes {
		result[i] = string(t)
	}
	return result
}
