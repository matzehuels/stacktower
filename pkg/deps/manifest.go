package deps

import (
	"fmt"
	"path/filepath"
	"strings"
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

// ManifestInfo describes a manifest file and its support status.
type ManifestInfo struct {
	// Filename is the manifest file name (e.g., "package.json", "go.mod").
	Filename string

	// Language is the language name (e.g., "python", "javascript").
	Language string

	// ManifestType is the internal type identifier (e.g., "poetry", "cargo").
	ManifestType string

	// Supported indicates whether stacktower can parse this manifest.
	Supported bool
}

// KnownManifests lists all manifest files that stacktower knows about.
// This includes both supported manifests (from Language.ManifestAliases) and
// commonly encountered manifests that are not yet supported.
//
// The languages parameter should contain all Language definitions to aggregate.
// Additional unsupported manifests can be added via extraUnsupported.
func KnownManifests(languages []*Language, extraUnsupported map[string]string) []ManifestInfo {
	var result []ManifestInfo
	seen := make(map[string]bool)

	// First, add all supported manifests from languages
	for _, lang := range languages {
		for filename, manifestType := range lang.ManifestAliases {
			if seen[filename] {
				continue
			}
			seen[filename] = true
			result = append(result, ManifestInfo{
				Filename:     filename,
				Language:     lang.Name,
				ManifestType: manifestType,
				Supported:    true,
			})
		}
	}

	// Then add extra unsupported manifests
	for filename, language := range extraUnsupported {
		if seen[filename] {
			continue
		}
		seen[filename] = true
		result = append(result, ManifestInfo{
			Filename:     filename,
			Language:     language,
			ManifestType: "",
			Supported:    false,
		})
	}

	return result
}

// SupportedManifests returns a map of filename -> language for all supported manifests.
// This is a convenience function for quick lookups.
func SupportedManifests(languages []*Language) map[string]string {
	result := make(map[string]string)
	for _, lang := range languages {
		for filename := range lang.ManifestAliases {
			result[filename] = lang.Name
		}
	}
	return result
}

// IsManifestSupported checks if a manifest filename is supported by any of the languages.
func IsManifestSupported(filename string, languages []*Language) bool {
	for _, lang := range languages {
		if _, ok := lang.ManifestAliases[filename]; ok {
			return true
		}
	}
	return false
}

// GetManifestLanguage returns the language name for a manifest file, if supported.
// Returns empty string if the manifest is not supported.
func GetManifestLanguage(filename string, languages []*Language) string {
	for _, lang := range languages {
		if _, ok := lang.ManifestAliases[filename]; ok {
			return lang.Name
		}
	}
	return ""
}

// NormalizeLanguageName maps external language names (e.g., from GitHub API)
// to our standard internal names. Returns the original (lowercased) if no mapping exists.
func NormalizeLanguageName(name string, languages []*Language) string {
	if name == "" {
		return ""
	}

	// Build a case-insensitive lookup map
	lower := strings.ToLower(name)

	// Check against our language names
	for _, lang := range languages {
		if strings.ToLower(lang.Name) == lower {
			return lang.Name
		}
	}

	// Common aliases not covered by Language.Name
	aliases := map[string]string{
		"golang": "go",
	}
	if mapped, ok := aliases[lower]; ok {
		// Find the actual language name
		for _, lang := range languages {
			if lang.Name == mapped {
				return lang.Name
			}
		}
	}

	// Return lowercase if no match
	return lower
}
