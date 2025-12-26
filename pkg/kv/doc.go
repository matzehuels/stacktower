// Package kv provides pluggable key-value storage with TTL and high-level caching.
//
// # Overview
//
// This package provides a generic key-value storage interface [Store] and
// several implementations:
//
//   - [MemoryStore]: In-memory storage for testing and small datasets
//   - [FilesystemStore]: File-based storage for CLI and local development
//   - [RedisStore]: Redis-based storage for production deployments
//
// It also provides a high-level [Cache] wrapper that handles JSON serialization,
// key hashing, and namespacing, as well as [Retry] utilities for resilient
// operations.
//
// # Usage
//
// For CLI/local development:
//
//	store, _ := kv.NewFilesystemStore("")
//	cache := kv.NewCache(store, 24*time.Hour)
//	pypi := cache.Namespace("pypi:")
//	pypi.Set(ctx, "requests", packageData)
//
// For production with Redis:
//
//	store := kv.NewRedisStore(redisClient, "myapp:")
//	cache := kv.NewCache(store, 24*time.Hour)
package kv
