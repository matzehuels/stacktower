package maven

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/matzehuels/stacktower/pkg/integrations"
)

// ArtifactInfo holds metadata for a Java artifact from Maven Central.
//
// Artifacts are identified by "groupId:artifactId" coordinates.
// Dependencies include only compile-scope dependencies; test, provided, and optional deps are excluded.
// Dependencies with unresolved Maven properties (${...}) are skipped.
//
// Zero values: All string fields are empty, Dependencies is nil.
// This struct is safe for concurrent reads after construction.
type ArtifactInfo struct {
	GroupID      string   // Maven groupId (e.g., "com.google.guava", never empty in valid info)
	ArtifactID   string   // Maven artifactId (e.g., "guava", never empty in valid info)
	Version      string   // Latest version (e.g., "32.1.3-jre", never empty in valid info)
	Dependencies []string // Compile-scope dependency coordinates (nil or empty if none or POM fetch failed)
	Description  string   // Artifact description from POM (may be empty)
	URL          string   // URL to the POM file on Maven Central (never empty in valid info)
}

// Coordinate returns the Maven coordinate string "groupId:artifactId".
// Example: "com.google.guava:guava"
func (a *ArtifactInfo) Coordinate() string {
	return a.GroupID + ":" + a.ArtifactID
}

// Client provides access to the Maven Central repository API.
// It handles HTTP requests with caching and automatic retries.
//
// All methods are safe for concurrent use by multiple goroutines.
type Client struct {
	*integrations.Client
	baseURL string
}

// NewClient creates a Maven Central client with the specified cache TTL.
//
// The cacheTTL parameter sets how long responses are cached.
// Typical values: 1-24 hours for production, 0 for testing (no cache).
//
// Returns an error if the cache directory cannot be created or accessed.
// The returned Client is safe for concurrent use.
func NewClient(cacheTTL time.Duration) (*Client, error) {
	cache, err := integrations.NewCacheWithNamespace("maven:", cacheTTL)
	if err != nil {
		return nil, err
	}
	return &Client{
		Client:  integrations.NewClient(cache, nil),
		baseURL: "https://search.maven.org/solrsearch/select",
	}, nil
}

// FetchArtifact retrieves metadata for a Java artifact from Maven Central.
//
// The coordinate parameter must be in the format "groupId:artifactId".
// Examples: "com.google.guava:guava", "org.apache.commons:commons-lang3"
// Coordinate cannot be empty or missing the colon separator.
//
// If refresh is true, the cache is bypassed and a fresh API call is made.
// If refresh is false, cached data is returned if available and not expired.
//
// This method performs two API calls:
//  1. Maven Central Search API to find the latest version
//  2. Direct POM fetch to extract dependencies
//
// POM fetch failures are silently ignored; Dependencies will be empty/nil if it fails.
//
// Returns:
//   - ArtifactInfo populated with metadata on success
//   - [integrations.ErrNotFound] if the artifact doesn't exist
//   - [integrations.ErrNetwork] for HTTP failures (timeout, 5xx, etc.)
//   - Error if coordinate format is invalid
//   - Other errors for JSON decoding failures
//
// The returned ArtifactInfo pointer is never nil if err is nil.
// This method is safe for concurrent use.
func (c *Client) FetchArtifact(ctx context.Context, coordinate string, refresh bool) (*ArtifactInfo, error) {
	groupID, artifactID, err := parseCoordinate(coordinate)
	if err != nil {
		return nil, err
	}

	key := coordinate

	var info ArtifactInfo
	err = c.Cached(ctx, key, refresh, &info, func() error {
		return c.fetch(ctx, groupID, artifactID, &info)
	})
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *Client) fetch(ctx context.Context, groupID, artifactID string, info *ArtifactInfo) error {
	// First, get the latest version
	query := fmt.Sprintf("g:%q AND a:%q", groupID, artifactID)
	url := fmt.Sprintf("%s?q=%s&rows=1&wt=json", c.baseURL, integrations.URLEncode(query))

	var searchResp searchResponse
	if err := c.Get(ctx, url, &searchResp); err != nil {
		if errors.Is(err, integrations.ErrNotFound) {
			return fmt.Errorf("%w: maven artifact %s:%s", err, groupID, artifactID)
		}
		return err
	}

	if searchResp.Response.NumFound == 0 {
		return fmt.Errorf("%w: maven artifact %s:%s", integrations.ErrNotFound, groupID, artifactID)
	}

	doc := searchResp.Response.Docs[0]
	version := doc.LatestVersion
	if version == "" {
		version = doc.Version
	}

	// Fetch POM to get dependencies
	deps, pomURL := c.fetchPOMDeps(ctx, groupID, artifactID, version)

	*info = ArtifactInfo{
		GroupID:      groupID,
		ArtifactID:   artifactID,
		Version:      version,
		Dependencies: deps,
		URL:          pomURL,
	}
	return nil
}

func (c *Client) fetchPOMDeps(ctx context.Context, groupID, artifactID, version string) ([]string, string) {
	groupPath := strings.ReplaceAll(groupID, ".", "/")
	pomURL := fmt.Sprintf("https://repo1.maven.org/maven2/%s/%s/%s/%s-%s.pom",
		groupPath, artifactID, version, artifactID, version)

	pom, err := c.fetchPOM(ctx, pomURL)
	if err != nil {
		return nil, pomURL
	}

	return extractDeps(pom), pomURL
}

func (c *Client) fetchPOM(ctx context.Context, url string) (*pomProject, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pom fetch failed: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pom pomProject
	if err := xml.Unmarshal(data, &pom); err != nil {
		return nil, err
	}
	return &pom, nil
}

func extractDeps(pom *pomProject) []string {
	var deps []string
	seen := make(map[string]bool)

	for _, dep := range pom.Dependencies {
		if dep.Scope == "test" || dep.Scope == "provided" || dep.Optional == "true" {
			continue
		}
		// Skip dependencies with unresolved properties
		if strings.HasPrefix(dep.GroupID, "${") || strings.HasPrefix(dep.ArtifactID, "${") {
			continue
		}
		coord := dep.GroupID + ":" + dep.ArtifactID
		if !seen[coord] {
			seen[coord] = true
			deps = append(deps, coord)
		}
	}
	return deps
}

func parseCoordinate(coord string) (groupID, artifactID string, err error) {
	parts := strings.Split(coord, ":")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid maven coordinate %q (expected groupId:artifactId)", coord)
	}
	return parts[0], parts[1], nil
}

type searchResponse struct {
	Response struct {
		NumFound int         `json:"numFound"`
		Docs     []searchDoc `json:"docs"`
	} `json:"response"`
}

type searchDoc struct {
	GroupID       string `json:"g"`
	ArtifactID    string `json:"a"`
	Version       string `json:"v"`
	LatestVersion string `json:"latestVersion"`
}

type pomProject struct {
	GroupID      string          `xml:"groupId"`
	ArtifactID   string          `xml:"artifactId"`
	Version      string          `xml:"version"`
	Name         string          `xml:"name"`
	Description  string          `xml:"description"`
	URL          string          `xml:"url"`
	Dependencies []pomDependency `xml:"dependencies>dependency"`
}

type pomDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Optional   string `xml:"optional"`
}
