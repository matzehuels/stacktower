package goproxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/cache"
	"github.com/matzehuels/stacktower/pkg/integrations"
)

// ModuleInfo holds metadata for a Go module from the Go module proxy.
//
// Dependencies include only direct dependencies; indirect dependencies (marked with "// indirect") are excluded.
// Some modules (pre-modules or minimal modules) may not have a go.mod file; Dependencies will be nil/empty.
//
// Zero values: All string fields are empty, Dependencies is nil.
// This struct is safe for concurrent reads after construction.
type ModuleInfo struct {
	Path         string   // Module path (e.g., "github.com/spf13/cobra", never empty in valid info)
	Version      string   // Latest version from @latest endpoint (e.g., "v1.8.0", never empty in valid info)
	Dependencies []string // Direct dependency module paths (nil or empty if none or no go.mod)
}

// Client provides access to the Go module proxy API.
// It handles HTTP requests with caching and automatic retries.
//
// All methods are safe for concurrent use by multiple goroutines.
type Client struct {
	*integrations.Client
	baseURL string
}

// NewClient creates a Go module proxy client with the given cache backend.
//
// Parameters:
//   - backend: Cache backend for HTTP response caching (use storage.NullBackend{} for no caching)
//   - cacheTTL: How long responses are cached (typical: 1-24 hours)
//
// The returned Client is safe for concurrent use.
func NewClient(backend cache.Cache, cacheTTL time.Duration) *Client {
	return &Client{
		Client:  integrations.NewClient(backend, "goproxy:", cacheTTL, nil),
		baseURL: "https://proxy.golang.org",
	}
}

// FetchModule retrieves metadata for a Go module from the module proxy.
//
// The mod parameter should be a full module path (e.g., "github.com/user/repo").
// Module paths with uppercase letters are escaped per the Go module proxy protocol.
// Module path cannot be empty; an empty string will result in an API error.
//
// If refresh is true, the cache is bypassed and a fresh API call is made.
// If refresh is false, cached data is returned if available and not expired.
//
// This method performs two API calls:
//  1. @latest endpoint to get the latest version
//  2. .mod endpoint to fetch and parse go.mod for dependencies
//
// go.mod fetch failures are silently ignored; Dependencies will be nil/empty if it fails.
// This is normal for pre-module packages or minimal modules without dependencies.
//
// Returns:
//   - ModuleInfo populated with metadata on success
//   - [integrations.ErrNotFound] if the module doesn't exist
//   - [integrations.ErrNetwork] for HTTP failures (timeout, 5xx, etc.)
//   - Other errors for JSON decoding failures
//
// The returned ModuleInfo pointer is never nil if err is nil.
// This method is safe for concurrent use.
func (c *Client) FetchModule(ctx context.Context, mod string, refresh bool) (*ModuleInfo, error) {
	mod = normalizePath(mod)
	key := mod

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
