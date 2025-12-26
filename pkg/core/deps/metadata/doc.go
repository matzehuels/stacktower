// Package metadata provides metadata enrichment from external sources.
//
// # Overview
//
// Package registries provide basic information (name, version, dependencies),
// but Stacktower's analysis features need additional data from source
// repositories. This package provides [deps.MetadataProvider] implementations
// that fetch supplementary metadata from GitHub and other sources.
//
// # GitHub Provider
//
// The [GitHub] provider enriches packages with repository data:
//
//   - Stars count (for popularity ranking)
//   - Owner username (for Nebraska maintainer analysis)
//   - Contributors list (for bus factor assessment)
//   - Last commit/release dates (for staleness detection)
//   - Archived status (for brittle package detection)
//
// Usage:
//
//	provider, err := metadata.NewGitHub(token, 24*time.Hour)
//	opts := deps.Options{MetadataProviders: []deps.MetadataProvider{provider}}
//	g, err := resolver.Resolve(ctx, "fastapi", opts)
//
// The provider automatically extracts GitHub URLs from package metadata
// (ProjectURLs, Repository, HomePage) or falls back to GitHub search.
//
// # Metadata Keys
//
// Enriched data is stored in node metadata using these standard keys:
//
//   - [RepoURL]: Repository URL
//   - [RepoOwner]: Repository owner username
//   - [RepoStars]: Star count
//   - [RepoArchived]: Whether the repo is archived
//   - [RepoMaintainers]: List of top contributor usernames
//   - [RepoLastCommit]: Date of last commit (YYYY-MM-DD)
//   - [RepoLastRelease]: Date of last release (YYYY-MM-DD)
//   - [RepoLanguage]: Primary repository language
//   - [RepoTopics]: Repository topic tags
//
// # Composite Provider
//
// [Composite] combines multiple providers, merging their results:
//
//	providers := metadata.NewComposite(
//	    github,
//	    gitlab,
//	)
//
// [deps.MetadataProvider]: github.com/matzehuels/stacktower/pkg/core/deps.MetadataProvider
package metadata
