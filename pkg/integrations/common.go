package integrations

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"stacktower/pkg/httputil"
)

const httpTimeout = 10 * time.Second

var (
	ErrNotFound = errors.New("resource not found")
	ErrNetwork  = errors.New("network error")
)

type RepoMetrics struct {
	RepoURL       string        `json:"repo_url"`
	Owner         string        `json:"owner"`
	Stars         int           `json:"stars"`
	SizeKB        int           `json:"size_kb,omitempty"`
	LastCommitAt  *time.Time    `json:"last_commit_at,omitempty"`
	LastReleaseAt *time.Time    `json:"last_release_at,omitempty"`
	License       string        `json:"license,omitempty"`
	Contributors  []Contributor `json:"top_contributors,omitempty"`
	Language      string        `json:"language,omitempty"`
	Topics        []string      `json:"topics,omitempty"`
	Archived      bool          `json:"archived"`
}

type Contributor struct {
	Login         string `json:"login"`
	Contributions int    `json:"contributions"`
}

var repoURLKeys = []string{"Source", "Repository", "Code", "Homepage"}

func ExtractRepoURL(re *regexp.Regexp, projectURLs map[string]string, homepage string) (owner, repo string, ok bool) {
	for _, key := range repoURLKeys {
		if url, exists := projectURLs[key]; exists {
			if owner, repo, ok = matchRepoURL(re, url); ok {
				return owner, repo, true
			}
		}
	}
	for _, url := range projectURLs {
		if owner, repo, ok = matchRepoURL(re, url); ok {
			return owner, repo, true
		}
	}
	if homepage != "" {
		if owner, repo, ok = matchRepoURL(re, homepage); ok {
			return owner, repo, true
		}
	}
	return "", "", false
}

func matchRepoURL(re *regexp.Regexp, url string) (owner, repo string, ok bool) {
	if strings.Contains(url, "/sponsors/") {
		return "", "", false
	}
	if match := re.FindStringSubmatch(url); len(match) >= 3 {
		return match[1], match[2], true
	}
	return "", "", false
}

func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: httpTimeout}
}

func NewCache(ttl time.Duration) (*httputil.Cache, error) {
	return httputil.NewCache("", ttl)
}

func URLEncode(s string) string {
	return url.QueryEscape(s)
}
