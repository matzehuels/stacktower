// Package packagist provides an HTTP client for the Packagist API.
//
// # Overview
//
// This package fetches package metadata from Packagist (https://packagist.org),
// the main Composer repository for PHP packages.
//
// # Usage
//
//	client, err := packagist.NewClient(24 * time.Hour)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	pkg, err := client.FetchPackage(ctx, "symfony/console", false)
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
//   - Name, Version: Package identity (latest stable version)
//   - Dependencies: Composer "require" dependencies (filtered)
//   - Description: Package description
//   - License, Author: Package metadata
//   - Repository, HomePage: URLs for enrichment
//
// # Caching
//
// Responses are cached to reduce load on Packagist. The cache TTL is set
// when creating the client. Pass refresh=true to bypass the cache.
//
// # Dependency Filtering
//
// The following are filtered from dependencies:
//
//   - php, composer-plugin-api, composer-runtime-api
//   - ext-* (PHP extensions)
//   - lib-* (system libraries)
//   - Packages without a "/" (non-Composer packages)
//
// # Version Selection
//
// The client selects the latest stable version, skipping dev versions.
// If no stable version exists, the first version is used.
package packagist
