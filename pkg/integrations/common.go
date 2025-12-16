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

func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: httpTimeout}
}

func NewCache(ttl time.Duration) (*httputil.Cache, error) {
	return httputil.NewCache("", ttl)
}

func NormalizePkgName(name string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "_", "-")
}

var repoURLReplacer = strings.NewReplacer(
	"git@github.com:", "https://github.com/",
	"git://github.com/", "https://github.com/",
)

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

func URLEncode(s string) string { return url.QueryEscape(s) }
