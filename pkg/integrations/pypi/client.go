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

// PackageInfo holds metadata for a Python package from PyPI.
//
// Package names are normalized following PEP 503 (lowercase, underscores→hyphens).
// Dependencies list only runtime dependencies; extras, dev, and test deps are excluded.
//
// Zero values: All string fields are empty, Dependencies is nil.
// A nil Dependencies slice is valid and indicates no dependencies or failed dependency fetch.
// This struct is safe for concurrent reads after construction.
type PackageInfo struct {
	Name         string            // Normalized package name (e.g., "fastapi", never empty in valid info)
	Version      string            // Version string (e.g., "0.104.1", never empty in valid info)
	Dependencies []string          // Direct runtime dependencies, normalized names (nil or empty if none)
	ProjectURLs  map[string]string // Project URLs from metadata (e.g., "Homepage", "Repository", may be nil)
	HomePage     string            // Homepage URL (may be empty)
	Summary      string            // Short package description (may be empty)
	License      string            // License name or expression (may be empty)
	Author       string            // Author name (may be empty)
}

// Client provides access to the PyPI package registry API.
// It handles HTTP requests with caching and automatic retries.
//
// All methods are safe for concurrent use by multiple goroutines.
type Client struct {
	*integrations.Client
	baseURL string
}

// NewClient creates a PyPI client with the specified cache TTL.
//
// The cacheTTL parameter sets how long responses are cached.
// Typical values: 1-24 hours for production, 0 for testing (no cache).
//
// Returns an error if the cache directory cannot be created or accessed.
// The returned Client is safe for concurrent use.
func NewClient(cacheTTL time.Duration) (*Client, error) {
	cache, err := integrations.NewCacheWithNamespace("pypi:", cacheTTL)
	if err != nil {
		return nil, err
	}
	return &Client{
		Client:  integrations.NewClient(cache, nil),
		baseURL: "https://pypi.org/pypi",
	}, nil
}

// FetchPackage retrieves metadata for a Python package from PyPI.
//
// The pkg parameter is normalized automatically (case-insensitive, underscores→hyphens).
// Package name cannot be empty; an empty string will result in an API error.
//
// If refresh is true, the cache is bypassed and a fresh API call is made.
// If refresh is false, cached data is returned if available and not expired.
//
// Returns:
//   - PackageInfo populated with metadata on success
//   - [integrations.ErrNotFound] if the package doesn't exist
//   - [integrations.ErrNetwork] for HTTP failures (timeout, 5xx, etc.)
//   - Other errors for JSON decoding failures
//
// The returned PackageInfo pointer is never nil if err is nil.
// This method is safe for concurrent use.
func (c *Client) FetchPackage(ctx context.Context, pkg string, refresh bool) (*PackageInfo, error) {
	pkg = integrations.NormalizePkgName(pkg)
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
