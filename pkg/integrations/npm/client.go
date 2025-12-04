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
	integrations.BaseClient
	baseURL string
}

func NewClient(cacheTTL time.Duration) (*Client, error) {
	cache, err := integrations.NewCache(cacheTTL)
	if err != nil {
		return nil, err
	}
	return &Client{
		BaseClient: integrations.BaseClient{
			HTTP:  integrations.NewHTTPClient(),
			Cache: cache,
		},
		baseURL: "https://registry.npmjs.org",
	}, nil
}

func (c *Client) FetchPackage(ctx context.Context, pkg string, refresh bool) (*PackageInfo, error) {
	pkg = normalizeName(pkg)
	cacheKey := "npm:" + pkg

	var info PackageInfo
	err := c.FetchWithCache(ctx, cacheKey, refresh, func() error {
		return c.fetchPackage(ctx, pkg, &info)
	}, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *Client) fetchPackage(ctx context.Context, pkg string, info *PackageInfo) error {
	var data registryResponse
	if err := c.DoRequest(ctx, c.baseURL+"/"+pkg, nil, &data); err != nil {
		if errors.Is(err, integrations.ErrNotFound) {
			return fmt.Errorf("%w: npm package %s", err, pkg)
		}
		return err
	}

	v := data.DistTags.Latest
	vd, ok := data.Versions[v]
	if !ok {
		return fmt.Errorf("version %s not found in registry data", v)
	}

	*info = PackageInfo{
		Name:         data.Name,
		Version:      v,
		Description:  vd.Description,
		License:      extractString(vd.License, "type"),
		Author:       extractString(vd.Author, "name"),
		Repository:   normalizeRepoURL(extractString(vd.Repository, "url")),
		HomePage:     vd.HomePage,
		Dependencies: slices.Collect(maps.Keys(vd.Dependencies)),
	}
	return nil
}

func extractString(v any, field string) string {
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

func normalizeRepoURL(url string) string {
	if url == "" {
		return ""
	}
	url = strings.TrimSpace(url)
	url = strings.TrimPrefix(url, "git+")
	url = strings.ReplaceAll(url, "git@github.com:", "https://github.com/")
	url = strings.ReplaceAll(url, "git://github.com/", "https://github.com/")
	return strings.TrimSuffix(url, ".git")
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
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
