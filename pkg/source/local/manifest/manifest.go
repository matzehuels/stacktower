package manifest

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/integrations/npm"
	"github.com/matzehuels/stacktower/pkg/source"
)

type Parser struct {
	registryClient *npm.Client
	manifestPath   string
	parsedRoot     *packageInfo
}

func NewParser(cacheTTL time.Duration) (*Parser, error) {
	c, err := npm.NewClient(cacheTTL)
	if err != nil {
		return nil, err
	}
	return &Parser{registryClient: c}, nil
}

func (p *Parser) Parse(ctx context.Context, manifestPath string, opts source.Options) (*dag.DAG, error) {
	// 1. Parse manifest for root package name + direct deps
	root, err := parsePackageJSON(manifestPath)
	if err != nil {
		return nil, err
	}
	p.parsedRoot = root

	// 2. Create a fetch function that returns manifest data for root,
	//    but delegates to registry for everything else
	return source.Parse(ctx, root.Name, opts, func(ctx context.Context, name string, refresh bool) (*packageInfo, error) {
		if name == root.Name {
			return root, nil
		}
		// For transitive dependencies, fetch from registry
		registryInfo, err := p.registryClient.FetchPackage(ctx, name, refresh)
		if err != nil {
			return nil, err
		}
		return &packageInfo{registryInfo}, nil
	})
}

type packageInfo struct {
	*npm.PackageInfo
}

func (pi *packageInfo) GetName() string           { return pi.Name }
func (pi *packageInfo) GetVersion() string        { return pi.Version }
func (pi *packageInfo) GetDependencies() []string { return pi.Dependencies }

func (pi *packageInfo) ToMetadata() map[string]any {
	m := map[string]any{"version": pi.Version}
	if pi.Description != "" {
		m["description"] = pi.Description
	}
	if pi.License != "" {
		m["license"] = pi.License
	}
	if pi.Author != "" {
		m["author"] = pi.Author
	}
	return m
}

func (pi *packageInfo) ToRepoInfo() *source.RepoInfo {
	urls := make(map[string]string, 2)
	if pi.Repository != "" {
		urls["repository"] = pi.Repository
	}
	if pi.HomePage != "" {
		urls["homepage"] = pi.HomePage
	}
	return &source.RepoInfo{
		Name:         pi.Name,
		Version:      pi.Version,
		ProjectURLs:  urls,
		HomePage:     pi.HomePage,
		ManifestFile: "package.json",
	}
}

// parsePackageJSON parses a package.json file and returns the package info
func parsePackageJSON(path string) (*packageInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Name         string            `json:"name"`
		Version      string            `json:"version"`
		Dependencies map[string]string `json:"dependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	// Extract dependency names
	deps := make([]string, 0, len(pkg.Dependencies))
	for name := range pkg.Dependencies {
		deps = append(deps, name)
	}

	return &packageInfo{
		PackageInfo: &npm.PackageInfo{
			Name:         pkg.Name,
			Version:      pkg.Version,
			Dependencies: deps,
		},
	}, nil
}

