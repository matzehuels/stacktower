package deps

import (
	"fmt"
	"path/filepath"
)

// ManifestParser reads dependency information from local manifest files.
type ManifestParser interface {
	// Parse reads the manifest at path and returns the dependency graph.
	Parse(path string, opts Options) (*ManifestResult, error)
	// Supports reports whether this parser handles the given filename.
	Supports(filename string) bool
	// Type returns the manifest type identifier (e.g., "poetry", "cargo").
	Type() string
	// IncludesTransitive reports whether the manifest contains the full
	// transitive closure (like lock files) or just direct dependencies.
	IncludesTransitive() bool
}

// ManifestResult holds the parsed dependency data from a manifest file.
type ManifestResult struct {
	Graph              any    // The dependency graph (typically *dag.DAG)
	Type               string // Parser type that produced this result
	IncludesTransitive bool   // Whether Graph includes transitive dependencies
	RootPackage        string // Name of the root package, if determinable
}

// DetectManifest finds a parser that supports the given file path.
// Returns an error if no parser matches.
func DetectManifest(path string, parsers ...ManifestParser) (ManifestParser, error) {
	name := filepath.Base(path)
	for _, p := range parsers {
		if p.Supports(name) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("unsupported manifest: %s", name)
}
