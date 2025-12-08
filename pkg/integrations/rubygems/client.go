package rubygems

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

type GemInfo struct {
	Name          string
	Version       string
	Dependencies  []string
	SourceCodeURI string
	HomepageURI   string
	Description   string
	License       string
	Downloads     int
	Authors       string
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
		baseURL: "https://rubygems.org/api/v1",
	}, nil
}

func (c *Client) FetchGem(ctx context.Context, gem string, refresh bool) (*GemInfo, error) {
	gem = normalizeName(gem)
	cacheKey := "rubygems:" + gem

	var info GemInfo
	err := c.FetchWithCache(ctx, cacheKey, refresh, func() error {
		return c.fetchGem(ctx, gem, &info)
	}, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *Client) fetchGem(ctx context.Context, gem string, info *GemInfo) error {
	url := fmt.Sprintf("%s/gems/%s.json", c.baseURL, gem)

	var data gemResponse
	if err := c.DoRequest(ctx, url, nil, &data); err != nil {
		if errors.Is(err, integrations.ErrNotFound) {
			return fmt.Errorf("%w: rubygems gem %s", err, gem)
		}
		return err
	}

	*info = GemInfo{
		Name:          data.Name,
		Version:       data.Version,
		Description:   data.Info,
		License:       joinLicenses(data.Licenses),
		SourceCodeURI: data.SourceCodeURI,
		HomepageURI:   data.HomepageURI,
		Downloads:     data.Downloads,
		Authors:       data.Authors,
		Dependencies:  extractDeps(data.Dependencies),
	}
	return nil
}

func extractDeps(deps dependenciesResponse) []string {
	seen := make(map[string]bool)
	var result []string

	// Only include runtime dependencies, skip development dependencies
	for _, dep := range deps.Runtime {
		name := normalizeName(dep.Name)
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	return result
}

func joinLicenses(licenses []string) string {
	if len(licenses) == 0 {
		return ""
	}
	return strings.Join(licenses, ", ")
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

type gemResponse struct {
	Name          string               `json:"name"`
	Version       string               `json:"version"`
	Info          string               `json:"info"`
	Licenses      []string             `json:"licenses"`
	SourceCodeURI string               `json:"source_code_uri"`
	HomepageURI   string               `json:"homepage_uri"`
	Downloads     int                  `json:"downloads"`
	Authors       string               `json:"authors"`
	Dependencies  dependenciesResponse `json:"dependencies"`
}

type dependenciesResponse struct {
	Development []dependencyInfo `json:"development"`
	Runtime     []dependencyInfo `json:"runtime"`
}

type dependencyInfo struct {
	Name         string `json:"name"`
	Requirements string `json:"requirements"`
}
