package javascript

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/deps"
	"github.com/matzehuels/stacktower/pkg/integrations/npm"
)

// Language provides JavaScript/TypeScript dependency resolution via npm.
// Supports package.json manifest files.
var Language = &deps.Language{
	Name:            "javascript",
	DefaultRegistry: "npm",
	ManifestTypes:   []string{"package"},
	ManifestAliases: map[string]string{"package.json": "package"},
	NewResolver:     newResolver,
	NewManifest:     newManifest,
	ManifestParsers: manifestParsers,
}

func newManifest(name string, res deps.Resolver) deps.ManifestParser {
	switch name {
	case "package":
		return &PackageJSON{resolver: res}
	default:
		return nil
	}
}

func manifestParsers(res deps.Resolver) []deps.ManifestParser {
	return []deps.ManifestParser{&PackageJSON{resolver: res}}
}

func newResolver(ttl time.Duration) (deps.Resolver, error) {
	c, err := npm.NewClient(ttl)
	if err != nil {
		return nil, err
	}
	return deps.NewRegistry("npm", fetcher{c}), nil
}

type fetcher struct{ *npm.Client }

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
		ManifestFile: "package.json",
	}, nil
}
