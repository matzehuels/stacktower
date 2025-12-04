package metadata

import (
	"context"
	"maps"

	"github.com/matzehuels/stacktower/pkg/source"
)

type Composite struct {
	providers []source.MetadataProvider
}

func NewComposite(providers ...source.MetadataProvider) *Composite {
	return &Composite{providers}
}

func (c *Composite) Name() string { return "composite" }

func (c *Composite) Enrich(ctx context.Context, repo *source.RepoInfo, refresh bool) (map[string]any, error) {
	m := make(map[string]any)
	for _, p := range c.providers {
		if meta, err := p.Enrich(ctx, repo, refresh); err == nil {
			maps.Copy(m, meta)
		}
	}
	return m, nil
}
