package httputil_test

import (
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
