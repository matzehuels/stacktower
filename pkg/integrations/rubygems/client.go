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
		baseURL: "https://rubygems.org/api/v1",
	}, nil
}

func (c *Client) FetchGem(ctx context.Context, gem string, refresh bool) (*GemInfo, error) {
	gem = strings.ToLower(strings.TrimSpace(gem))
	key := "rubygems:" + gem

	var info GemInfo
	err := c.Cached(ctx, key, refresh, &info, func() error {
		return c.fetch(ctx, gem, &info)
	})
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *Client) fetch(ctx context.Context, gem string, info *GemInfo) error {
	var data gemResponse
	if err := c.Get(ctx, fmt.Sprintf("%s/gems/%s.json", c.baseURL, gem), &data); err != nil {
		if errors.Is(err, integrations.ErrNotFound) {
			return fmt.Errorf("%w: gem %s", err, gem)
		}
		return err
	}

	*info = GemInfo{
		Name:          data.Name,
		Version:       data.Version,
		Description:   data.Info,
		License:       strings.Join(data.Licenses, ", "),
		SourceCodeURI: data.SourceCodeURI,
		HomepageURI:   data.HomepageURI,
		Downloads:     data.Downloads,
		Authors:       data.Authors,
		Dependencies:  runtimeDeps(data.Dependencies.Runtime),
	}
	return nil
}

func runtimeDeps(deps []dependency) []string {
	seen := make(map[string]bool)
	var result []string
	for _, d := range deps {
		name := strings.ToLower(strings.TrimSpace(d.Name))
		if !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	return result
}

type gemResponse struct {
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	Info          string   `json:"info"`
	Licenses      []string `json:"licenses"`
	SourceCodeURI string   `json:"source_code_uri"`
	HomepageURI   string   `json:"homepage_uri"`
	Downloads     int      `json:"downloads"`
	Authors       string   `json:"authors"`
	Dependencies  struct {
		Runtime []dependency `json:"runtime"`
	} `json:"dependencies"`
}

type dependency struct {
	Name string `json:"name"`
}
