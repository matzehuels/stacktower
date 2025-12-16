package ruby

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/deps"
	"github.com/matzehuels/stacktower/pkg/integrations/rubygems"
)

var Language = &deps.Language{
	Name:            "ruby",
	DefaultRegistry: "rubygems",
	RegistryAliases: map[string]string{"gems": "rubygems"},
	ManifestTypes:   []string{"gemfile"},
	ManifestAliases: map[string]string{"Gemfile": "gemfile"},
	NewResolver:     newResolver,
	NewManifest:     newManifest,
	ManifestParsers: manifestParsers,
}

func newManifest(name string, res deps.Resolver) deps.ManifestParser {
	switch name {
	case "gemfile":
		return &Gemfile{resolver: res}
	default:
		return nil
	}
}

func manifestParsers(res deps.Resolver) []deps.ManifestParser {
	return []deps.ManifestParser{&Gemfile{resolver: res}}
}

func newResolver(ttl time.Duration) (deps.Resolver, error) {
	c, err := rubygems.NewClient(ttl)
	if err != nil {
		return nil, err
	}
	return deps.NewRegistry("rubygems", fetcher{c}), nil
}

type fetcher struct{ *rubygems.Client }

func (f fetcher) Fetch(ctx context.Context, name string, refresh bool) (*deps.Package, error) {
	g, err := f.FetchGem(ctx, name, refresh)
	if err != nil {
		return nil, err
	}
	return &deps.Package{
		Name:         g.Name,
		Version:      g.Version,
		Dependencies: g.Dependencies,
		Description:  g.Description,
		License:      g.License,
		Author:       g.Authors,
		Downloads:    g.Downloads,
		Repository:   g.SourceCodeURI,
		HomePage:     g.HomepageURI,
		ManifestFile: "Gemfile",
	}, nil
}
