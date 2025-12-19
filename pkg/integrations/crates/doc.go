// Package crates provides an HTTP client for the crates.io API.
//
// # Overview
//
// This package fetches crate metadata from crates.io (https://crates.io),
// the Rust community's package registry.
//
// # Usage
//
//	client, err := crates.NewClient(24 * time.Hour)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	crate, err := client.FetchCrate(ctx, "serde", false)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println(crate.Name, crate.Version)
//	fmt.Println("Dependencies:", crate.Dependencies)
//
// # CrateInfo
//
// [FetchCrate] returns a [CrateInfo] containing:
//
//   - Name, Version: Crate identity (max_version from API)
//   - Dependencies: Normal (non-optional, non-dev) dependencies
//   - Description: Crate description
//   - License: SPDX license identifier
//   - Repository, HomePage: URLs for enrichment
//   - Downloads: Total download count
//
// # Caching
//
// Responses are cached to reduce load on crates.io. The cache TTL is set
// when creating the client. Pass refresh=true to bypass the cache.
//
// # Dependency Filtering
//
// Only "normal" dependencies are included. Development dependencies,
// build dependencies, and optional dependencies are filtered out.
//
// # User-Agent
//
// The client includes a User-Agent header as requested by crates.io policy.
package crates
