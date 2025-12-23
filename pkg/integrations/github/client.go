package github

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

var repoURLPattern = regexp.MustCompile(`https?://github\.com/([^/]+)/([^/]+?)(?:\.git)?(?:[/?#]|$)`)

// Client provides access to the GitHub API for repository metadata enrichment.
// It handles HTTP requests with caching, automatic retries, and optional authentication.
//
// All methods are safe for concurrent use by multiple goroutines.
type Client struct {
	*integrations.Client
	baseURL string
}

// NewClient creates a GitHub API client with optional authentication.
//
// The token parameter is a GitHub personal access token for authentication.
// Pass an empty string to use unauthenticated requests.
//
// Rate limits:
//   - Unauthenticated: 60 requests/hour per IP
//   - Authenticated: 5,000 requests/hour per token
//
// Authentication is strongly recommended for production use to avoid rate limiting.
//
// The cacheTTL parameter sets how long responses are cached.
// Typical values: 1-24 hours for production, 0 for testing (no cache).
//
// Returns an error if the cache directory cannot be created or accessed.
// The returned Client is safe for concurrent use.
func NewClient(token string, cacheTTL time.Duration) (*Client, error) {
	cache, err := integrations.NewCacheWithNamespace("github:", cacheTTL)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{"Accept": "application/vnd.github.v3+json"}
	if token != "" {
		headers["Authorization"] = "Bearer " + token
	}

	return &Client{
		Client:  integrations.NewClient(cache, headers),
		baseURL: "https://api.github.com",
	}, nil
}

// Fetch retrieves repository metrics (stars, maintainers, activity) from GitHub.
//
// Parameters:
//   - owner: Repository owner username (e.g., "pallets")
//   - repo: Repository name (e.g., "flask")
//   - refresh: If true, bypass cache and fetch fresh data
//
// The method performs up to 3 API calls:
//  1. Repository metadata (required)
//  2. Latest release data (optional, silently ignored if no releases)
//  3. Contributors list (optional, top 5, silently ignored on failure)
//
// If refresh is true, the cache is bypassed and a fresh API call is made.
// If refresh is false, cached data is returned if available and not expired.
//
// Returns:
//   - RepoMetrics populated with repository data on success
//   - [integrations.ErrNotFound] if the repository doesn't exist
//   - [integrations.ErrNetwork] for HTTP failures (timeout, 5xx, rate limits, etc.)
//   - Other errors for JSON decoding failures
//
// The returned RepoMetrics pointer is never nil if err is nil.
// This method is safe for concurrent use.
func (c *Client) Fetch(ctx context.Context, owner, repo string, refresh bool) (*integrations.RepoMetrics, error) {
	key := owner + "/" + repo

	var m integrations.RepoMetrics
	err := c.Cached(ctx, key, refresh, &m, func() error {
		return c.fetchMetrics(ctx, owner, repo, &m)
	})
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *Client) fetchMetrics(ctx context.Context, owner, repo string, m *integrations.RepoMetrics) error {
	data, err := c.fetchRepo(ctx, owner, repo)
	if err != nil {
		return err
	}

	*m = integrations.RepoMetrics{
		RepoURL:  fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		Owner:    owner,
		Stars:    data.Stars,
		SizeKB:   data.Size,
		License:  data.License.SPDXID,
		Language: data.Language,
		Topics:   data.Topics,
		Archived: data.Archived,
	}
	if data.PushedAt != nil {
		m.LastCommitAt = data.PushedAt
	}
	if rel, err := c.fetchRelease(ctx, owner, repo); err == nil {
		m.LastReleaseAt = &rel.PublishedAt
	}
	if contribs, err := c.fetchContributors(ctx, owner, repo); err == nil {
		m.Contributors = contribs
	}
	return nil
}

func (c *Client) fetchRepo(ctx context.Context, owner, repo string) (*repoResponse, error) {
	var data repoResponse
	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repo)
	if err := c.Get(ctx, url, &data); err != nil {
		if errors.Is(err, integrations.ErrNotFound) {
			return nil, fmt.Errorf("%w: github repo %s/%s", err, owner, repo)
		}
		return nil, err
	}
	return &data, nil
}

func (c *Client) fetchRelease(ctx context.Context, owner, repo string) (*releaseResponse, error) {
	var data releaseResponse
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.baseURL, owner, repo)
	if err := c.Get(ctx, url, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *Client) fetchContributors(ctx context.Context, owner, repo string) ([]integrations.Contributor, error) {
	var data []contributorResponse
	url := fmt.Sprintf("%s/repos/%s/%s/contributors?per_page=5", c.baseURL, owner, repo)
	if err := c.Get(ctx, url, &data); err != nil {
		return nil, err
	}

	var result []integrations.Contributor
	for _, cr := range data {
		if cr.Type != "Bot" {
			result = append(result, integrations.Contributor{
				Login:         cr.Login,
				Contributions: cr.Contributions,
			})
		}
	}
	return result, nil
}

// SearchPackageRepo searches GitHub code for a manifest file containing a package name.
//
// This is useful for finding repository URLs when package metadata doesn't include them.
//
// Parameters:
//   - pkgName: Package name to search for (exact match in manifest file)
//   - manifestFile: Manifest filename to search in (e.g., "package.json", "Gemfile")
//
// Example:
//
//	owner, repo, ok := client.SearchPackageRepo(ctx, "fastapi", "pyproject.toml")
//
// Returns:
//   - owner: Repository owner username (empty if not found)
//   - repo: Repository name (empty if not found)
//   - ok: true if a match was found, false otherwise
//
// Search results are always cached (refresh=false) to conserve GitHub API quota.
// This method is safe for concurrent use.
func (c *Client) SearchPackageRepo(ctx context.Context, pkgName, manifestFile string) (owner, repo string, ok bool) {
	key := fmt.Sprintf("search:%s:%s", manifestFile, pkgName)

	var result searchResult
	_ = c.Cached(ctx, key, false, &result, func() error {
		result.Owner, result.Repo, result.Found = c.doSearch(ctx, pkgName, manifestFile)
		return nil
	})
	return result.Owner, result.Repo, result.Found
}

func (c *Client) doSearch(ctx context.Context, pkgName, manifestFile string) (owner, repo string, ok bool) {
	query := fmt.Sprintf(`name = "%s" filename:%s`, pkgName, manifestFile)
	url := fmt.Sprintf("%s/search/code?q=%s&per_page=1", c.baseURL, integrations.URLEncode(query))

	var data searchResponse
	if err := c.Get(ctx, url, &data); err != nil || len(data.Items) == 0 {
		return "", "", false
	}
	item := data.Items[0]
	return item.Repository.Owner.Login, item.Repository.Name, true
}

// ExtractURL extracts GitHub repository owner and name from package URLs.
//
// This function searches through urls map and homepage for GitHub URLs.
// It looks for patterns like "https://github.com/owner/repo".
//
// Parameters:
//   - urls: Map of URL keys to URL values from package metadata (may be nil)
//   - homepage: Fallback homepage URL (may be empty)
//
// Returns:
//   - owner: Repository owner username (empty if not found)
//   - repo: Repository name (empty if not found)
//   - ok: true if a GitHub URL was found, false otherwise
//
// This function is safe for concurrent use.
func ExtractURL(urls map[string]string, homepage string) (owner, repo string, ok bool) {
	return integrations.ExtractRepoURL(repoURLPattern, urls, homepage)
}

type repoResponse struct {
	Stars    int        `json:"stargazers_count"`
	Size     int        `json:"size"`
	PushedAt *time.Time `json:"pushed_at"`
	License  struct {
		SPDXID string `json:"spdx_id"`
	} `json:"license"`
	Language string   `json:"language"`
	Topics   []string `json:"topics"`
	Archived bool     `json:"archived"`
}

type releaseResponse struct {
	PublishedAt time.Time `json:"published_at"`
}

type contributorResponse struct {
	Login         string `json:"login"`
	Contributions int    `json:"contributions"`
	Type          string `json:"type"`
}

type searchResponse struct {
	Items []struct {
		Repository struct {
			Name  string `json:"name"`
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repository"`
	} `json:"items"`
}

type searchResult struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	Found bool   `json:"found"`
}
