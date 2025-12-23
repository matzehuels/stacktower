package gitlab

import (
	"regexp"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

var repoURLPattern = regexp.MustCompile(`https?://gitlab\.com/([^/]+)/([^/]+)`)

// Client provides access to the GitLab API for repository metadata enrichment.
// It handles HTTP requests with caching, automatic retries, and optional authentication.
//
// All methods are safe for concurrent use by multiple goroutines.
//
// Note: Full metrics fetching (stars, contributors, etc.) is not yet implemented.
// Currently, this client focuses on URL extraction. Use [ExtractURL] to identify GitLab-hosted packages.
type Client struct {
	*integrations.Client
}

// NewClient creates a GitLab API client with optional authentication.
//
// The token parameter is a GitLab personal access token for authentication.
// Pass an empty string to use unauthenticated requests (public repositories only).
//
// The cacheTTL parameter sets how long responses are cached.
// Typical values: 1-24 hours for production, 0 for testing (no cache).
//
// Returns an error if the cache directory cannot be created or accessed.
// The returned Client is safe for concurrent use.
func NewClient(token string, cacheTTL time.Duration) (*Client, error) {
	cache, err := integrations.NewCacheWithNamespace("gitlab:", cacheTTL)
	if err != nil {
		return nil, err
	}

	var headers map[string]string
	if token != "" {
		headers = map[string]string{"PRIVATE-TOKEN": token}
	}

	return &Client{integrations.NewClient(cache, headers)}, nil
}

// ExtractURL extracts GitLab repository owner and name from package URLs.
//
// This function searches through urls map and homepage for GitLab URLs.
// It looks for patterns like "https://gitlab.com/owner/repo".
//
// Parameters:
//   - urls: Map of URL keys to URL values from package metadata (may be nil)
//   - homepage: Fallback homepage URL (may be empty)
//
// Returns:
//   - owner: Repository owner username (empty if not found)
//   - repo: Repository name (empty if not found)
//   - ok: true if a GitLab URL was found, false otherwise
//
// This function is safe for concurrent use.
func ExtractURL(urls map[string]string, homepage string) (owner, repo string, ok bool) {
	return integrations.ExtractRepoURL(repoURLPattern, urls, homepage)
}
