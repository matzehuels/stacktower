package crates

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations"
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
	*integrations.Client
	baseURL string
}

func NewClient(cacheTTL time.Duration) (*Client, error) {
	cache, err := integrations.NewCache(cacheTTL)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{
		"User-Agent": "stacktower/1.0 (https://github.com/matzehuels/stacktower)",
	}
	return &Client{
		Client:  integrations.NewClient(cache, headers),
		baseURL: "https://crates.io/api/v1",
	}, nil
}

func (c *Client) FetchCrate(ctx context.Context, crate string, refresh bool) (*CrateInfo, error) {
	key := "crates:" + crate

	var info CrateInfo
	err := c.Cached(ctx, key, refresh, &info, func() error {
		return c.fetch(ctx, crate, &info)
	})
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *Client) fetch(ctx context.Context, crate string, info *CrateInfo) error {
	var data crateResponse
	if err := c.Get(ctx, fmt.Sprintf("%s/crates/%s", c.baseURL, crate), &data); err != nil {
		if errors.Is(err, integrations.ErrNotFound) {
			return fmt.Errorf("%w: crate %s", err, crate)
		}
		return err
	}

	deps, _ := c.fetchDeps(ctx, crate, data.Crate.MaxVersion)

	*info = CrateInfo{
		Name:         data.Crate.Name,
		Version:      data.Crate.MaxVersion,
		Description:  data.Crate.Description,
		License:      data.Crate.License,
		Repository:   data.Crate.Repository,
		HomePage:     data.Crate.HomePage,
		Downloads:    data.Crate.Downloads,
		Dependencies: deps,
	}
	return nil
}

func (c *Client) fetchDeps(ctx context.Context, crate, version string) ([]string, error) {
	url := fmt.Sprintf("%s/crates/%s/%s/dependencies", c.baseURL, crate, version)

	var data depsResponse
	if err := c.Get(ctx, url, &data); err != nil {
		return nil, err
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
	Crate struct {
		Name        string `json:"name"`
		MaxVersion  string `json:"max_version"`
		Description string `json:"description"`
		License     string `json:"license"`
		Repository  string `json:"repository"`
		HomePage    string `json:"homepage"`
		Downloads   int    `json:"downloads"`
	} `json:"crate"`
}

type depsResponse struct {
	Dependencies []struct {
		CrateID  string `json:"crate_id"`
		Kind     string `json:"kind"`
		Optional bool   `json:"optional"`
	} `json:"dependencies"`
}
