package java

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/deps"
	"github.com/matzehuels/stacktower/pkg/integrations/maven"
)

var Language = &deps.Language{
	Name:            "java",
	DefaultRegistry: "maven",
	RegistryAliases: map[string]string{"maven-central": "maven", "mvn": "maven"},
	ManifestTypes:   []string{"pom"},
	ManifestAliases: map[string]string{"pom.xml": "pom"},
	NewResolver:     newResolver,
	NewManifest:     newManifest,
	ManifestParsers: manifestParsers,
}

func newResolver(ttl time.Duration) (deps.Resolver, error) {
	c, err := maven.NewClient(ttl)
	if err != nil {
		return nil, err
	}
	return deps.NewRegistry("maven", fetcher{c}), nil
}

type fetcher struct{ *maven.Client }

func (f fetcher) Fetch(ctx context.Context, name string, refresh bool) (*deps.Package, error) {
	a, err := f.FetchArtifact(ctx, name, refresh)
	if err != nil {
		return nil, err
	}
	return &deps.Package{
		Name:         a.Coordinate(),
		Version:      a.Version,
		Dependencies: a.Dependencies,
		Description:  a.Description,
		HomePage:     a.URL,
		ManifestFile: "pom.xml",
	}, nil
}

func newManifest(name string, res deps.Resolver) deps.ManifestParser {
	switch name {
	case "pom":
		return &POMParser{resolver: res}
	default:
		return nil
	}
}

func manifestParsers(res deps.Resolver) []deps.ManifestParser {
	return []deps.ManifestParser{
		&POMParser{resolver: res},
	}
}
