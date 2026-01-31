package rubygems

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/cache"
	"github.com/matzehuels/stacktower/pkg/integrations"
)

// GemInfo holds metadata for a Ruby gem from RubyGems.
//
// Gem names are normalized to lowercase.
// Dependencies include only runtime dependencies; development dependencies are excluded.
//
// Zero values: All string fields are empty, Dependencies is nil, Downloads is 0.
// A Downloads value of 0 is valid for newly published gems.
// This struct is safe for concurrent reads after construction.
type GemInfo struct {
	Name          string   // Gem name, normalized lowercase (e.g., "rails", never empty in valid info)
	Version       string   // Current version (e.g., "7.1.2", never empty in valid info)
	Dependencies  []string // Runtime dependency gem names, normalized (nil or empty if none)
	SourceCodeURI string   // Source code repository URL (may be empty)
	HomepageURI   string   // Homepage URL (may be empty)
	Description   string   // Gem description/info (may be empty)
	License       string   // License(s), comma-separated if multiple (may be empty)
	Downloads     int      // Total download count (0 for new gems)
	Authors       string   // Author name(s) (may be empty)
}

// Client provides access to the RubyGems package registry API.
// It handles HTTP requests with caching and automatic retries.
//
// All methods are safe for concurrent use by multiple goroutines.
type Client struct {
	*integrations.Client
	baseURL string
}

// NewClient creates a RubyGems client with the given cache backend.
//
// Parameters:
//   - backend: Cache backend for HTTP response caching (use storage.NullBackend{} for no caching)
//   - cacheTTL: How long responses are cached (typical: 1-24 hours)
//
// The returned Client is safe for concurrent use.
func NewClient(backend cache.Cache, cacheTTL time.Duration) *Client {
	return &Client{
		Client:  integrations.NewClient(backend, "rubygems:", cacheTTL, nil),
		baseURL: "https://rubygems.org/api/v1",
	}
}

// FetchGem retrieves metadata for a Ruby gem from RubyGems.
//
// The gem parameter is normalized to lowercase with whitespace trimmed.
// Gem name cannot be empty; an empty string will result in an API error.
//
// If refresh is true, the cache is bypassed and a fresh API call is made.
// If refresh is false, cached data is returned if available and not expired.
//
// Returns:
//   - GemInfo populated with metadata on success
//   - [integrations.ErrNotFound] if the gem doesn't exist
//   - [integrations.ErrNetwork] for HTTP failures (timeout, 5xx, etc.)
//   - Other errors for JSON decoding failures
//
// The returned GemInfo pointer is never nil if err is nil.
// This method is safe for concurrent use.
func (c *Client) FetchGem(ctx context.Context, gem string, refresh bool) (*GemInfo, error) {
	gem = strings.ToLower(strings.TrimSpace(gem))
	key := gem

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
