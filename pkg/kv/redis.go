package kv

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore stores cache entries in Redis.
//
// This backend is suitable for:
//   - Multi-instance deployments (shared cache)
//   - Production environments
//   - High-throughput caching
//
// Redis handles TTL expiration automatically.
type RedisStore struct {
	client    *redis.Client
	keyPrefix string
}

// NewRedisStore creates a Redis-based cache backend.
//
// The client should be obtained from infra.NewRedis().Raw() or created directly.
// The keyPrefix is prepended to all keys to avoid collisions with other
// applications using the same Redis instance.
func NewRedisStore(client *redis.Client, keyPrefix string) *RedisStore {
	if keyPrefix == "" {
		keyPrefix = "stacktower:cache:"
	}
	return &RedisStore{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// Get retrieves a cached value from Redis.
func (s *RedisStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	result, err := s.client.Get(ctx, s.keyPrefix+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return result, true, nil
}

// Set stores a value in Redis with TTL.
func (s *RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return s.client.Set(ctx, s.keyPrefix+key, value, ttl).Err()
}

// Delete removes an entry from Redis.
func (s *RedisStore) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, s.keyPrefix+key).Err()
}

// Close is a no-op - the Redis client should be closed by its owner.
func (s *RedisStore) Close() error {
	return nil
}

// Client returns the underlying Redis client for advanced operations.
func (s *RedisStore) Client() *redis.Client {
	return s.client
}

