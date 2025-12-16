package rust

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/deps"
	"github.com/matzehuels/stacktower/pkg/integrations/crates"
)

var Language = &deps.Language{
	Name:            "rust",
	DefaultRegistry: "crates",
	RegistryAliases: map[string]string{"crates.io": "crates"},
	ManifestTypes:   []string{"cargo"},
	ManifestAliases: map[string]string{"Cargo.toml": "cargo", "cargo.toml": "cargo"},
	NewResolver:     newResolver,
	NewManifest:     newManifest,
	ManifestParsers: manifestParsers,
}

func newManifest(name string, res deps.Resolver) deps.ManifestParser {
	switch name {
	case "cargo":
		return &CargoToml{resolver: res}
	default:
		return nil
	}
}

func manifestParsers(res deps.Resolver) []deps.ManifestParser {
	return []deps.ManifestParser{&CargoToml{resolver: res}}
}

func newResolver(ttl time.Duration) (deps.Resolver, error) {
	c, err := crates.NewClient(ttl)
	if err != nil {
		return nil, err
	}
	return deps.NewRegistry("crates.io", fetcher{c}), nil
}

type fetcher struct{ *crates.Client }

func (f fetcher) Fetch(ctx context.Context, name string, refresh bool) (*deps.Package, error) {
	cr, err := f.FetchCrate(ctx, name, refresh)
	if err != nil {
		return nil, err
	}
	return &deps.Package{
		Name:         cr.Name,
		Version:      cr.Version,
		Dependencies: cr.Dependencies,
		Description:  cr.Description,
		License:      cr.License,
		Downloads:    cr.Downloads,
		Repository:   cr.Repository,
		HomePage:     cr.HomePage,
		ManifestFile: "Cargo.toml",
	}, nil
}
