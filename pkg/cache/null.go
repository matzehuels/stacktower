package cache

import (
	"context"
	"time"
)

// NullCache is a no-op cache that never stores anything.
// Useful for testing or when caching should be disabled.
type NullCache struct{}

// NewNullCache creates a null cache.
func NewNullCache() Cache {
	return &NullCache{}
}

// Get always returns a cache miss.
func (c *NullCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	return nil, false, nil
}

// Set does nothing.
func (c *NullCache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	return nil
}

// Delete does nothing.
func (c *NullCache) Delete(ctx context.Context, key string) error {
	return nil
}

// Close does nothing.
func (c *NullCache) Close() error {
	return nil
}

// Ensure NullCache implements Cache.
var _ Cache = (*NullCache)(nil)
