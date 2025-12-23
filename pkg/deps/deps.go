package deps

import (
	"context"
	"maps"
	"time"
)

const (
	// DefaultMaxDepth is the default maximum dependency depth (50 levels).
	// This prevents infinite recursion in circular or very deep dependency trees.
	DefaultMaxDepth = 50

	// DefaultMaxNodes is the default maximum number of packages to fetch (5000 nodes).
	// This caps memory usage and prevents unbounded crawling of large ecosystems.
	DefaultMaxNodes = 5000

	// DefaultCacheTTL is the default HTTP cache duration (24 hours).
	// Cached registry responses are reused within this window unless Refresh is true.
	DefaultCacheTTL = 24 * time.Hour
)

// Options configures dependency resolution behavior.
//
// All fields are optional. Zero values are replaced by defaults when passed
// to WithDefaults. Options is safe to copy and does not modify any inputs.
type Options struct {
	// MaxDepth limits how many levels deep to traverse. A value of 1 fetches
	// only direct dependencies. Zero or negative values use DefaultMaxDepth (50).
	MaxDepth int

	// MaxNodes limits the total number of packages to fetch. When this limit
	// is reached, deeper dependencies are ignored but already-queued packages
	// may still be fetched. Zero or negative values use DefaultMaxNodes (5000).
	MaxNodes int

	// CacheTTL controls how long HTTP responses are cached. Registry clients
	// will reuse cached data within this duration. Zero or negative values use
	// DefaultCacheTTL (24 hours).
	CacheTTL time.Duration

	// Refresh bypasses the HTTP cache when true, forcing fresh registry fetches.
	// This is useful for getting the latest package versions but increases latency.
	Refresh bool

	// MetadataProviders is an optional list of enrichment sources (e.g., GitHub)
	// that add extra metadata to package nodes. Providers are called concurrently
	// after fetching each package. Nil or empty is safe.
	MetadataProviders []MetadataProvider

	// Logger is an optional callback for progress and error messages. If nil,
	// WithDefaults replaces it with a no-op logger. The format string follows
	// fmt.Printf conventions. Logger may be called concurrently from multiple
	// goroutines and must be safe for concurrent use.
	Logger func(string, ...any)
}

// WithDefaults returns a copy of Options with zero values replaced by defaults.
//
// This method is safe to call on a zero Options value. It fills in:
//   - MaxDepth: DefaultMaxDepth (50)
//   - MaxNodes: DefaultMaxNodes (5000)
//   - CacheTTL: DefaultCacheTTL (24h)
//   - Logger: no-op function if nil
//
// All other fields (Refresh, MetadataProviders) are preserved as-is, including
// nil slices. The original Options value is not modified.
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
//
// Implementations fetch supplementary information that is not available in package
// registries, such as repository activity, maintainer counts, or security metrics.
// Providers are called concurrently after fetching each package during resolution.
type MetadataProvider interface {
	// Name returns the provider identifier (e.g., "github", "gitlab").
	// This is used for logging and error messages.
	Name() string

	// Enrich fetches additional metadata for the package.
	//
	// The pkg parameter contains registry information and URLs for lookup.
	// If refresh is true, the provider should bypass its cache.
	//
	// Returns a map of metadata keys to values, which are merged into the
	// package node's metadata. Keys should be provider-specific (e.g.,
	// "github_stars") to avoid conflicts with other providers.
	//
	// Returns an error if enrichment fails. The resolver logs the error
	// but continues without failing the entire resolution.
	Enrich(ctx context.Context, pkg *PackageRef, refresh bool) (map[string]any, error)
}

// PackageRef identifies a package for metadata enrichment lookups.
//
// It contains the information metadata providers need to look up external data
// like GitHub repository statistics. Created by [Package.Ref].
type PackageRef struct {
	// Name is the package name as it appears in the registry.
	Name string

	// Version is the specific version being referenced.
	Version string

	// ProjectURLs contains URL mappings from the package registry, typically
	// including "repository", "homepage", "documentation", etc. The keys
	// depend on the registry. May be nil or empty.
	ProjectURLs map[string]string

	// HomePage is the project's home page URL, if available. May be empty.
	HomePage string

	// ManifestFile is the associated manifest type (e.g., "poetry", "cargo")
	// when the package comes from manifest parsing. Empty for registry-only packages.
	ManifestFile string
}

// Package holds metadata fetched from a package registry.
//
// This is the core data structure returned by [Fetcher.Fetch] and used throughout
// the resolution process. Not all fields are populated by every registry—consult
// the specific integration documentation for field availability.
type Package struct {
	// Name is the package identifier in the registry (e.g., "requests", "serde").
	Name string

	// Version is the package version (e.g., "2.31.0"). For registry lookups
	// without a version constraint, this is typically the latest stable version.
	Version string

	// Dependencies lists direct dependency names. The resolver recursively fetches
	// these to build the dependency tree. Nil and empty slices are equivalent.
	Dependencies []string

	// Description is a short summary of the package purpose. May be empty.
	Description string

	// License is the package license identifier (e.g., "MIT", "Apache-2.0").
	// May be empty or unknown.
	License string

	// Author is the primary package author or maintainer. May be empty.
	Author string

	// Downloads is the total download count or recent download rate, depending
	// on the registry. Zero if unavailable. Not all registries provide this.
	Downloads int

	// Repository is the source code repository URL (e.g., GitHub, GitLab).
	// May be empty if not specified in registry metadata.
	Repository string

	// HomePage is the project home page URL. May be empty or identical to Repository.
	HomePage string

	// ProjectURLs contains additional URLs from the registry (docs, issues, etc.).
	// Keys and availability vary by registry. May be nil.
	ProjectURLs map[string]string

	// ManifestFile identifies the manifest type when this Package comes from
	// manifest parsing (e.g., "poetry", "cargo"). Empty for registry packages.
	ManifestFile string
}

// Metadata converts Package fields to a map for node metadata.
//
// The returned map always contains "version". Optional fields (description,
// license, author, downloads) are included only if non-empty/non-zero.
//
// This map is suitable for use as dag.Node.Meta and can be further enriched
// by [MetadataProvider] implementations. The map is newly allocated and safe
// to modify. Returns a non-nil map even if the Package has no optional fields.
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
//
// The returned PackageRef consolidates URL information from multiple Package
// fields (ProjectURLs, Repository, HomePage) into a single ProjectURLs map
// for convenient provider lookups.
//
// The ProjectURLs map is a clone of the original, so modifying it does not
// affect the Package. If the Package has nil ProjectURLs, an empty map is
// allocated. Repository and HomePage are added to the map under "repository"
// and "homepage" keys if non-empty.
//
// This method never returns nil. Safe to call on a zero Package value.
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
