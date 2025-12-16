package goproxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

type ModuleInfo struct {
	Path         string
	Version      string
	Dependencies []string
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
		baseURL: "https://proxy.golang.org",
	}, nil
}

func (c *Client) FetchModule(ctx context.Context, mod string, refresh bool) (*ModuleInfo, error) {
	mod = normalizePath(mod)
	key := "goproxy:" + mod

	var info ModuleInfo
	err := c.Cached(ctx, key, refresh, &info, func() error {
		return c.fetch(ctx, mod, &info)
	})
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *Client) fetch(ctx context.Context, mod string, info *ModuleInfo) error {
	// Get latest version
	version, err := c.fetchLatest(ctx, mod)
	if err != nil {
		return err
	}

	// Get go.mod for this version
	deps, err := c.fetchGoMod(ctx, mod, version)
	if err != nil {
		// Some modules don't have go.mod, that's OK
		deps = nil
	}

	*info = ModuleInfo{
		Path:         mod,
		Version:      version,
		Dependencies: deps,
	}
	return nil
}

func (c *Client) fetchLatest(ctx context.Context, mod string) (string, error) {
	url := fmt.Sprintf("%s/%s/@latest", c.baseURL, escapePath(mod))

	var data latestResponse
	if err := c.Get(ctx, url, &data); err != nil {
		if errors.Is(err, integrations.ErrNotFound) {
			return "", fmt.Errorf("%w: go module %s", err, mod)
		}
		return "", err
	}
	return data.Version, nil
}

func (c *Client) fetchGoMod(ctx context.Context, mod, version string) ([]string, error) {
	url := fmt.Sprintf("%s/%s/@v/%s.mod", c.baseURL, escapePath(mod), version)

	body, err := c.GetText(ctx, url)
	if err != nil {
		return nil, err
	}
	return parseGoMod(strings.NewReader(body))
}

func parseGoMod(r io.Reader) ([]string, error) {
	var deps []string
	seen := make(map[string]bool)
	inRequire := false

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Handle require block
		if strings.HasPrefix(line, "require (") || line == "require(" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		// Single-line require
		if strings.HasPrefix(line, "require ") && !strings.Contains(line, "(") {
			line = strings.TrimPrefix(line, "require ")
		} else if !inRequire {
			continue
		}

		// Parse module path from require line
		// Format: module/path v1.2.3 [// indirect]
		if dep := parseRequireLine(line); dep != "" && !seen[dep] {
			seen[dep] = true
			deps = append(deps, dep)
		}
	}

	return deps, scanner.Err()
}

func parseRequireLine(line string) string {
	// Skip indirect dependencies
	if strings.Contains(line, "// indirect") {
		return ""
	}

	// Remove inline comments
	if idx := strings.Index(line, "//"); idx != -1 {
		line = line[:idx]
	}

	line = strings.TrimSpace(line)
	fields := strings.Fields(line)
	if len(fields) >= 1 {
		// Strip quotes from old-style go.mod files
		return strings.Trim(fields[0], `"`)
	}
	return ""
}

func normalizePath(path string) string {
	return strings.TrimSpace(path)
}

func escapePath(path string) string {
	var b strings.Builder
	for _, r := range path {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

type latestResponse struct {
	Version string `json:"Version"`
	Time    string `json:"Time"`
}
