package golang

import (
	"context"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/cache"
	"github.com/matzehuels/stacktower/pkg/core/deps"
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

func newResolver(backend cache.Cache, ttl time.Duration) (deps.Resolver, error) {
	c := goproxy.NewClient(backend, ttl)
	return deps.NewRegistry("goproxy", fetcher{c}), nil
}

type fetcher struct{ *goproxy.Client }

func (f fetcher) Fetch(ctx context.Context, name string, refresh bool) (*deps.Package, error) {
	m, err := f.FetchModule(ctx, name, refresh)
	if err != nil {
		return nil, err
	}
	pkg := &deps.Package{
		Name:         m.Path,
		Version:      m.Version,
		Dependencies: m.Dependencies,
		ManifestFile: "go.mod",
	}

	// For github.com modules, extract repository URL from module path
	// e.g., github.com/spf13/cobra → https://github.com/spf13/cobra
	pkg.Repository = inferRepoURL(m.Path)

	return pkg, nil
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

// inferRepoURL extracts the repository URL from a Go module path.
// For github.com, gitlab.com, and bitbucket.org modules, it converts
// the module path to an HTTPS URL by taking the first two path segments.
//
// Examples:
//   - github.com/spf13/cobra → https://github.com/spf13/cobra
//   - github.com/gofiber/fiber/v2 → https://github.com/gofiber/fiber
//   - gitlab.com/user/repo → https://gitlab.com/user/repo
//   - gopkg.in/yaml.v3 → (returns empty string)
//
// Returns an empty string for non-repository-based modules or modules
// from unsupported hosting platforms.
func inferRepoURL(modulePath string) string {
	// Common hosting platforms that use path-based module names
	for _, prefix := range []string{"github.com/", "gitlab.com/", "bitbucket.org/"} {
		if strings.HasPrefix(modulePath, prefix) {
			// Extract owner/repo (first two path segments after the domain)
			// e.g., "github.com/spf13/cobra/doc" → owner="spf13", repo="cobra"
			parts := strings.Split(strings.TrimPrefix(modulePath, prefix), "/")
			if len(parts) >= 2 {
				return "https://" + prefix + parts[0] + "/" + parts[1]
			}
		}
	}
	return ""
}
