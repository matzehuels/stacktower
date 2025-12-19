// Package npm provides an HTTP client for the npm registry API.
//
// # Overview
//
// This package fetches package metadata from the npm registry
// (https://registry.npmjs.org), the package manager for JavaScript.
//
// # Usage
//
//	client, err := npm.NewClient(24 * time.Hour)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	pkg, err := client.FetchPackage(ctx, "express", false)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println(pkg.Name, pkg.Version)
//	fmt.Println("Dependencies:", pkg.Dependencies)
//
// # PackageInfo
//
// [FetchPackage] returns a [PackageInfo] containing:
//
//   - Name, Version: Package identity (latest version)
//   - Dependencies: Runtime dependencies from "dependencies" field
//   - Description: Package description
//   - License, Author: Package metadata
//   - Repository, HomePage: URLs for enrichment
//
// # Caching
//
// Responses are cached to reduce load on the registry. The cache TTL is set
// when creating the client. Pass refresh=true to bypass the cache.
//
// # Version Selection
//
// The client fetches the version tagged as "latest" in dist-tags.
// devDependencies, peerDependencies, and optionalDependencies are not included.
package npm
