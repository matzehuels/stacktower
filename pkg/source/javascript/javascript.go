package javascript

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/integrations/npm"
	"github.com/matzehuels/stacktower/pkg/source"
)

type Parser struct {
	client *npm.Client
}

func NewParser(cacheTTL time.Duration) (*Parser, error) {
	c, err := npm.NewClient(cacheTTL)
	if err != nil {
		return nil, err
	}
	return &Parser{client: c}, nil
}

func (p *Parser) Parse(ctx context.Context, pkg string, opts source.Options) (*dag.DAG, error) {
	return source.Parse(ctx, pkg, opts, p.fetch)
}

func (p *Parser) fetch(ctx context.Context, name string, refresh bool) (*packageInfo, error) {
	info, err := p.client.FetchPackage(ctx, name, refresh)
	if err != nil {
		return nil, err
	}
	return &packageInfo{info}, nil
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
