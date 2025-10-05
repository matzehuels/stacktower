package crates

import (
	"context"
	"errors"
	"fmt"
	"time"

	"stacktower/pkg/integrations"
)

type CrateInfo struct {
	Name         string
	Version      string
	Dependencies []string
	Repository   string
	HomePage     string
	Description  string
	License      string
	Downloads    int
}

type Client struct {
	integrations.BaseClient
	baseURL string
	headers map[string]string
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
		baseURL: "https://crates.io/api/v1",
		headers: map[string]string{"User-Agent": "stacktower/1.0 (https://github.com/stacktower)"},
	}, nil
}

func (c *Client) FetchCrate(ctx context.Context, crate string, refresh bool) (*CrateInfo, error) {
	cacheKey := "crates:" + crate

	var info CrateInfo
	err := c.FetchWithCache(ctx, cacheKey, refresh, func() error {
		return c.fetchCrate(ctx, crate, &info)
	}, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *Client) fetchCrate(ctx context.Context, crate string, info *CrateInfo) error {
	var crateData crateResponse
	if err := c.DoRequest(ctx, fmt.Sprintf("%s/crates/%s", c.baseURL, crate), c.headers, &crateData); err != nil {
		if errors.Is(err, integrations.ErrNotFound) {
			return fmt.Errorf("%w: crate %s", err, crate)
		}
		return err
	}

	deps, err := c.fetchDependencies(ctx, crate, crateData.Crate.MaxVersion)
	if err != nil {
		return err
	}

	*info = CrateInfo{
		Name:         crateData.Crate.Name,
		Version:      crateData.Crate.MaxVersion,
		Description:  crateData.Crate.Description,
		License:      crateData.Crate.License,
		Repository:   crateData.Crate.Repository,
		HomePage:     crateData.Crate.HomePage,
		Downloads:    crateData.Crate.Downloads,
		Dependencies: deps,
	}
	return nil
}

func (c *Client) fetchDependencies(ctx context.Context, crate, version string) ([]string, error) {
	url := fmt.Sprintf("%s/crates/%s/%s/dependencies", c.baseURL, crate, version)

	var data depsResponse
	if err := c.DoRequest(ctx, url, c.headers, &data); err != nil {
		return nil, nil
	}

	var deps []string
	for _, d := range data.Dependencies {
		if d.Kind == "normal" && !d.Optional {
			deps = append(deps, d.CrateID)
		}
	}
	return deps, nil
}

type crateResponse struct {
	Crate crateData `json:"crate"`
}

type crateData struct {
	Name        string `json:"name"`
	MaxVersion  string `json:"max_version"`
	Description string `json:"description"`
	License     string `json:"license"`
	Repository  string `json:"repository"`
	HomePage    string `json:"homepage"`
	Downloads   int    `json:"downloads"`
}

type depsResponse struct {
	Dependencies []dependency `json:"dependencies"`
}

type dependency struct {
	CrateID  string `json:"crate_id"`
	Kind     string `json:"kind"`
	Optional bool   `json:"optional"`
}
