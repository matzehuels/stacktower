// Package httputil provides HTTP utilities for package registry clients.
//
// # Overview
//
// This package provides infrastructure used by all registry API clients:
//
//   - [Cache]: File-based HTTP response caching
//   - [Retry]: Automatic retry with exponential backoff
//
// # Caching
//
// [Cache] stores HTTP responses in the filesystem (~/.cache/stacktower/)
// with configurable TTL. This dramatically speeds up repeated operations
// and reduces load on package registries.
//
// Usage:
//
//	cache, err := httputil.NewCache(24 * time.Hour)
//	data, ok := cache.Get("pypi:fastapi")  // Check cache
//	if !ok {
//	    data = fetchFromAPI()
//	    cache.Set("pypi:fastapi", data)   // Store for later
//	}
//
// Cache keys should be namespaced by registry to avoid collisions.
//
// # Retry
//
// [Retry] wraps HTTP requests with automatic retry for transient failures:
//
//   - Network errors
//   - 5xx server errors
//   - 429 rate limit responses
//
// It uses exponential backoff with jitter to avoid thundering herd:
//
//	resp, err := httputil.Retry(func() (*http.Response, error) {
//	    return http.Get(url)
//	})
//
// # Configuration
//
// Default settings are suitable for most use cases:
//
//   - Cache directory: ~/.cache/stacktower/
//   - Default TTL: 24 hours
//   - Max retries: 3
//   - Base backoff: 1 second
//
// The cache can be cleared via `stacktower cache clear` or by deleting
// the cache directory.
package httputil
