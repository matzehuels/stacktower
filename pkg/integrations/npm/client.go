package npm

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/cache"
	"github.com/matzehuels/stacktower/pkg/integrations"
)

// PackageInfo holds metadata for a JavaScript/TypeScript package from npm.
//
// The Version field always contains the "latest" dist-tag version.
// Dependencies include only runtime "dependencies", not devDependencies or peerDependencies.
//
// Zero values: All string fields are empty, Dependencies is nil.
// This struct is safe for concurrent reads after construction.
type PackageInfo struct {
	Name         string   // Package name as published (e.g., "@scope/package", never empty in valid info)
	Version      string   // Latest version tag (e.g., "4.18.2", never empty in valid info)
	Dependencies []string // Runtime dependency names (nil or empty if none)
	Repository   string   // Normalized repository URL (empty if not provided)
	HomePage     string   // Homepage URL (may be empty)
	Description  string   // Package description (may be empty)
	License      string   // License identifier (e.g., "MIT", may be empty)
	Author       string   // Author name (may be empty)
}

// Client provides access to the npm package registry API.
// It handles HTTP requests with caching and automatic retries.
//
// All methods are safe for concurrent use by multiple goroutines.
type Client struct {
	*integrations.Client
	baseURL string
}

// NewClient creates an npm client with the given cache backend.
//
// Parameters:
//   - backend: Cache backend for HTTP response caching (use storage.NullBackend{} for no caching)
//   - cacheTTL: How long responses are cached (typical: 1-24 hours)
//
// The returned Client is safe for concurrent use.
func NewClient(backend cache.Cache, cacheTTL time.Duration) *Client {
	return &Client{
		Client:  integrations.NewClient(backend, "npm:", cacheTTL, nil),
		baseURL: "https://registry.npmjs.org",
	}
}

// FetchPackage retrieves metadata for a JavaScript/TypeScript package from npm.
//
// The pkg parameter is normalized to lowercase with whitespace trimmed.
// Supports scoped packages (e.g., "@types/node").
// Package name cannot be empty; an empty string will result in an API error.
//
// If refresh is true, the cache is bypassed and a fresh API call is made.
// If refresh is false, cached data is returned if available and not expired.
//
// Returns:
//   - PackageInfo populated with metadata for the "latest" dist-tag version
//   - [integrations.ErrNotFound] if the package doesn't exist
//   - [integrations.ErrNetwork] for HTTP failures (timeout, 5xx, etc.)
//   - Other errors for JSON decoding failures or missing "latest" version
//
// The returned PackageInfo pointer is never nil if err is nil.
// This method is safe for concurrent use.
func (c *Client) FetchPackage(ctx context.Context, pkg string, refresh bool) (*PackageInfo, error) {
	pkg = strings.ToLower(strings.TrimSpace(pkg))
	key := pkg

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
