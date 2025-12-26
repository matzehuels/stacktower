package kv

import (
	"context"
	"encoding/json"
	"time"

	"github.com/matzehuels/stacktower/pkg/hash"
)

// Cache provides caching of arbitrary JSON-marshalable data.
//
// Cache is a high-level wrapper around a [Store] that handles:
//   - JSON serialization/deserialization
//   - Key hashing (SHA-256)
//   - Namespace prefixing
//   - TTL management
//
// Use [Cache.Namespace] to create scoped views that automatically prefix
// keys, avoiding collisions between different data sources.
type Cache struct {
	store  Store
	ttl    time.Duration
	prefix string
}

// NewCache creates a Cache with the given store and default TTL.
func NewCache(s Store, ttl time.Duration) *Cache {
	return &Cache{
		store:  s,
		ttl:    ttl,
		prefix: "",
	}
}

// Store returns the underlying storage backend.
func (c *Cache) Store() Store {
	return c.store
}

// TTL returns the time-to-live duration for cache entries.
func (c *Cache) TTL() time.Duration {
	return c.ttl
}

// Get retrieves a cached value by key and unmarshals it into v.
func (c *Cache) Get(ctx context.Context, key string, v any) (bool, error) {
	hashedKey := c.hashKey(c.prefix + key)

	data, found, err := c.store.Get(ctx, hashedKey)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}

	return true, json.Unmarshal(data, v)
}

// Set stores a value in the cache under the given key.
func (c *Cache) Set(ctx context.Context, key string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	hashedKey := c.hashKey(c.prefix + key)
	return c.store.Set(ctx, hashedKey, data, c.ttl)
}

// Delete removes an entry from the cache.
func (c *Cache) Delete(ctx context.Context, key string) error {
	hashedKey := c.hashKey(c.prefix + key)
	return c.store.Delete(ctx, hashedKey)
}

// Namespace returns a new Cache that automatically prefixes all keys with prefix.
func (c *Cache) Namespace(prefix string) *Cache {
	return &Cache{
		store:  c.store,
		ttl:    c.ttl,
		prefix: c.prefix + prefix,
	}
}

// Close releases resources held by the underlying store.
func (c *Cache) Close() error {
	return c.store.Close()
}

// hashKey creates a SHA-256 hash of the key for storage.
func (c *Cache) hashKey(key string) string {
	return hash.Bytes([]byte(key))
}

