package python

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/deps"
	"github.com/matzehuels/stacktower/pkg/integrations"
	"github.com/matzehuels/stacktower/pkg/integrations/pypi"
)

// Language provides Python dependency resolution via PyPI.
// Supports poetry.lock and requirements.txt manifest files.
var Language = &deps.Language{
	Name:            "python",
	DefaultRegistry: "pypi",
	ManifestTypes:   []string{"poetry", "requirements"},
	ManifestAliases: map[string]string{
		"poetry.lock":      "poetry",
		"requirements.txt": "requirements",
	},
	NewResolver:     newResolver,
	NewManifest:     newManifest,
	ManifestParsers: manifestParsers,
}

func newResolver(ttl time.Duration) (deps.Resolver, error) {
	c, err := pypi.NewClient(ttl)
	if err != nil {
		return nil, err
	}
	return deps.NewRegistry("pypi", fetcher{c}), nil
}

type fetcher struct{ *pypi.Client }

func (f fetcher) Fetch(ctx context.Context, name string, refresh bool) (*deps.Package, error) {
	p, err := f.FetchPackage(ctx, name, refresh)
	if err != nil {
		return nil, err
	}
	return &deps.Package{
		Name:         p.Name,
		Version:      p.Version,
		Dependencies: p.Dependencies,
		Description:  p.Summary,
		License:      p.License,
		Author:       p.Author,
		HomePage:     p.HomePage,
		ProjectURLs:  p.ProjectURLs,
		ManifestFile: "pyproject.toml",
	}, nil
}

func newManifest(name string, res deps.Resolver) deps.ManifestParser {
	switch name {
	case "poetry":
		return &PoetryLock{}
	case "requirements":
		return &Requirements{resolver: res}
	default:
		return nil
	}
}

func manifestParsers(res deps.Resolver) []deps.ManifestParser {
	return []deps.ManifestParser{
		&Requirements{resolver: res},
		&PoetryLock{},
	}
}

func normalize(name string) string {
	return integrations.NormalizePkgName(name)
}
