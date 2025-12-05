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

type PackageInfo struct {
	Name         string
	Version      string
	Dependencies []string
	Repository   string
	HomePage     string
	Description  string
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
		baseURL: "https://repo.packagist.org",
	}, nil
}

func (c *Client) FetchPackage(ctx context.Context, pkg string, refresh bool) (*PackageInfo, error) {
	pkg = normalizeName(pkg)
	cacheKey := "packagist:" + pkg

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
	url := fmt.Sprintf("%s/p2/%s.json", c.baseURL, pkg)

	var data p2Response
	if err := c.DoRequest(ctx, url, nil, &data); err != nil {
		if errors.Is(err, integrations.ErrNotFound) {
			return fmt.Errorf("%w: packagist package %s", err, pkg)
		}
		return err
	}

	versions, ok := data.Packages[pkg]
	if !ok || len(versions) == 0 {
		return fmt.Errorf("no versions found for %s", pkg)
	}

	v := chooseLatestStable(versions)
	deps := filterComposerDeps(v.Require)

	license := ""
	if len(v.License) > 0 {
		license = v.License[0]
	}

	author := ""
	if len(v.Authors) > 0 {
		author = strings.TrimSpace(v.Authors[0].Name)
	}

	*info = PackageInfo{
		Name:         v.Name,
		Version:      v.Version,
		Description:  v.Description,
		License:      license,
		Author:       author,
		Repository:   normalizeRepoURL(v.Source.URL),
		HomePage:     v.Homepage,
		Dependencies: slices.Collect(maps.Keys(deps)),
	}

	return nil
}

func filterComposerDeps(require map[string]string) map[string]string {
	if require == nil {
		return map[string]string{}
	}

	deps := make(map[string]string)
	for name, constraint := range require {
		ln := strings.ToLower(name)

		if ln == "php" || strings.HasPrefix(ln, "ext-") || strings.HasPrefix(ln, "lib-") ||
			ln == "composer-plugin-api" || ln == "composer-runtime-api" {
			continue
		}

		if strings.Contains(ln, "/") {
			deps[ln] = constraint
		}
	}

	return deps
}

func chooseLatestStable(versions []p2Version) p2Version {
	for _, v := range versions {
		lv := strings.ToLower(v.Version)

		if strings.Contains(lv, "dev") {
			continue
		}

		versionNum := strings.TrimPrefix(lv, "v")
		if strings.Contains(versionNum, ".") {
			return v
		}
	}
	return versions[0]
}

func normalizeName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

func normalizeRepoURL(url string) string {
	if url == "" {
		return ""
	}

	url = strings.TrimSpace(url)
	url = strings.TrimPrefix(url, "git+")
	url = strings.ReplaceAll(url, "git@github.com:", "https://github.com/")
	url = strings.ReplaceAll(url, "git://github.com/", "https://github.com/")
	url = strings.TrimSuffix(url, ".git")

	return url
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
	Support     map[string]string `json:"support"`
	Source      struct{ URL string `json:"url"` } `json:"source"`
	Dist        struct{ URL string `json:"url"` } `json:"dist"`
	Authors     []struct{ Name string `json:"name"` } `json:"authors"`
}

func (v *p2Version) UnmarshalJSON(b []byte) error {
	type rawVersion struct {
		Name        string            `json:"name"`
		Version     string            `json:"version"`
		Description string            `json:"description"`
		Homepage    string            `json:"homepage"`
		License     json.RawMessage   `json:"license"`
		Require     json.RawMessage   `json:"require"`
		Support     map[string]string `json:"support"`
		Source      struct{ URL string `json:"url"` } `json:"source"`
		Dist        struct{ URL string `json:"url"` } `json:"dist"`
		Authors     []struct{ Name string `json:"name"` } `json:"authors"`
	}

	var rv rawVersion
	if err := json.Unmarshal(b, &rv); err != nil {
		return err
	}

	var license []string
	if len(rv.License) > 0 && string(rv.License) != "null" {
		if err := json.Unmarshal(rv.License, &license); err != nil {
			var single string
			if err := json.Unmarshal(rv.License, &single); err == nil && single != "" {
				license = []string{single}
			}
		}
	}

	require := map[string]string{}
	if len(rv.Require) > 0 && string(rv.Require) != "null" {
		if err := json.Unmarshal(rv.Require, &require); err != nil {
			var anyObj map[string]any
			if err := json.Unmarshal(rv.Require, &anyObj); err == nil {
				require = make(map[string]string, len(anyObj))
				for k, val := range anyObj {
					if s, ok := val.(string); ok {
						require[k] = s
					}
				}
			}
		}
	}

	v.Name = rv.Name
	v.Version = rv.Version
	v.Description = rv.Description
	v.Homepage = rv.Homepage
	v.License = license
	v.Require = require
	v.Support = rv.Support
	v.Source = rv.Source
	v.Dist = rv.Dist
	v.Authors = rv.Authors

	return nil
}
