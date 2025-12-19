// Package rubygems provides an HTTP client for the RubyGems.org API.
//
// # Overview
//
// This package fetches gem metadata from RubyGems.org (https://rubygems.org),
// the Ruby community's gem hosting service.
//
// # Usage
//
//	client, err := rubygems.NewClient(24 * time.Hour)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	gem, err := client.FetchGem(ctx, "rails", false)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println(gem.Name, gem.Version)
//	fmt.Println("Dependencies:", gem.Dependencies)
//
// # GemInfo
//
// [FetchGem] returns a [GemInfo] containing:
//
//   - Name, Version: Gem identity
//   - Dependencies: Runtime dependencies only
//   - Description: Gem info/summary
//   - License: License string (may be comma-separated)
//   - SourceCodeURI, HomepageURI: URLs for enrichment
//   - Downloads: Total download count
//   - Authors: Author names
//
// # Caching
//
// Responses are cached to reduce load on RubyGems. The cache TTL is set
// when creating the client. Pass refresh=true to bypass the cache.
//
// # Dependency Filtering
//
// Only runtime dependencies are included. Development dependencies are
// filtered out. Gem names are normalized to lowercase.
package rubygems
