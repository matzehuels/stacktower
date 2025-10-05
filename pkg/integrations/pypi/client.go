package pypi

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"stacktower/pkg/integrations"
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
		baseURL: "https://pypi.org/pypi",
	}, nil
}

func (c *Client) FetchPackage(ctx context.Context, pkg string, refresh bool) (*PackageInfo, error) {
	pkg = normalizeName(pkg)
	cacheKey := "pypi:" + pkg

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
	url := fmt.Sprintf("%s/%s/json", c.baseURL, pkg)

	var data apiResponse
	if err := c.DoRequest(ctx, url, nil, &data); err != nil {
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

func extractDeps(requiresDist []string) []string {
	seen := make(map[string]bool)
	var deps []string

	for _, req := range requiresDist {
		if m := markerRE.FindStringSubmatch(req); len(m) > 1 && skipRE.MatchString(m[1]) {
			continue
		}
		if m := depRE.FindStringSubmatch(req); len(m) > 1 {
			dep := normalizeName(m[1])
			if !seen[dep] {
				seen[dep] = true
				deps = append(deps, dep)
			}
		}
	}
	return deps
}

func normalizeName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "_", "-")
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
