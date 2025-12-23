package httputil

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// ErrExpired is returned by [Cache.Get] when a cached entry exists but has
// exceeded its time-to-live (TTL).
//
// When you receive ErrExpired, the cached data still exists on disk but is
// considered stale. Callers should fetch fresh data from the source and
// update the cache with [Cache.Set].
//
// Use errors.Is to check for this error:
//
//	ok, err := cache.Get("key", &value)
//	if errors.Is(err, httputil.ErrExpired) {
//	    // Fetch fresh data and update cache
//	}
var ErrExpired = errors.New("cache entry expired")

// Cache provides file-based caching of arbitrary JSON-marshalable data.
//
// Each cache entry is stored as a JSON file in the cache directory, with
// the filename derived from a SHA-256 hash of the cache key. This design
// ensures safe key names (no filesystem special characters) and prevents
// key collisions across different namespaces.
//
// Cache operations are not goroutine-safe. If multiple goroutines access
// the same Cache instance, the caller must synchronize access. However,
// multiple Cache instances (even in different processes) can safely share
// the same directory, as the filesystem provides atomic file operations.
//
// Cache entries have a time-to-live (TTL) based on file modification time.
// A TTL of 0 means entries never expire.
//
// Use [Cache.Namespace] to create scoped views that automatically prefix
// keys, avoiding collisions between different data sources:
//
//	pypi := cache.Namespace("pypi:")
//	npm := cache.Namespace("npm:")
//	pypi.Set("requests", data)  // key becomes "pypi:requests"
type Cache struct {
	dir    string
	ttl    time.Duration
	prefix string
}

// NewCache creates a Cache that stores entries in dir with the given TTL.
//
// If dir is empty, NewCache uses the default directory ~/.cache/stacktower/.
// The directory is created with mode 0755 if it doesn't exist. If directory
// creation fails (e.g., due to permissions), NewCache returns an error.
//
// Parameters:
//   - dir: Cache directory path. Use "" for default (~/.cache/stacktower/).
//   - ttl: Time-to-live for cache entries. Use 0 for no expiration.
//
// The returned Cache is ready to use. Directory creation errors are the
// only possible source of failure.
func NewCache(dir string, ttl time.Duration) (*Cache, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, ".cache", "stacktower")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Cache{dir: dir, ttl: ttl, prefix: ""}, nil
}

// Dir returns the absolute path to the cache directory.
func (c *Cache) Dir() string { return c.dir }

// TTL returns the time-to-live duration for cache entries.
// A TTL of 0 means cache entries never expire.
func (c *Cache) TTL() time.Duration { return c.ttl }

// Get retrieves a cached value by key and unmarshals it into v.
//
// Return values indicate three distinct outcomes:
//   - (true, nil): Cache hit. The value was found, is fresh, and unmarshaled into v.
//   - (false, nil): Cache miss. No entry exists for this key. v is unchanged.
//   - (false, ErrExpired): Entry exists but exceeded its TTL. v is unchanged.
//   - (false, other error): I/O error, JSON unmarshal error, etc. v may be partially modified.
//
// The key can be any string. Consider namespacing keys to avoid collisions
// (e.g., "pypi:requests", "npm:react"). The key is hashed with SHA-256,
// so long keys are acceptable.
//
// The value v must be a pointer to a type compatible with json.Unmarshal.
// Common types include *string, *[]byte, *map[string]any, and pointers to
// custom structs with JSON tags.
//
// Get does not modify the cache or update modification times; reads are
// non-mutating operations.
func (c *Cache) Get(key string, v any) (bool, error) {
	path := c.keyPath(c.prefix + key)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if c.ttl > 0 && time.Since(info.ModTime()) > c.ttl {
		return false, ErrExpired
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(data, v)
}

// Set stores a value in the cache under the given key.
//
// The value v is marshaled to JSON using encoding/json and written to disk.
// If v cannot be marshaled (e.g., contains channels or functions), Set
// returns a json.MarshalError. If the write fails (e.g., disk full,
// permission denied), Set returns the underlying I/O error.
//
// Set overwrites any existing entry for key, resetting its modification time
// to the current time. This effectively refreshes the TTL.
//
// The value v is not modified by Set; marshaling operates on a copy.
func (c *Cache) Set(key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(c.keyPath(c.prefix+key), data, 0o644)
}

// Namespace returns a new Cache that automatically prefixes all keys with prefix.
//
// This creates a scoped view of the cache, useful for avoiding key collisions
// between different data sources or components. The returned Cache shares the
// same underlying directory and TTL as the parent.
//
// Example:
//
//	cache, _ := httputil.NewCache("", 24*time.Hour)
//	pypiCache := cache.Namespace("pypi:")
//	npmCache := cache.Namespace("npm:")
//
//	pypiCache.Set("requests", pypiData)  // Stored as "pypi:requests"
//	npmCache.Set("express", npmData)     // Stored as "npm:express"
//
// Namespace calls can be chained to create hierarchical key spaces:
//
//	cache.Namespace("python:").Namespace("pypi:")  // prefix: "python:pypi:"
//
// The prefix is applied transparently to all Get and Set operations.
// An empty prefix is valid and results in no key transformation.
func (c *Cache) Namespace(prefix string) *Cache {
	return &Cache{
		dir:    c.dir,
		ttl:    c.ttl,
		prefix: c.prefix + prefix,
	}
}

func (c *Cache) keyPath(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, hex.EncodeToString(h[:]))
}
