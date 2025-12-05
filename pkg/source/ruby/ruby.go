package ruby

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/integrations/rubygems"
	"github.com/matzehuels/stacktower/pkg/source"
)

type Parser struct {
	client *rubygems.Client
}

func NewParser(cacheTTL time.Duration) (*Parser, error) {
	c, err := rubygems.NewClient(cacheTTL)
	if err != nil {
		return nil, err
	}
	return &Parser{client: c}, nil
}

func (p *Parser) Parse(ctx context.Context, gem string, opts source.Options) (*dag.DAG, error) {
	return source.Parse(ctx, gem, opts, p.fetch)
}

func (p *Parser) fetch(ctx context.Context, name string, refresh bool) (*gemInfo, error) {
	info, err := p.client.FetchGem(ctx, name, refresh)
	if err != nil {
		return nil, err
	}
	return &gemInfo{info}, nil
}

type gemInfo struct {
	*rubygems.GemInfo
}

func (gi *gemInfo) GetName() string           { return gi.Name }
func (gi *gemInfo) GetVersion() string        { return gi.Version }
func (gi *gemInfo) GetDependencies() []string { return gi.Dependencies }

func (gi *gemInfo) ToMetadata() map[string]any {
	m := map[string]any{"version": gi.Version}
	if gi.Description != "" {
		m["description"] = gi.Description
	}
	if gi.License != "" {
		m["license"] = gi.License
	}
	if gi.Authors != "" {
		m["author"] = gi.Authors
	}
	if gi.Downloads > 0 {
		m["downloads"] = gi.Downloads
	}
	return m
}

func (gi *gemInfo) ToRepoInfo() *source.RepoInfo {
	urls := make(map[string]string, 2)
	if gi.SourceCodeURI != "" {
		urls["repository"] = gi.SourceCodeURI
	}
	if gi.HomepageURI != "" {
		urls["homepage"] = gi.HomepageURI
	}
	return &source.RepoInfo{
		Name:         gi.Name,
		Version:      gi.Version,
		ProjectURLs:  urls,
		HomePage:     gi.HomepageURI,
		ManifestFile: "Gemfile",
	}
}

