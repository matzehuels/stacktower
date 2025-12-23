package java

import (
	"context"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/deps"
	"github.com/matzehuels/stacktower/pkg/integrations/maven"
)

// Language provides Java dependency resolution via Maven Central.
// Supports pom.xml manifest files.
var Language = &deps.Language{
	Name:            "java",
	DefaultRegistry: "maven",
	RegistryAliases: map[string]string{"maven-central": "maven", "mvn": "maven"},
	ManifestTypes:   []string{"pom"},
	ManifestAliases: map[string]string{"pom.xml": "pom"},
	NewResolver:     newResolver,
	NewManifest:     newManifest,
	ManifestParsers: manifestParsers,
	NormalizeName:   NormalizeCoordinate,
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
	coord := NormalizeCoordinate(name)
	a, err := f.FetchArtifact(ctx, coord, refresh)
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

// NormalizeCoordinate converts filename-safe coordinates to Maven format.
// Since colons are not allowed in filenames (especially on Windows and in some
// build tools), underscores can be used as a substitute. This function converts
// "groupId_artifactId" to "groupId:artifactId" when no colon is present.
//
// Examples:
//   - "com.google.guava:guava" → "com.google.guava:guava" (unchanged)
//   - "com.google.guava_guava" → "com.google.guava:guava" (converted)
func NormalizeCoordinate(coord string) string {
	if strings.Contains(coord, ":") {
		return coord
	}
	// Replace the last underscore with a colon
	// GroupIds follow reverse domain notation (no underscores typically)
	// while artifactIds may contain hyphens or underscores
	if idx := strings.LastIndex(coord, "_"); idx != -1 {
		return coord[:idx] + ":" + coord[idx+1:]
	}
	return coord
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
