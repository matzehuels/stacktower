package packagist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

// PackageInfo holds metadata for a PHP package from Packagist.
//
// Package names follow Composer conventions (vendor/package format).
// Version is the latest stable version; dev versions are skipped.
// Dependencies exclude PHP, extensions (ext-*), libraries (lib-*), and Composer platform packages.
//
// Zero values: All string fields are empty, Dependencies is nil.
// This struct is safe for concurrent reads after construction.
type PackageInfo struct {
	Name         string   // Package name (e.g., "symfony/console", never empty in valid info)
	Version      string   // Latest stable version (e.g., "6.3.0", never empty in valid info)
	Dependencies []string // Composer require dependencies, filtered (nil or empty if none)
	Repository   string   // Normalized repository URL (empty if not provided)
	HomePage     string   // Homepage URL (may be empty)
	Description  string   // Package description (may be empty)
	License      string   // License identifier (may be empty, only first license if multiple)
	Author       string   // First author name (may be empty)
}

// Client provides access to the Packagist package registry API.
// It handles HTTP requests with caching and automatic retries.
//
// All methods are safe for concurrent use by multiple goroutines.
type Client struct {
	*integrations.Client
	baseURL string
}

// NewClient creates a Packagist client with the specified cache TTL.
//
// The cacheTTL parameter sets how long responses are cached.
// Typical values: 1-24 hours for production, 0 for testing (no cache).
//
// Returns an error if the cache directory cannot be created or accessed.
// The returned Client is safe for concurrent use.
func NewClient(cacheTTL time.Duration) (*Client, error) {
	cache, err := integrations.NewCacheWithNamespace("packagist:", cacheTTL)
	if err != nil {
		return nil, err
	}
	return &Client{
		Client:  integrations.NewClient(cache, nil),
		baseURL: "https://repo.packagist.org",
	}, nil
}

// FetchPackage retrieves metadata for a PHP package from Packagist.
//
// The pkg parameter must be in "vendor/package" format (e.g., "symfony/console").
// Package name is normalized to lowercase with whitespace trimmed.
// Package name cannot be empty; an empty string will result in an API error.
//
// If refresh is true, the cache is bypassed and a fresh API call is made.
// If refresh is false, cached data is returned if available and not expired.
//
// Version selection: The latest stable version is selected, skipping dev versions.
// If no stable version exists, the first version in the list is used.
//
// Returns:
//   - PackageInfo populated with metadata on success
//   - [integrations.ErrNotFound] if the package doesn't exist
//   - [integrations.ErrNetwork] for HTTP failures (timeout, 5xx, etc.)
//   - Other errors for JSON decoding failures or missing version data
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
	var data p2Response
	if err := c.Get(ctx, fmt.Sprintf("%s/p2/%s.json", c.baseURL, pkg), &data); err != nil {
		if errors.Is(err, integrations.ErrNotFound) {
			return fmt.Errorf("%w: packagist package %s", err, pkg)
		}
		return err
	}

	versions, ok := data.Packages[pkg]
	if !ok || len(versions) == 0 {
		return fmt.Errorf("no versions found for %s", pkg)
	}

	v := latestStable(versions)

	var license, author string
	if len(v.License) > 0 {
		license = v.License[0]
	}
	if len(v.Authors) > 0 {
		author = strings.TrimSpace(v.Authors[0].Name)
	}

	*info = PackageInfo{
		Name:         v.Name,
		Version:      v.Version,
		Description:  v.Description,
		License:      license,
		Author:       author,
		Repository:   integrations.NormalizeRepoURL(v.Source.URL),
		HomePage:     v.Homepage,
		Dependencies: slices.Collect(maps.Keys(filterDeps(v.Require))),
	}
	return nil
}

func filterDeps(require map[string]string) map[string]string {
	deps := make(map[string]string)
	for name, constraint := range require {
		ln := strings.ToLower(name)
		switch {
		case ln == "php" || ln == "composer-plugin-api" || ln == "composer-runtime-api":
			continue
		case strings.HasPrefix(ln, "ext-") || strings.HasPrefix(ln, "lib-"):
			continue
		case !strings.Contains(ln, "/"):
			continue
		}
		deps[ln] = constraint
	}
	return deps
}

func latestStable(versions []p2Version) p2Version {
	for _, v := range versions {
		lv := strings.ToLower(v.Version)
		if strings.Contains(lv, "dev") {
			continue
		}
		if strings.Contains(strings.TrimPrefix(lv, "v"), ".") {
			return v
		}
	}
	return versions[0]
}

type p2Response struct {
	Packages map[string][]p2Version `json:"packages"`
}

type p2Version struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Homepage    string            `json:"homepage"`
	License     []string          `json:"license"`
	Require     map[string]string `json:"require"`
	Source      struct {
		URL string `json:"url"`
	} `json:"source"`
	Authors []struct {
		Name string `json:"name"`
	} `json:"authors"`
}

func (v *p2Version) UnmarshalJSON(b []byte) error {
	type raw struct {
		Name        string          `json:"name"`
		Version     string          `json:"version"`
		Description string          `json:"description"`
		Homepage    string          `json:"homepage"`
		License     json.RawMessage `json:"license"`
		Require     json.RawMessage `json:"require"`
		Source      struct {
			URL string `json:"url"`
		} `json:"source"`
		Authors []struct {
			Name string `json:"name"`
		} `json:"authors"`
	}

	var r raw
	if err := json.Unmarshal(b, &r); err != nil {
		return err
	}

	v.Name = r.Name
	v.Version = r.Version
	v.Description = r.Description
	v.Homepage = r.Homepage
	v.Source = r.Source
	v.Authors = r.Authors

	if len(r.License) > 0 && string(r.License) != "null" {
		if err := json.Unmarshal(r.License, &v.License); err != nil {
			var single string
			if json.Unmarshal(r.License, &single) == nil && single != "" {
				v.License = []string{single}
			}
		}
	}

	if len(r.Require) > 0 && string(r.Require) != "null" {
		v.Require = make(map[string]string)
		if err := json.Unmarshal(r.Require, &v.Require); err != nil {
			var anyObj map[string]any
			if json.Unmarshal(r.Require, &anyObj) == nil {
				for k, val := range anyObj {
					if s, ok := val.(string); ok {
						v.Require[k] = s
					}
				}
			}
		}
	}
	return nil
}
