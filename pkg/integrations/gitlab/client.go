package gitlab

import (
	"regexp"
	"time"

	"stacktower/pkg/integrations"
)

var repoURLPattern = regexp.MustCompile(`https?://gitlab\.com/([^/]+)/([^/]+)`)

type Client struct {
	integrations.BaseClient
	token   string
	baseURL string
}

func NewClient(token string, cacheTTL time.Duration) (*Client, error) {
	cache, err := integrations.NewCache(cacheTTL)
	if err != nil {
		return nil, err
	}
	return &Client{
		BaseClient: integrations.BaseClient{
			HTTP:  integrations.NewHTTPClient(),
			Cache: cache,
		},
		token:   token,
		baseURL: "https://gitlab.com/api/v4",
	}, nil
}

func ExtractURL(projectURLs map[string]string, homepage string) (owner, repo string, ok bool) {
	return integrations.ExtractRepoURL(repoURLPattern, projectURLs, homepage)
}
