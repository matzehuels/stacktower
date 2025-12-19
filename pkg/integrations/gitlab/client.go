package gitlab

import (
	"regexp"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

var repoURLPattern = regexp.MustCompile(`https?://gitlab\.com/([^/]+)/([^/]+)`)

// Client provides access to the GitLab API for repository metadata enrichment.
// It handles HTTP requests with caching, automatic retries, and optional authentication.
type Client struct {
	*integrations.Client
}

// NewClient creates a GitLab API client with optional authentication.
// Pass an empty string for token to use unauthenticated requests (lower rate limits).
func NewClient(token string, cacheTTL time.Duration) (*Client, error) {
	cache, err := integrations.NewCache(cacheTTL)
	if err != nil {
		return nil, err
	}

	var headers map[string]string
	if token != "" {
		headers = map[string]string{"PRIVATE-TOKEN": token}
	}

	return &Client{integrations.NewClient(cache, headers)}, nil
}

func ExtractURL(urls map[string]string, homepage string) (owner, repo string, ok bool) {
	return integrations.ExtractRepoURL(repoURLPattern, urls, homepage)
}
