package integrations

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/httputil"
)

// httpTimeout is the default timeout for all HTTP requests to registry APIs.
// Individual registries do not override this value.
const httpTimeout = 10 * time.Second

var (
	// ErrNotFound is returned when a package or resource doesn't exist in the registry.
	// This corresponds to HTTP 404 responses.
	// Callers should check with errors.Is(err, integrations.ErrNotFound).
	// This error is never wrapped with additional context.
	ErrNotFound = errors.New("resource not found")

	// ErrNetwork is returned for HTTP failures (timeouts, connection errors, 5xx responses).
	// This error may be wrapped with [httputil.RetryableError] for 5xx status codes.
	// Callers should check with errors.Is(err, integrations.ErrNetwork) for any network issue,
	// or errors.As(err, &httputil.RetryableError{}) to detect retryable failures specifically.
	ErrNetwork = errors.New("network error")
)

// RepoMetrics holds repository-level data fetched from GitHub or GitLab.
// Used to enrich package metadata with maintenance and popularity indicators.
//
// Zero values: All string fields are empty, integers are 0, time pointers are nil.
// Nil Contributors slice is valid and indicates no contributor data was fetched.
//
// This struct is safe for concurrent reads after construction but not for concurrent writes.
type RepoMetrics struct {
	RepoURL       string        `json:"repo_url"`                   // Canonical repository URL (https://...). Never empty in valid metrics.
	Owner         string        `json:"owner"`                      // Repository owner username. Never empty in valid metrics.
	Stars         int           `json:"stars"`                      // GitHub/GitLab star count. 0 is a valid value for new repositories.
	SizeKB        int           `json:"size_kb,omitempty"`          // Repository size in kilobytes. 0 means not available or very small.
	LastCommitAt  *time.Time    `json:"last_commit_at,omitempty"`   // Date of most recent commit. Nil if not available.
	LastReleaseAt *time.Time    `json:"last_release_at,omitempty"`  // Date of most recent release. Nil if no releases or not available.
	License       string        `json:"license,omitempty"`          // SPDX license identifier (e.g., "MIT", "Apache-2.0"). Empty if not detected.
	Contributors  []Contributor `json:"top_contributors,omitempty"` // Top contributors by commit count (typically top 5). Nil or empty if not available.
	Language      string        `json:"language,omitempty"`         // Primary repository language (e.g., "Go", "Python"). Empty if not detected.
	Topics        []string      `json:"topics,omitempty"`           // Repository topic tags. Nil or empty if none.
	Archived      bool          `json:"archived"`                   // Whether the repository is archived. False means active or unknown.
}

// Contributor represents a repository contributor with their contribution count.
// Used for bus factor analysis and maintainer identification.
//
// Zero values: Login is empty, Contributions is 0. A Contributor with 0 contributions is invalid.
// This struct is safe for concurrent reads.
type Contributor struct {
	Login         string `json:"login"`         // GitHub/GitLab username. Never empty in valid contributors.
	Contributions int    `json:"contributions"` // Number of commits. Always positive in valid contributors.
}

// NewHTTPClient creates an HTTP client with a standard timeout for registry requests.
// The returned client has a 10-second timeout applied to all requests.
//
// The client is safe for concurrent use by multiple goroutines.
// Returns a new client on every call; clients are not pooled.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: httpTimeout}
}

// NewCache creates a file-based cache with the given TTL in the default cache directory.
// See [httputil.NewCache] for details on cache location and behavior.
//
// The ttl parameter must be positive. A ttl of 0 means items never expire (not recommended).
// Negative ttl values are invalid and will be treated as 0.
//
// For registry-specific clients, prefer using [NewCacheWithNamespace] to automatically
// scope cache keys by registry name and prevent collisions.
//
// Returns an error if the cache directory cannot be created or accessed.
// The returned cache is safe for concurrent use by multiple goroutines.
func NewCache(ttl time.Duration) (*httputil.Cache, error) {
	return httputil.NewCache("", ttl)
}

// NewCacheWithNamespace creates a namespaced cache for a specific registry.
// The namespace parameter (e.g., "pypi:", "npm:") is automatically prefixed to all
// cache keys, preventing collisions between different registries.
//
// The namespace should be non-empty and typically ends with a colon. An empty namespace
// is valid but defeats the purpose of this function; use [NewCache] instead.
//
// The ttl parameter must be positive. A ttl of 0 means items never expire (not recommended).
//
// This is the preferred way to create caches for registry clients:
//
//	cache, err := integrations.NewCacheWithNamespace("pypi:", 24*time.Hour)
//	client := integrations.NewClient(cache, nil)
//
// Returns an error if the cache directory cannot be created or accessed.
// The returned cache is safe for concurrent use by multiple goroutines.
func NewCacheWithNamespace(namespace string, ttl time.Duration) (*httputil.Cache, error) {
	cache, err := httputil.NewCache("", ttl)
	if err != nil {
		return nil, err
	}
	return cache.Namespace(namespace), nil
}

// NormalizePkgName converts a package name to its canonical form.
// Applies lowercase and replaces underscores with hyphens, following PEP 503
// normalization rules used by PyPI and other registries.
//
// Normalization steps:
//  1. Trim leading and trailing whitespace
//  2. Convert to lowercase
//  3. Replace all underscores with hyphens
//
// Examples:
//
//	NormalizePkgName("FastAPI")      → "fastapi"
//	NormalizePkgName("my_package")   → "my-package"
//	NormalizePkgName("  Spaces  ")   → "spaces"
//
// An empty string input returns an empty string.
// This function is safe for concurrent use.
func NormalizePkgName(name string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "_", "-")
}

var repoURLReplacer = strings.NewReplacer(
	"git@github.com:", "https://github.com/",
	"git://github.com/", "https://github.com/",
)

// NormalizeRepoURL converts various repository URL formats to canonical HTTPS form.
// Handles git@, git://, and git+ prefixes, and removes .git suffixes.
//
// Transformations applied:
//   - git@github.com:user/repo → https://github.com/user/repo
//   - git://github.com/user/repo → https://github.com/user/repo
//   - git+https://example.com/repo.git → https://example.com/repo
//   - https://example.com/repo.git → https://example.com/repo
//
// Returns an empty string if the input is empty or contains only whitespace.
// Non-git URLs are returned unchanged after whitespace trimming and .git suffix removal.
// This function is safe for concurrent use.
func NormalizeRepoURL(raw string) string {
	if raw == "" {
		return ""
	}
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "git+")
	s = repoURLReplacer.Replace(s)
	return strings.TrimSuffix(s, ".git")
}

var repoURLKeys = []string{"Source", "Repository", "Code", "Homepage"}

// ExtractRepoURL finds GitHub/GitLab owner and repo from package URLs.
// It searches through urls using standard keys (Source, Repository, Code, Homepage)
// and falls back to homepage if no match is found.
//
// The re parameter should match URLs and capture:
//   - Group 1: owner/organization name
//   - Group 2: repository name
//
// Examples:
//
//	re := regexp.MustCompile(`https?://github\.com/([^/]+)/([^/]+)`)
//	owner, repo, ok := ExtractRepoURL(re, pkg.ProjectURLs, pkg.HomePage)
//
// URLs containing "/sponsors/" are automatically skipped to avoid false positives.
// The .git suffix is trimmed from the repository name if present.
//
// Parameters:
//   - re: Regular expression with exactly 2 capture groups (must not be nil)
//   - urls: Map of URL keys to URL values (may be nil or empty)
//   - homepage: Fallback homepage URL (may be empty)
//
// Returns:
//   - owner: The repository owner/organization (empty if not found)
//   - repo: The repository name without .git suffix (empty if not found)
//   - ok: true if a valid match was found, false otherwise
//
// This function is safe for concurrent use if re is not mutated.
// Panics if re is nil.
func ExtractRepoURL(re *regexp.Regexp, urls map[string]string, homepage string) (owner, repo string, ok bool) {
	match := func(u string) bool {
		if strings.Contains(u, "/sponsors/") {
			return false
		}
		if m := re.FindStringSubmatch(u); len(m) >= 3 {
			owner = m[1]
			repo = strings.TrimSuffix(m[2], ".git")
			ok = true
			return true
		}
		return false
	}

	for _, key := range repoURLKeys {
		if u, exists := urls[key]; exists && match(u) {
			return
		}
	}
	for _, u := range urls {
		if match(u) {
			return
		}
	}
	if homepage != "" {
		match(homepage)
	}
	return
}

// URLEncode percent-encodes a string for use in URLs.
// This is a convenience wrapper around [url.QueryEscape].
//
// Spaces are encoded as "+", and special characters as "%XX" hex sequences.
// An empty string returns an empty string.
// This function is safe for concurrent use.
func URLEncode(s string) string { return url.QueryEscape(s) }
