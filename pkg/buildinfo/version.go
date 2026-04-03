// Package buildinfo provides build-time version information.
//
// Variables are set via ldflags during build:
//
//	go build -ldflags "-X github.com/matzehuels/stacktower/pkg/buildinfo.Version=v1.0.0 \
//	    -X github.com/matzehuels/stacktower/pkg/buildinfo.Commit=$(git rev-parse HEAD) \
//	    -X github.com/matzehuels/stacktower/pkg/buildinfo.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
package buildinfo

import "fmt"

var (
	// Version is the semantic version (e.g., "v1.2.3").
	// Set via ldflags: -X github.com/matzehuels/stacktower/pkg/buildinfo.Version=...
	Version = "dev"

	// Commit is the git commit SHA.
	// Set via ldflags: -X github.com/matzehuels/stacktower/pkg/buildinfo.Commit=...
	Commit = "none"

	// Date is the build timestamp.
	// Set via ldflags: -X github.com/matzehuels/stacktower/pkg/buildinfo.Date=...
	Date = "unknown"

	// GitHubAppClientID is the OAuth client ID for GitHub device flow authentication.
	GitHubAppClientID = "Iv23liGZy2ueQyYDFD6q"

	// GitHubAppSlug is the GitHub App slug for installation URLs.
	// Set via ldflags: -X github.com/matzehuels/stacktower/pkg/buildinfo.GitHubAppSlug=...
	GitHubAppSlug = "stacktower"
)

// String returns the formatted build information.
func String() string {
	return fmt.Sprintf("version: %s\ncommit: %s\nbuilt: %s", Version, Commit, Date)
}

// Template returns the version template string for cobra.
func Template() string {
	return fmt.Sprintf("{{.Name}} version %s\ncommit: %s\nbuilt: %s\n", Version, Commit, Date)
}
