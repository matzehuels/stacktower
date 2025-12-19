// Package goproxy provides an HTTP client for the Go Module Proxy.
//
// # Overview
//
// This package fetches module metadata from the Go Module Proxy
// (https://proxy.golang.org), the default proxy for Go modules.
//
// # Usage
//
//	client, err := goproxy.NewClient(24 * time.Hour)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	mod, err := client.FetchModule(ctx, "github.com/spf13/cobra", false)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println(mod.Path, mod.Version)
//	fmt.Println("Dependencies:", mod.Dependencies)
//
// # ModuleInfo
//
// [FetchModule] returns a [ModuleInfo] containing:
//
//   - Path: Module path (e.g., "github.com/spf13/cobra")
//   - Version: Latest version from @latest endpoint
//   - Dependencies: Direct dependencies from go.mod
//
// # Caching
//
// Responses are cached to reduce load on the proxy. The cache TTL is set
// when creating the client. Pass refresh=true to bypass the cache.
//
// # Dependency Filtering
//
// Only direct dependencies are included. Indirect dependencies (marked with
// "// indirect" comment) are filtered out.
//
// # Two-Phase Fetch
//
// The client performs two requests:
//  1. @latest endpoint to get the latest version
//  2. .mod endpoint to fetch and parse go.mod for dependencies
//
// Some modules don't have a go.mod file (pre-modules or minimal modules).
// In this case, dependencies will be empty.
//
// # Path Escaping
//
// Module paths with uppercase letters are escaped per the Go module proxy
// protocol (uppercase becomes !lowercase).
package goproxy
