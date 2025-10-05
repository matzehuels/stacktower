package rust

import (
	"context"
	"time"

	"stacktower/pkg/dag"
	"stacktower/pkg/integrations/crates"
	"stacktower/pkg/source"
)

type Parser struct {
	client *crates.Client
}

func NewParser(cacheTTL time.Duration) (*Parser, error) {
	c, err := crates.NewClient(cacheTTL)
	if err != nil {
		return nil, err
	}
	return &Parser{client: c}, nil
}

func (p *Parser) Parse(ctx context.Context, crate string, opts source.Options) (*dag.DAG, error) {
	return source.Parse(ctx, crate, opts, p.fetch)
}

func (p *Parser) fetch(ctx context.Context, name string, refresh bool) (*crateInfo, error) {
	info, err := p.client.FetchCrate(ctx, name, refresh)
	if err != nil {
		return nil, err
	}
	return &crateInfo{info}, nil
}

type crateInfo struct {
	*crates.CrateInfo
}

func (ci *crateInfo) GetName() string           { return ci.Name }
func (ci *crateInfo) GetVersion() string        { return ci.Version }
func (ci *crateInfo) GetDependencies() []string { return ci.Dependencies }

func (ci *crateInfo) ToMetadata() map[string]any {
	m := map[string]any{"version": ci.Version}
	if ci.Description != "" {
		m["description"] = ci.Description
	}
	if ci.License != "" {
		m["license"] = ci.License
	}
	if ci.Downloads > 0 {
		m["downloads"] = ci.Downloads
	}
	return m
}

func (ci *crateInfo) ToRepoInfo() *source.RepoInfo {
	urls := make(map[string]string, 2)
	if ci.Repository != "" {
		urls["repository"] = ci.Repository
	}
	if ci.HomePage != "" {
		urls["homepage"] = ci.HomePage
	}
	return &source.RepoInfo{
		Name:         ci.Name,
		Version:      ci.Version,
		ProjectURLs:  urls,
		HomePage:     ci.HomePage,
		ManifestFile: "Cargo.toml",
	}
}
