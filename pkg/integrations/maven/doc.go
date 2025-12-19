// Package maven provides an HTTP client for Maven Central.
//
// # Overview
//
// This package fetches artifact metadata from Maven Central
// (https://search.maven.org), the primary repository for Java packages.
//
// # Usage
//
//	client, err := maven.NewClient(24 * time.Hour)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	artifact, err := client.FetchArtifact(ctx, "com.google.guava:guava", false)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println(artifact.Coordinate())
//	fmt.Println("Dependencies:", artifact.Dependencies)
//
// # Coordinates
//
// Maven artifacts are identified by coordinates in the format "groupId:artifactId".
// For example: "com.google.guava:guava", "org.apache.commons:commons-lang3".
//
// # ArtifactInfo
//
// [FetchArtifact] returns an [ArtifactInfo] containing:
//
//   - GroupID, ArtifactID: Artifact identity
//   - Version: Latest version from Maven Central search
//   - Dependencies: Compile-scope dependencies from POM
//   - Description: Project description from POM
//   - URL: Link to the POM file
//
// # Caching
//
// Responses are cached to reduce load on Maven Central. The cache TTL is set
// when creating the client. Pass refresh=true to bypass the cache.
//
// # Dependency Filtering
//
// Only compile-scope dependencies are included. Test, provided, and optional
// dependencies are filtered out. Dependencies with unresolved Maven properties
// (${...}) are skipped.
//
// # Two-Phase Fetch
//
// The client performs two requests:
//  1. Maven Central Search API to find the latest version
//  2. Direct POM fetch from repo1.maven.org for dependencies
package maven
