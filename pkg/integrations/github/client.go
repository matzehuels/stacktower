package github

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

var repoURLPattern = regexp.MustCompile(`https?://github\.com/([^/]+)/([^/]+)`)

type Client struct {
	integrations.BaseClient
	token   string
	baseURL string
	headers map[string]string
}

func NewClient(token string, cacheTTL time.Duration) (*Client, error) {
	cache, err := integrations.NewCache(cacheTTL)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{"Accept": "application/vnd.github.v3+json"}
	if token != "" {
		headers["Authorization"] = "Bearer " + token
	}

	return &Client{
		BaseClient: integrations.BaseClient{
			HTTP:  integrations.NewHTTPClient(),
			Cache: cache,
		},
		token:   token,
		baseURL: "https://api.github.com",
		headers: headers,
	}, nil
}

func (c *Client) Fetch(ctx context.Context, owner, repo string, refresh bool) (*integrations.RepoMetrics, error) {
	cacheKey := "github:" + owner + "/" + repo

	var m integrations.RepoMetrics
	err := c.FetchWithCache(ctx, cacheKey, refresh, func() error {
		return c.fetchMetrics(ctx, owner, repo, &m)
	}, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *Client) fetchMetrics(ctx context.Context, owner, repo string, m *integrations.RepoMetrics) error {
	repoData, err := c.fetchRepo(ctx, owner, repo)
	if err != nil {
		return err
	}

	*m = integrations.RepoMetrics{
		RepoURL:  fmt.Sprintf("https://github.com/%s/%s", owner, repo),
		Owner:    owner,
		Stars:    repoData.Stars,
		SizeKB:   repoData.Size,
		License:  repoData.License.SPDXID,
		Language: repoData.Language,
		Topics:   repoData.Topics,
		Archived: repoData.Archived,
	}
	if repoData.PushedAt != nil {
		m.LastCommitAt = repoData.PushedAt
	}
	if release, err := c.fetchLatestRelease(ctx, owner, repo); err == nil {
		m.LastReleaseAt = &release.PublishedAt
	}
	if contributors, err := c.fetchContributors(ctx, owner, repo); err == nil {
		m.Contributors = contributors
	}
	return nil
}

func (c *Client) fetchRepo(ctx context.Context, owner, repo string) (*repoResponse, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repo)

	var data repoResponse
	if err := c.DoRequest(ctx, url, c.headers, &data); err != nil {
		if errors.Is(err, integrations.ErrNotFound) {
			return nil, fmt.Errorf("%w: github repo %s/%s", err, owner, repo)
		}
		return nil, err
	}
	return &data, nil
}

func (c *Client) fetchLatestRelease(ctx context.Context, owner, repo string) (*releaseResponse, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.baseURL, owner, repo)

	var data releaseResponse
	if err := c.DoRequest(ctx, url, c.headers, &data); err != nil {
		return nil, fmt.Errorf("no releases")
	}
	return &data, nil
}

func (c *Client) fetchContributors(ctx context.Context, owner, repo string) ([]integrations.Contributor, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contributors?per_page=5", c.baseURL, owner, repo)

	var data []contributorResponse
	if err := c.DoRequest(ctx, url, c.headers, &data); err != nil {
		return nil, fmt.Errorf("no contributors")
	}

	contributors := make([]integrations.Contributor, 0, len(data))
	for _, c := range data {
		if c.Type != "Bot" {
			contributors = append(contributors, integrations.Contributor{
				Login:         c.Login,
				Contributions: c.Contributions,
			})
		}
	}
	return contributors, nil
}

func (c *Client) SearchPackageRepo(ctx context.Context, pkgName, manifestFile string) (owner, repo string, ok bool) {
	if c.token == "" {
		return "", "", false
	}

	cacheKey := fmt.Sprintf("github:search:%s:%s", manifestFile, pkgName)

	var result searchCacheEntry
	err := c.FetchWithCache(ctx, cacheKey, false, func() error {
		o, r, found := c.doCodeSearch(ctx, pkgName, manifestFile)
		result = searchCacheEntry{Owner: o, Repo: r, Found: found}
		return nil
	}, &result)

	if err != nil || !result.Found {
		return "", "", false
	}
	return result.Owner, result.Repo, true
}

func (c *Client) doCodeSearch(ctx context.Context, pkgName, manifestFile string) (owner, repo string, ok bool) {
	query := fmt.Sprintf(`name = "%s" filename:%s`, pkgName, manifestFile)
	searchURL := fmt.Sprintf("%s/search/code?q=%s&per_page=1", c.baseURL, integrations.URLEncode(query))

	var data codeSearchResponse
	if err := c.DoRequest(ctx, searchURL, c.headers, &data); err != nil {
		return "", "", false
	}

	if len(data.Items) == 0 {
		return "", "", false
	}

	return data.Items[0].Repository.Owner.Login, data.Items[0].Repository.Name, true
}

func ExtractURL(projectURLs map[string]string, homepage string) (owner, repo string, ok bool) {
	return integrations.ExtractRepoURL(repoURLPattern, projectURLs, homepage)
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

type codeSearchResponse struct {
	Items []struct {
		Repository struct {
			Name  string `json:"name"`
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repository"`
	} `json:"items"`
}

type searchCacheEntry struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	Found bool   `json:"found"`
}
