package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// ContentClient provides access to GitHub repository content.
// Use this for fetching files, listing directories, and user operations.
type ContentClient struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewContentClient creates a new content client with the given access token.
func NewContentClient(token string) *ContentClient {
	return &ContentClient{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.github.com",
	}
}

// FetchUser retrieves the authenticated user's info.
func (c *ContentClient) FetchUser(ctx context.Context) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/user", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, string(body))
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &user, nil
}

// FetchUserRepos retrieves all of the authenticated user's repositories.
// This includes private repos if the OAuth token has the 'repo' scope.
// Results are paginated automatically to retrieve all repos.
func (c *ContentClient) FetchUserRepos(ctx context.Context) ([]Repo, error) {
	var allRepos []Repo
	page := 1

	for {
		url := fmt.Sprintf("%s/user/repos?sort=updated&per_page=100&page=%d", c.baseURL, page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		c.setHeaders(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("send request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, string(body))
		}

		var repos []Repo
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode response: %w", err)
		}
		resp.Body.Close()

		if len(repos) == 0 {
			break // No more pages
		}

		allRepos = append(allRepos, repos...)
		page++

		// Safety limit to avoid infinite loops
		if page > 10 {
			break
		}
	}

	return allRepos, nil
}

// ListContents lists files and directories in a repository path.
func (c *ContentClient) ListContents(ctx context.Context, owner, repo, path string) ([]ContentItem, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.baseURL, owner, repo, path)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, string(body))
	}

	var items []apiContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result := make([]ContentItem, len(items))
	for i, item := range items {
		result[i] = ContentItem{
			Name: item.Name,
			Path: item.Path,
			Type: item.Type,
			Size: item.Size,
		}
	}

	return result, nil
}

// FetchFile retrieves the content of a file from a repository.
// The content is returned as a string (decoded from base64).
func (c *ContentClient) FetchFile(ctx context.Context, owner, repo, path string) (*FileContent, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.baseURL, owner, repo, path)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, string(body))
	}

	var fileResp apiContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&fileResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Decode base64 content
	content, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(fileResp.Content, "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("decode content: %w", err)
	}

	return &FileContent{
		Path:    fileResp.Path,
		Size:    fileResp.Size,
		Content: string(content),
	}, nil
}

// FetchFileRaw retrieves the raw content of a file from a repository.
// This is more efficient for large files as it doesn't use base64 encoding.
func (c *ContentClient) FetchFileRaw(ctx context.Context, owner, repo, path string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.baseURL, owner, repo, path)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3.raw") // Get raw content

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, string(body))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read content: %w", err)
	}

	return string(content), nil
}

// DetectManifests finds manifest files in a repository's root directory.
// The patterns map filename -> language name (e.g., "go.mod" -> "go").
// Use deps.SupportedManifests(languages) to get patterns from the deps package.
func (c *ContentClient) DetectManifests(ctx context.Context, owner, repo string, patterns map[string]string) ([]ManifestFile, error) {
	items, err := c.ListContents(ctx, owner, repo, "")
	if err != nil {
		return nil, err
	}

	var manifests []ManifestFile
	for _, item := range items {
		if item.Type == "file" {
			if lang, ok := patterns[item.Name]; ok {
				manifests = append(manifests, ManifestFile{
					Path:     item.Path,
					Language: lang,
					Name:     item.Name,
				})
			}
		}
	}

	return manifests, nil
}

// setHeaders sets common headers for GitHub API requests.
func (c *ContentClient) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

// SearchCode searches for code in a repository.
// Query follows GitHub code search syntax: https://docs.github.com/en/search-github/searching-on-github/searching-code
func (c *ContentClient) SearchCode(ctx context.Context, owner, repo, query string) ([]CodeSearchResult, error) {
	// Build search query with repo filter
	fullQuery := fmt.Sprintf("%s repo:%s/%s", query, owner, repo)
	url := fmt.Sprintf("%s/search/code?q=%s&per_page=20", c.baseURL, urlEncode(fullQuery))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, string(body))
	}

	var searchResp codeSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	results := make([]CodeSearchResult, len(searchResp.Items))
	for i, item := range searchResp.Items {
		results[i] = CodeSearchResult{
			Name: item.Name,
			Path: item.Path,
		}
		// Extract text matches if available
		for _, match := range item.TextMatches {
			results[i].Matches = append(results[i].Matches, match.Fragment)
		}
	}

	return results, nil
}

// GetTree retrieves the full file tree of a repository.
func (c *ContentClient) GetTree(ctx context.Context, owner, repo, branch string) ([]TreeEntry, error) {
	if branch == "" {
		branch = "HEAD"
	}
	url := fmt.Sprintf("%s/repos/%s/%s/git/trees/%s?recursive=1", c.baseURL, owner, repo, branch)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, string(body))
	}

	var treeResp treeResponse
	if err := json.NewDecoder(resp.Body).Decode(&treeResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	entries := make([]TreeEntry, 0, len(treeResp.Tree))
	for _, item := range treeResp.Tree {
		entries = append(entries, TreeEntry{
			Path: item.Path,
			Type: item.Type,
			Size: item.Size,
		})
	}

	return entries, nil
}

// GetRepoInfo retrieves repository metadata.
func (c *ContentClient) GetRepoInfo(ctx context.Context, owner, repo string) (*RepoInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, string(body))
	}

	var repoResp apiRepoResponse
	if err := json.NewDecoder(resp.Body).Decode(&repoResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &RepoInfo{
		Name:          repoResp.Name,
		FullName:      repoResp.FullName,
		Description:   repoResp.Description,
		Language:      repoResp.Language,
		DefaultBranch: repoResp.DefaultBranch,
		Stars:         repoResp.Stars,
		Forks:         repoResp.Forks,
		OpenIssues:    repoResp.OpenIssues,
		License:       repoResp.License.SPDXID,
		Topics:        repoResp.Topics,
		Archived:      repoResp.Archived,
	}, nil
}

func urlEncode(s string) string {
	// Simple URL encoding for search queries
	replacer := strings.NewReplacer(
		" ", "+",
		":", "%3A",
		"/", "%2F",
	)
	return replacer.Replace(s)
}

// CodeSearchResult represents a code search match.
type CodeSearchResult struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Matches []string `json:"matches,omitempty"`
}

// TreeEntry represents a file or directory in the repository tree.
type TreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"` // "blob" or "tree"
	Size int    `json:"size,omitempty"`
}

// RepoInfo contains repository metadata.
type RepoInfo struct {
	Name          string   `json:"name"`
	FullName      string   `json:"full_name"`
	Description   string   `json:"description"`
	Language      string   `json:"language"`
	DefaultBranch string   `json:"default_branch"`
	Stars         int      `json:"stars"`
	Forks         int      `json:"forks"`
	OpenIssues    int      `json:"open_issues"`
	License       string   `json:"license"`
	Topics        []string `json:"topics"`
	Archived      bool     `json:"archived"`
}

type codeSearchResponse struct {
	TotalCount int `json:"total_count"`
	Items      []struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		TextMatches []struct {
			Fragment string `json:"fragment"`
		} `json:"text_matches"`
	} `json:"items"`
}

type treeResponse struct {
	Tree []struct {
		Path string `json:"path"`
		Type string `json:"type"`
		Size int    `json:"size"`
	} `json:"tree"`
	Truncated bool `json:"truncated"`
}

// DetectLanguageFromManifest determines the language from a manifest filename.
func DetectLanguageFromManifest(path string) string {
	name := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		name = path[idx+1:]
	}

	switch name {
	case "package.json", "package-lock.json":
		return "javascript"
	case "requirements.txt", "setup.py", "pyproject.toml", "Pipfile":
		return "python"
	case "Cargo.toml":
		return "rust"
	case "go.mod":
		return "go"
	case "Gemfile":
		return "ruby"
	case "composer.json":
		return "php"
	case "pom.xml", "build.gradle":
		return "java"
	default:
		return ""
	}
}

// ScanReposForManifests fetches repos and detects manifests in parallel.
// It returns repos sorted by UpdatedAt (most recent first).
// The manifestPatterns map should be filename -> language (use deps.SupportedManifests).
// Set publicOnly=true to filter out private repos.
func (c *ContentClient) ScanReposForManifests(ctx context.Context, manifestPatterns map[string]string, publicOnly bool) ([]RepoWithManifests, error) {
	repos, err := c.FetchUserRepos(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch repos: %w", err)
	}

	// Sort by most recently updated
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].UpdatedAt > repos[j].UpdatedAt
	})

	// Filter private repos if requested
	if publicOnly {
		var filtered []Repo
		for _, r := range repos {
			if !r.Private {
				filtered = append(filtered, r)
			}
		}
		repos = filtered
	}

	if len(repos) == 0 {
		return nil, nil
	}

	// Parallel manifest detection with worker pool
	type repoResult struct {
		idx       int
		repo      Repo
		manifests []ManifestFile
	}

	results := make([]repoResult, len(repos))
	var wg sync.WaitGroup

	// Semaphore for concurrency limit (10 parallel requests)
	sem := make(chan struct{}, 10)

	for i, r := range repos {
		wg.Add(1)
		go func(idx int, repo Repo) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			parts := strings.SplitN(repo.FullName, "/", 2)
			var manifests []ManifestFile
			if len(parts) == 2 {
				manifests, _ = c.DetectManifests(ctx, parts[0], parts[1], manifestPatterns)
			}
			results[idx] = repoResult{idx: idx, repo: repo, manifests: manifests}
		}(i, r)
	}

	wg.Wait()

	// Build final list preserving order
	rwm := make([]RepoWithManifests, len(repos))
	for _, r := range results {
		rwm[r.idx] = RepoWithManifests{Repo: r.repo, Manifests: r.manifests}
	}

	return rwm, nil
}
