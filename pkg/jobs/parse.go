package jobs

import "fmt"

// ParsePayload defines input parameters for parse jobs.
//
// A parse job resolves dependencies for a package and stores the resulting
// dependency graph as graph.json in storage.
//
// Result:
//
//	{
//	  "graph_path": "job-123/graph.json",
//	  "nodes": 50,
//	  "edges": 80
//	}
type ParsePayload struct {
	// Language is the package ecosystem (required).
	// Supported: "python", "rust", "javascript", "ruby", "php", "java", "go"
	Language string `json:"language"`

	// Package is the package name or manifest file path (required).
	// Examples: "requests", "serde", "./package.json"
	// When Manifest is provided, this is optional (used for naming only).
	Package string `json:"package"`

	// Manifest is the raw manifest file content (e.g., package.json, requirements.txt).
	// When provided, the manifest is parsed directly instead of looking up a package.
	// Use this for GitHub repo integration where we have manifest content but no package name.
	Manifest string `json:"manifest,omitempty"`

	// ManifestFilename is the filename of the manifest (e.g., "package.json").
	// Required when Manifest is provided, used to determine the parser.
	ManifestFilename string `json:"manifest_filename,omitempty"`

	// MaxDepth limits how deep to traverse dependencies.
	// Default: 10
	MaxDepth int `json:"max_depth,omitempty"`

	// MaxNodes limits the total number of nodes in the graph.
	// Default: 5000
	MaxNodes int `json:"max_nodes,omitempty"`

	// Enrich fetches additional metadata from GitHub (stars, last commit, etc.).
	// Requires GITHUB_TOKEN environment variable.
	Enrich bool `json:"enrich,omitempty"`

	// Refresh bypasses the dependency cache and fetches fresh data.
	Refresh bool `json:"refresh,omitempty"`

	// Normalize applies DAG normalization (remove cycles, transitive edges).
	Normalize bool `json:"normalize,omitempty"`

	// Webhook is an optional callback URL to notify when the job completes.
	Webhook string `json:"webhook,omitempty"`
}

// ValidateAndSetDefaults checks required fields and applies defaults.
func (p *ParsePayload) ValidateAndSetDefaults() error {
	if p.Language == "" {
		return fmt.Errorf("language is required")
	}
	// Package is required unless Manifest is provided
	if p.Package == "" && p.Manifest == "" {
		return fmt.Errorf("package or manifest is required")
	}
	// ManifestFilename is required when Manifest is provided
	if p.Manifest != "" && p.ManifestFilename == "" {
		return fmt.Errorf("manifest_filename is required when manifest is provided")
	}

	if p.MaxDepth == 0 {
		p.MaxDepth = 10
	}
	if p.MaxNodes == 0 {
		p.MaxNodes = 5000
	}
	return nil
}
