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

const httpTimeout = 10 * time.Second

var (
	// ErrNotFound is returned when a package or resource doesn't exist in the registry.
	ErrNotFound = errors.New("resource not found")

	// ErrNetwork is returned for HTTP failures (timeouts, connection errors, 5xx responses).
	ErrNetwork = errors.New("network error")
)

// RepoMetrics holds repository-level data fetched from GitHub or GitLab.
// Used to enrich package metadata with maintenance and popularity indicators.
type RepoMetrics struct {
	RepoURL       string        `json:"repo_url"`                   // Canonical repository URL (https://...)
	Owner         string        `json:"owner"`                      // Repository owner username
	Stars         int           `json:"stars"`                      // GitHub/GitLab star count
	SizeKB        int           `json:"size_kb,omitempty"`          // Repository size in kilobytes
	LastCommitAt  *time.Time    `json:"last_commit_at,omitempty"`   // Date of most recent commit
	LastReleaseAt *time.Time    `json:"last_release_at,omitempty"`  // Date of most recent release
	License       string        `json:"license,omitempty"`          // SPDX license identifier
	Contributors  []Contributor `json:"top_contributors,omitempty"` // Top contributors by commit count
	Language      string        `json:"language,omitempty"`         // Primary repository language
	Topics        []string      `json:"topics,omitempty"`           // Repository topic tags
	Archived      bool          `json:"archived"`                   // Whether the repository is archived
}

// Contributor represents a repository contributor with their contribution count.
type Contributor struct {
	Login         string `json:"login"`         // GitHub/GitLab username
	Contributions int    `json:"contributions"` // Number of commits
}

// NewHTTPClient creates an HTTP client with a standard timeout for registry requests.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: httpTimeout}
}

// NewCache creates a file-based cache with the given TTL in the default cache directory.
// See [httputil.NewCache] for details on cache location and behavior.
func NewCache(ttl time.Duration) (*httputil.Cache, error) {
	return httputil.NewCache("", ttl)
}

// NormalizePkgName converts a package name to its canonical form.
// Applies lowercase and replaces underscores with hyphens, following PEP 503
// normalization rules used by PyPI and other registries.
func NormalizePkgName(name string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "_", "-")
}

var repoURLReplacer = strings.NewReplacer(
	"git@github.com:", "https://github.com/",
	"git://github.com/", "https://github.com/",
)

// NormalizeRepoURL converts various repository URL formats to canonical HTTPS form.
// Handles git@, git://, and git+ prefixes, and removes .git suffixes.
// Returns empty string if raw is empty.
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
// and falls back to homepage if no match is found. The re parameter should match
// URLs and capture owner (group 1) and repo name (group 2).
// Returns ok=false if no valid repository URL is found.
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
func URLEncode(s string) string { return url.QueryEscape(s) }
