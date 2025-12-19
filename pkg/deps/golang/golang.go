package golang

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/deps"
	"github.com/matzehuels/stacktower/pkg/integrations/goproxy"
)

// Language provides Go dependency resolution via the Go module proxy.
// Supports go.mod manifest files.
var Language = &deps.Language{
	Name:            "go",
	DefaultRegistry: "goproxy",
	RegistryAliases: map[string]string{"proxy": "goproxy", "go": "goproxy"},
	ManifestTypes:   []string{"gomod"},
	ManifestAliases: map[string]string{"go.mod": "gomod"},
	NewResolver:     newResolver,
	NewManifest:     newManifest,
	ManifestParsers: manifestParsers,
}

func newResolver(ttl time.Duration) (deps.Resolver, error) {
	c, err := goproxy.NewClient(ttl)
	if err != nil {
		return nil, err
	}
	return deps.NewRegistry("goproxy", fetcher{c}), nil
}

type fetcher struct{ *goproxy.Client }

func (f fetcher) Fetch(ctx context.Context, name string, refresh bool) (*deps.Package, error) {
	m, err := f.FetchModule(ctx, name, refresh)
	if err != nil {
		return nil, err
	}
	return &deps.Package{
		Name:         m.Path,
		Version:      m.Version,
		Dependencies: m.Dependencies,
		ManifestFile: "go.mod",
	}, nil
}

func newManifest(name string, res deps.Resolver) deps.ManifestParser {
	switch name {
	case "gomod":
		return &GoModParser{resolver: res}
	default:
		return nil
	}
}

func manifestParsers(res deps.Resolver) []deps.ManifestParser {
	return []deps.ManifestParser{
		&GoModParser{resolver: res},
	}
}
