package deps

import (
	"context"
	"maps"
	"time"
)

const (
	DefaultMaxDepth = 50             // Default maximum dependency depth
	DefaultMaxNodes = 5000           // Default maximum packages to fetch
	DefaultCacheTTL = 24 * time.Hour // Default HTTP cache duration
)

// Options configures dependency resolution behavior.
type Options struct {
	MaxDepth          int                  // Maximum depth to traverse (default: 50)
	MaxNodes          int                  // Maximum packages to fetch (default: 5000)
	CacheTTL          time.Duration        // HTTP cache duration (default: 24h)
	Refresh           bool                 // Bypass cache for fresh data
	MetadataProviders []MetadataProvider   // Sources for enrichment (GitHub, etc.)
	Logger            func(string, ...any) // Progress/error callback (optional)
}

// WithDefaults returns a copy of Options with zero values replaced by defaults.
func (o Options) WithDefaults() Options {
	opts := o
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = DefaultMaxDepth
	}
	if opts.MaxNodes <= 0 {
		opts.MaxNodes = DefaultMaxNodes
	}
	if opts.CacheTTL <= 0 {
		opts.CacheTTL = DefaultCacheTTL
	}
	if opts.Logger == nil {
		opts.Logger = func(string, ...any) {}
	}
	return opts
}

// MetadataProvider enriches package nodes with external data (e.g., GitHub stars).
type MetadataProvider interface {
	// Name returns the provider identifier (e.g., "github").
	Name() string
	// Enrich fetches additional metadata for the package.
	Enrich(ctx context.Context, pkg *PackageRef, refresh bool) (map[string]any, error)
}

// PackageRef identifies a package for metadata enrichment lookups.
type PackageRef struct {
	Name         string            // Package name
	Version      string            // Package version
	ProjectURLs  map[string]string // URLs from registry (repository, homepage, etc.)
	HomePage     string            // Homepage URL
	ManifestFile string            // Associated manifest file type
}

// Package holds metadata fetched from a package registry.
type Package struct {
	Name         string            // Package name
	Version      string            // Latest or specified version
	Dependencies []string          // Direct dependency names
	Description  string            // Package summary/description
	License      string            // License identifier
	Author       string            // Primary author or maintainer
	Downloads    int               // Download count (where available)
	Repository   string            // Source repository URL
	HomePage     string            // Project homepage URL
	ProjectURLs  map[string]string // Additional URLs from registry
	ManifestFile string            // Associated manifest type
}

// Metadata converts Package fields to a map for node metadata.
func (p *Package) Metadata() map[string]any {
	m := map[string]any{"version": p.Version}
	if p.Description != "" {
		m["description"] = p.Description
	}
	if p.License != "" {
		m["license"] = p.License
	}
	if p.Author != "" {
		m["author"] = p.Author
	}
	if p.Downloads > 0 {
		m["downloads"] = p.Downloads
	}
	return m
}

// Ref creates a PackageRef for metadata provider lookups.
func (p *Package) Ref() *PackageRef {
	urls := maps.Clone(p.ProjectURLs)
	if urls == nil {
		urls = make(map[string]string)
	}
	if p.Repository != "" {
		urls["repository"] = p.Repository
	}
	if p.HomePage != "" {
		urls["homepage"] = p.HomePage
	}
	return &PackageRef{
		Name:         p.Name,
		Version:      p.Version,
		ProjectURLs:  urls,
		HomePage:     p.HomePage,
		ManifestFile: p.ManifestFile,
	}
}
