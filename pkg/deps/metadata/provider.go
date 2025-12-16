package metadata

import (
	"context"
	"maps"

	"github.com/matzehuels/stacktower/pkg/deps"
)

type Composite struct {
	providers []deps.MetadataProvider
}

func NewComposite(providers ...deps.MetadataProvider) *Composite {
	return &Composite{providers}
}

func (c *Composite) Name() string { return "composite" }

func (c *Composite) Enrich(ctx context.Context, pkg *deps.PackageRef, refresh bool) (map[string]any, error) {
	m := make(map[string]any)
	for _, p := range c.providers {
		if meta, err := p.Enrich(ctx, pkg, refresh); err == nil {
			maps.Copy(m, meta)
		}
	}
	return m, nil
}
