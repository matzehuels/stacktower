package crates

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/matzehuels/stacktower/pkg/cache"
	"github.com/matzehuels/stacktower/pkg/integrations"
)

// CrateInfo holds metadata for a Rust crate from crates.io.
//
// The Version field contains the max_version (latest stable or highest version).
// Dependencies include only "normal" (non-dev, non-optional) dependencies.
//
// Zero values: All string fields are empty, Dependencies is nil, Downloads is 0.
// A Downloads value of 0 is valid for newly published crates.
// This struct is safe for concurrent reads after construction.
type CrateInfo struct {
	Name         string   // Crate name (e.g., "serde", never empty in valid info)
	Version      string   // Latest version (e.g., "1.0.193", never empty in valid info)
	Dependencies []string // Normal dependency crate names (nil or empty if none)
	Repository   string   // Repository URL (may be empty)
	HomePage     string   // Homepage URL (may be empty)
	Description  string   // Crate description (may be empty)
	License      string   // License identifier(s) (may be empty or "MIT OR Apache-2.0")
	Downloads    int      // Total download count across all versions (0 for new crates)
}

// Client provides access to the crates.io package registry API.
// It handles HTTP requests with caching and automatic retries.
//
// All methods are safe for concurrent use by multiple goroutines.
//
// Note: crates.io requires a User-Agent header; this client sets one automatically.
type Client struct {
	*integrations.Client
	baseURL string
}

// NewClient creates a crates.io client with the given cache backend.
//
// Parameters:
//   - backend: Cache backend for HTTP response caching (use storage.NullBackend{} for no caching)
//   - cacheTTL: How long responses are cached (typical: 1-24 hours)
//
// The client includes a User-Agent header as required by crates.io API policy.
// The returned Client is safe for concurrent use.
func NewClient(backend cache.Cache, cacheTTL time.Duration) *Client {
	headers := map[string]string{
		"User-Agent": "stacktower/1.0 (https://github.com/matzehuels/stacktower)",
	}
	return &Client{
		Client:  integrations.NewClient(backend, "crates:", cacheTTL, headers),
		baseURL: "https://crates.io/api/v1",
	}
}

// FetchCrate retrieves metadata for a Rust crate from crates.io.
//
// The crate parameter is case-sensitive and must match the published crate name exactly.
// Crate name cannot be empty; an empty string will result in an API error.
//
// If refresh is true, the cache is bypassed and a fresh API call is made.
// If refresh is false, cached data is returned if available and not expired.
//
// Dependency fetching failures are silently ignored; Dependencies will be empty/nil
// if the secondary API call fails. This is not considered an error.
//
// Returns:
//   - CrateInfo populated with metadata on success
//   - [integrations.ErrNotFound] if the crate doesn't exist
//   - [integrations.ErrNetwork] for HTTP failures (timeout, 5xx, etc.)
//   - Other errors for JSON decoding failures
//
// The returned CrateInfo pointer is never nil if err is nil.
// This method is safe for concurrent use.
func (c *Client) FetchCrate(ctx context.Context, crate string, refresh bool) (*CrateInfo, error) {
	key := crate

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
