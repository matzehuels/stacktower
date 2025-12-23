package httputil_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/matzehuels/stacktower/pkg/httputil"
)

func ExampleCache() {
	// Create a cache with 24-hour TTL in a temp directory
	dir := filepath.Join(os.TempDir(), "stacktower-example")
	cache, err := httputil.NewCache(dir, 24*time.Hour)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Store a value
	data := map[string]string{"name": "example", "version": "1.0.0"}
	if err := cache.Set("mykey", data); err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Retrieve the value
	var result map[string]string
	if ok, err := cache.Get("mykey", &result); ok && err == nil {
		fmt.Println("Name:", result["name"])
		fmt.Println("Version:", result["version"])
	}

	// Clean up
	os.RemoveAll(dir)
	// Output:
	// Name: example
	// Version: 1.0.0
}

func ExampleCache_miss() {
	dir := filepath.Join(os.TempDir(), "stacktower-example-miss")
	cache, _ := httputil.NewCache(dir, time.Hour)
	defer os.RemoveAll(dir)

	// Try to get a non-existent key
	var result string
	ok, err := cache.Get("nonexistent", &result)
	fmt.Println("Found:", ok)
	fmt.Println("Error:", err)
	// Output:
	// Found: false
	// Error: <nil>
}

func ExampleNewCache_defaultDir() {
	// Pass empty string to use default directory (~/.cache/stacktower/)
	cache, err := httputil.NewCache("", 24*time.Hour)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Cache TTL:", cache.TTL())
	// Output:
	// Cache TTL: 24h0m0s
}

func ExampleRetry() {
	ctx := context.Background()
	attempts := 0

	// Simulate an operation that fails twice then succeeds
	err := httputil.Retry(ctx, 3, 10*time.Millisecond, func() error {
		attempts++
		if attempts < 3 {
			// Wrap transient errors to enable retry
			return &httputil.RetryableError{
				Err: fmt.Errorf("temporary failure (attempt %d)", attempts),
			}
		}
		return nil // Success
	})

	if err != nil {
		fmt.Println("Failed:", err)
	} else {
		fmt.Println("Success after", attempts, "attempts")
	}
	// Output:
	// Success after 3 attempts
}

func ExampleRetryWithBackoff() {
	ctx := context.Background()

	// Fetch data with automatic retry on transient failures
	err := httputil.RetryWithBackoff(ctx, func() error {
		// Your HTTP request or other operation here
		// Return &httputil.RetryableError{...} for transient failures
		// Return regular errors for permanent failures
		return nil
	})

	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Success")
	}
	// Output:
	// Success
}

func ExampleRetryableError() {
	ctx := context.Background()
	networkErr := errors.New("connection refused")

	err := httputil.Retry(ctx, 2, 10*time.Millisecond, func() error {
		// Permanent error - no retry
		if false {
			return errors.New("invalid request")
		}
		// Transient error - will retry
		return &httputil.RetryableError{Err: networkErr}
	})

	// Check if the underlying error is our network error
	if errors.Is(err, networkErr) {
		fmt.Println("Failed due to network error")
	}
	// Output:
	// Failed due to network error
}

func ExampleRetryable() {
	ctx := context.Background()
	attempts := 0

	// Using the Retryable helper for cleaner code
	err := httputil.RetryWithBackoff(ctx, func() error {
		attempts++
		if attempts < 2 {
			// Wrap errors concisely with Retryable()
			return httputil.Retryable(errors.New("temporary failure"))
		}
		return nil
	})

	if err == nil {
		fmt.Println("Success")
	}
	// Output:
	// Success
}

func ExampleCache_Namespace() {
	dir := filepath.Join(os.TempDir(), "stacktower-namespace-example")
	cache, _ := httputil.NewCache(dir, 24*time.Hour)
	defer os.RemoveAll(dir)

	// Create namespaced caches for different registries
	pypiCache := cache.Namespace("pypi:")
	npmCache := cache.Namespace("npm:")

	// Store values in different namespaces
	pypiCache.Set("requests", map[string]string{"version": "2.31.0"})
	npmCache.Set("express", map[string]string{"version": "4.18.2"})

	// Retrieve from appropriate namespace
	var pypiData map[string]string
	pypiCache.Get("requests", &pypiData)
	fmt.Println("PyPI requests:", pypiData["version"])

	var npmData map[string]string
	npmCache.Get("express", &npmData)
	fmt.Println("npm express:", npmData["version"])

	// Output:
	// PyPI requests: 2.31.0
	// npm express: 4.18.2
}
