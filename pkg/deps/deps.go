package deps

import (
	"context"
	"maps"
	"time"
)

const (
	DefaultMaxDepth = 50
	DefaultMaxNodes = 5000
	DefaultCacheTTL = 24 * time.Hour
)

type Options struct {
	MaxDepth          int
	MaxNodes          int
	CacheTTL          time.Duration
	Refresh           bool
	MetadataProviders []MetadataProvider
	Logger            func(string, ...any)
}

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

type MetadataProvider interface {
	Name() string
	Enrich(ctx context.Context, pkg *PackageRef, refresh bool) (map[string]any, error)
}

type PackageRef struct {
	Name         string
	Version      string
	ProjectURLs  map[string]string
	HomePage     string
	ManifestFile string
}

type Package struct {
	Name         string
	Version      string
	Dependencies []string
	Description  string
	License      string
	Author       string
	Downloads    int
	Repository   string
	HomePage     string
	ProjectURLs  map[string]string
	ManifestFile string
}

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
