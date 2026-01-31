package php

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/cache"
	"github.com/matzehuels/stacktower/pkg/core/deps"
	"github.com/matzehuels/stacktower/pkg/integrations/packagist"
)

// Language provides PHP dependency resolution via Packagist.
// Supports composer.json manifest files.
var Language = &deps.Language{
	Name:            "php",
	DefaultRegistry: "packagist",
	ManifestTypes:   []string{"composer"},
	ManifestAliases: map[string]string{"composer.json": "composer"},
	NewResolver:     newResolver,
	NewManifest:     newManifest,
	ManifestParsers: manifestParsers,
}

func newManifest(name string, res deps.Resolver) deps.ManifestParser {
	switch name {
	case "composer":
		return &ComposerJSON{resolver: res}
	default:
		return nil
	}
}

func manifestParsers(res deps.Resolver) []deps.ManifestParser {
	return []deps.ManifestParser{&ComposerJSON{resolver: res}}
}

func newResolver(backend cache.Cache, ttl time.Duration) (deps.Resolver, error) {
	c := packagist.NewClient(backend, ttl)
	return deps.NewRegistry("packagist", fetcher{c}), nil
}

type fetcher struct{ *packagist.Client }

func (f fetcher) Fetch(ctx context.Context, name string, refresh bool) (*deps.Package, error) {
	p, err := f.FetchPackage(ctx, name, refresh)
	if err != nil {
		return nil, err
	}
	return &deps.Package{
		Name:         p.Name,
		Version:      p.Version,
		Dependencies: p.Dependencies,
		Description:  p.Description,
		License:      p.License,
		Author:       p.Author,
		Repository:   p.Repository,
		HomePage:     p.HomePage,
		ManifestFile: "composer.json",
	}, nil
}
