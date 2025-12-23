package deps

import (
	"fmt"
	"path/filepath"
)

// ManifestParser reads dependency information from local manifest files.
//
// Manifest files describe a project's dependencies and may be either:
//   - Requirement files (package.json, requirements.txt) with direct deps only
//   - Lock files (poetry.lock, Cargo.lock) with full transitive closures
//
// Implementations are found in language subpackages (e.g., python.PoetryParser).
type ManifestParser interface {
	// Parse reads the manifest file at path and returns the dependency graph.
	//
	// The path is typically a local file system path. Options may influence
	// parsing behavior (e.g., MaxDepth for resolvers that fetch additional data).
	//
	// Returns an error if the file cannot be read, is malformed, or if
	// dependency resolution fails. Common errors:
	//   - File not found or unreadable
	//   - Invalid JSON/TOML/YAML syntax
	//   - Missing required fields
	//   - Dependency fetching failures (if the parser resolves transitive deps)
	Parse(path string, opts Options) (*ManifestResult, error)

	// Supports reports whether this parser handles the given filename.
	//
	// The filename is typically the basename of a path (e.g., "package.json").
	// Returns true if this parser recognizes the file format.
	Supports(filename string) bool

	// Type returns the manifest type identifier (e.g., "poetry", "cargo", "npm").
	//
	// This identifier appears in ManifestResult.Type and is used for
	// logging and error messages.
	Type() string

	// IncludesTransitive reports whether this parser produces transitive deps.
	//
	// Returns true for lock files (poetry.lock, Cargo.lock) that contain the
	// full dependency closure. Returns false for requirement files (requirements.txt,
	// package.json) that only list direct dependencies.
	//
	// This is used by the CLI to decide whether additional resolution is needed.
	IncludesTransitive() bool
}

// ManifestResult holds the parsed dependency data from a manifest file.
//
// Returned by [ManifestParser.Parse] after successfully reading a manifest.
type ManifestResult struct {
	// Graph is the dependency graph, typically a *dag.DAG with nodes for
	// packages and edges for dependencies. The concrete type depends on
	// the parser implementation.
	Graph any

	// Type is the manifest type identifier (from ManifestParser.Type).
	// Examples: "poetry", "cargo", "npm", "requirements".
	Type string

	// IncludesTransitive indicates whether Graph contains the full transitive
	// closure (true for lock files) or just direct dependencies (false).
	IncludesTransitive bool

	// RootPackage is the name of the root package, if determinable from the
	// manifest. Empty if the manifest doesn't specify a package name (e.g.,
	// requirements.txt has no root package).
	RootPackage string
}

// DetectManifest finds a parser that supports the given file path.
//
// The path is matched against each parser's Supports method using the basename.
// Parsers are checked in order, and the first match is returned.
//
// Typical usage:
//
//	lang := python.Language
//	parsers := lang.ManifestParsers(nil)
//	parser, err := deps.DetectManifest("poetry.lock", parsers...)
//	if err != nil {
//	    // No parser supports poetry.lock
//	}
//	result, err := parser.Parse("poetry.lock", opts)
//
// Returns an error if no parser in the list supports the file. An empty
// parsers list always returns an error.
func DetectManifest(path string, parsers ...ManifestParser) (ManifestParser, error) {
	name := filepath.Base(path)
	for _, p := range parsers {
		if p.Supports(name) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("unsupported manifest: %s", name)
}
