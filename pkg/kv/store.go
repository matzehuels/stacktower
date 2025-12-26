// Package kv provides pluggable key-value storage with TTL.
package kv

import (
	"context"
	"time"

	pkgerr "github.com/matzehuels/stacktower/pkg/errors"
)

// ErrExpired is returned when a cached entry exists but has exceeded its TTL.
// This allows callers to distinguish between "not found" and "found but stale".
var ErrExpired = pkgerr.ErrExpired

// Store defines the interface for key-value storage with TTL.
//
// Implementations must be safe for concurrent use by multiple goroutines.
// Keys are typically pre-hashed strings, so backends don't need to handle
// key normalization or length limits.
type Store interface {
	// Get retrieves a value by key.
	// Returns (value, true, nil) on hit, (nil, false, nil) on miss.
	// May return ErrExpired if the entry exists but has exceeded its TTL.
	Get(ctx context.Context, key string) ([]byte, bool, error)

	// Set stores a value with TTL. TTL of 0 means no expiration.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes a key. Returns nil if key doesn't exist.
	Delete(ctx context.Context, key string) error

	// Close releases resources.
	Close() error
}
