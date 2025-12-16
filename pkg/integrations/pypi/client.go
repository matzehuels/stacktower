package pypi

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

var (
	depRE    = regexp.MustCompile(`^([a-zA-Z0-9_-]+)`)
	markerRE = regexp.MustCompile(`;\s*(.+)`)
	skipRE   = regexp.MustCompile(`extra|dev|test`)
)

type PackageInfo struct {
	Name         string
	Version      string
	Dependencies []string
	ProjectURLs  map[string]string
	HomePage     string
	Summary      string
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
		baseURL: "https://pypi.org/pypi",
	}, nil
}

func (c *Client) FetchPackage(ctx context.Context, pkg string, refresh bool) (*PackageInfo, error) {
	pkg = integrations.NormalizePkgName(pkg)
	key := "pypi:" + pkg

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
	var data apiResponse
	if err := c.Get(ctx, fmt.Sprintf("%s/%s/json", c.baseURL, pkg), &data); err != nil {
		if errors.Is(err, integrations.ErrNotFound) {
			return fmt.Errorf("%w: pypi package %s", err, pkg)
		}
		return err
	}

	urls := make(map[string]string, len(data.Info.ProjectURLs))
	for k, v := range data.Info.ProjectURLs {
		if s, ok := v.(string); ok {
			urls[k] = s
		}
	}

	*info = PackageInfo{
		Name:         data.Info.Name,
		Version:      data.Info.Version,
		Summary:      data.Info.Summary,
		License:      data.Info.License,
		Dependencies: extractDeps(data.Info.RequiresDist),
		ProjectURLs:  urls,
		HomePage:     data.Info.HomePage,
		Author:       data.Info.Author,
	}
	return nil
}

func extractDeps(requires []string) []string {
	seen := make(map[string]bool)
	var deps []string
	for _, req := range requires {
		if m := markerRE.FindStringSubmatch(req); len(m) > 1 && skipRE.MatchString(m[1]) {
			continue
		}
		if m := depRE.FindStringSubmatch(req); len(m) > 1 {
			dep := integrations.NormalizePkgName(m[1])
			if !seen[dep] {
				seen[dep] = true
				deps = append(deps, dep)
			}
		}
	}
	return deps
}

type apiResponse struct {
	Info apiInfo `json:"info"`
}

type apiInfo struct {
	Name         string         `json:"name"`
	Version      string         `json:"version"`
	Summary      string         `json:"summary"`
	License      string         `json:"license"`
	RequiresDist []string       `json:"requires_dist"`
	ProjectURLs  map[string]any `json:"project_urls"`
	HomePage     string         `json:"home_page"`
	Author       string         `json:"author"`
}
