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
// exceeded its TTL. The caller should fetch fresh data and update the cache.
var ErrExpired = errors.New("cache entry expired")

// Cache provides file-based caching for HTTP responses.
// Each entry is stored as a JSON file with a SHA-256 hash of the key as filename.
type Cache struct {
	dir string
	ttl time.Duration
}

// NewCache creates a cache in the specified directory with the given TTL.
// If dir is empty, defaults to ~/.cache/stacktower/. The directory is created
// if it doesn't exist.
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
	return &Cache{dir: dir, ttl: ttl}, nil
}

// Dir returns the cache directory path.
func (c *Cache) Dir() string { return c.dir }

// TTL returns the cache time-to-live duration.
func (c *Cache) TTL() time.Duration { return c.ttl }

// Get retrieves a cached value by key into v. Returns (true, nil) on cache hit,
// (false, nil) on cache miss, and (false, ErrExpired) if the entry exists but
// has exceeded its TTL. The value is JSON-decoded into v.
func (c *Cache) Get(key string, v any) (bool, error) {
	path := c.keyPath(key)
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
// The value is JSON-encoded before writing to disk.
func (c *Cache) Set(key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(c.keyPath(key), data, 0o644)
}

func (c *Cache) keyPath(key string) string {
	h := sha256.Sum256([]byte(key))
	return filepath.Join(c.dir, hex.EncodeToString(h[:]))
}
