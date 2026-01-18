// Package integrations provides HTTP clients for package registry APIs.
//
// # Overview
//
// This package contains low-level API clients for fetching package metadata
// from various registries. Each registry has its own subpackage:
//
//   - [pypi]: Python Package Index
//   - [npm]: Node Package Manager
//   - [crates]: Rust crates.io
//   - [rubygems]: Ruby gems
//   - [packagist]: PHP Composer packages
//   - [maven]: Java Maven Central
//   - [goproxy]: Go Module Proxy
//   - [github]: GitHub API for metadata enrichment
//   - [gitlab]: GitLab API for metadata enrichment
//
// # Client Pattern
//
// All registry clients follow a consistent pattern:
//
//	client, err := pypi.NewClient(24 * time.Hour)  // Cache TTL
//	pkg, err := client.FetchPackage(ctx, "fastapi", false)  // false = use cache
//
// Clients handle:
//   - HTTP requests with retry and rate limiting
//   - Response caching (file-based, configurable TTL)
//   - API-specific parsing and normalization
//
// # Shared Infrastructure
//
// The [Client] type provides shared HTTP functionality used by all registry
// clients, including HTTP response caching via [cache.Cache].
//
// # Adding a New Registry
//
// To add support for a new package registry:
//
//  1. Create a subpackage: pkg/integrations/<registry>/
//  2. Define response structs matching the API schema
//  3. Implement a Client with FetchPackage method
//  4. Use [NewClient] for HTTP with caching
//  5. Wire into [deps] as a new language
//
// [pypi]: github.com/matzehuels/stacktower/pkg/integrations/pypi
// [npm]: github.com/matzehuels/stacktower/pkg/integrations/npm
// [crates]: github.com/matzehuels/stacktower/pkg/integrations/crates
// [rubygems]: github.com/matzehuels/stacktower/pkg/integrations/rubygems
// [packagist]: github.com/matzehuels/stacktower/pkg/integrations/packagist
// [maven]: github.com/matzehuels/stacktower/pkg/integrations/maven
// [goproxy]: github.com/matzehuels/stacktower/pkg/integrations/goproxy
// [github]: github.com/matzehuels/stacktower/pkg/integrations/github
// [gitlab]: github.com/matzehuels/stacktower/pkg/integrations/gitlab
// [cache.Cache]: github.com/matzehuels/stacktower/pkg/cache.Cache
// [deps]: github.com/matzehuels/stacktower/pkg/core/deps
package integrations
