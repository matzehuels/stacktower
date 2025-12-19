// Package gitlab provides an HTTP client for the GitLab API.
//
// # Overview
//
// This package provides GitLab integration for metadata enrichment,
// complementing the GitHub provider for packages hosted on GitLab.
//
// # Usage
//
//	client, err := gitlab.NewClient(token, 24 * time.Hour)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Authentication
//
// A GitLab personal access token is optional. Without a token, only
// public repositories can be accessed.
//
// # URL Extraction
//
// [ExtractURL] parses GitLab repository URLs from package metadata:
//
//	owner, repo, ok := gitlab.ExtractURL(pkg.ProjectURLs, pkg.HomePage)
//	if ok {
//	    // Found GitLab repository: gitlab.com/owner/repo
//	}
//
// This is useful for identifying GitLab-hosted packages and fetching
// their repository metrics.
//
// # Current Limitations
//
// The GitLab client currently focuses on URL extraction. Full metrics
// fetching (stars, contributors, etc.) is planned for future releases.
package gitlab
