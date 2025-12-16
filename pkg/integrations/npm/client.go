package npm

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

type PackageInfo struct {
	Name         string
	Version      string
	Dependencies []string
	Repository   string
	HomePage     string
	Description  string
	License      string
	Author       string
}

type Client struct {
	*integrations.Client
	baseURL string
}

func NewClient(cacheTTL time.Duration) (*Client, error) {
	cache, err := integrations.NewCache(cacheTTL)
	if err != nil {
		return nil, err
	}
	return &Client{
		Client:  integrations.NewClient(cache, nil),
		baseURL: "https://registry.npmjs.org",
	}, nil
}

func (c *Client) FetchPackage(ctx context.Context, pkg string, refresh bool) (*PackageInfo, error) {
	pkg = strings.ToLower(strings.TrimSpace(pkg))
	key := "npm:" + pkg

	var info PackageInfo
	err := c.Cached(ctx, key, refresh, &info, func() error {
		return c.fetch(ctx, pkg, &info)
	})
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *Client) fetch(ctx context.Context, pkg string, info *PackageInfo) error {
	var data registryResponse
	if err := c.Get(ctx, c.baseURL+"/"+pkg, &data); err != nil {
		if errors.Is(err, integrations.ErrNotFound) {
			return fmt.Errorf("%w: npm package %s", err, pkg)
		}
		return err
	}

	latest := data.DistTags.Latest
	v, ok := data.Versions[latest]
	if !ok {
		return fmt.Errorf("version %s not found", latest)
	}

	*info = PackageInfo{
		Name:         data.Name,
		Version:      latest,
		Description:  v.Description,
		License:      extractField(v.License, "type"),
		Author:       extractField(v.Author, "name"),
		Repository:   integrations.NormalizeRepoURL(extractField(v.Repository, "url")),
		HomePage:     v.HomePage,
		Dependencies: slices.Collect(maps.Keys(v.Dependencies)),
	}
	return nil
}

func extractField(v any, field string) string {
	switch val := v.(type) {
	case string:
		return val
	case map[string]any:
		if s, ok := val[field].(string); ok {
			return s
		}
	}
	return ""
}

type registryResponse struct {
	Name     string                    `json:"name"`
	DistTags distTags                  `json:"dist-tags"`
	Versions map[string]versionDetails `json:"versions"`
}

type distTags struct {
	Latest string `json:"latest"`
}

type versionDetails struct {
	Description  string            `json:"description"`
	License      any               `json:"license"`
	Author       any               `json:"author"`
	Repository   any               `json:"repository"`
	HomePage     string            `json:"homepage"`
	Dependencies map[string]string `json:"dependencies"`
}
