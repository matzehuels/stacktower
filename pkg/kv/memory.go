package kv

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is an in-memory cache backend for testing and small datasets.
//
// This backend is suitable for:
//   - Unit tests
//   - Small datasets that fit in memory
//   - Single-process applications
//
// Data is lost when the process exits. For persistent caching,
// use [FilesystemStore] or [RedisStore] instead.
type MemoryStore struct {
	mu      sync.RWMutex
	entries map[string]*memoryEntry
}

type memoryEntry struct {
	data      []byte
	expiresAt time.Time
}

// NewMemoryStore creates an in-memory cache backend.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		entries: make(map[string]*memoryEntry),
	}
}

// Get retrieves a cached value from memory.
func (s *MemoryStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	s.mu.RLock()
	entry, ok := s.entries[key]
	s.mu.RUnlock()

	if !ok {
		return nil, false, nil
	}

	// Check expiration
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		// Clean up expired entry
		s.mu.Lock()
		delete(s.entries, key)
		s.mu.Unlock()
		return nil, false, ErrExpired
	}

	return entry.data, true, nil
}

// Set stores a value in memory with TTL.
func (s *MemoryStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	entry := &memoryEntry{
		data: value,
	}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}

	s.mu.Lock()
	s.entries[key] = entry
	s.mu.Unlock()

	return nil
}

// Delete removes an entry from memory.
func (s *MemoryStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	delete(s.entries, key)
	s.mu.Unlock()
	return nil
}

// Close is a no-op for memory backend.
func (s *MemoryStore) Close() error {
	return nil
}

// Clear removes all entries from the cache (useful for testing).
func (s *MemoryStore) Clear() {
	s.mu.Lock()
	s.entries = make(map[string]*memoryEntry)
	s.mu.Unlock()
}
