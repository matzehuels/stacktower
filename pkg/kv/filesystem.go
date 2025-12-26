package kv

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// FilesystemStore stores cache entries as files in a directory.
//
// Each entry is stored as a JSON file containing the value and metadata.
// The filename is the cache key (typically a pre-hashed string).
//
// This backend is suitable for:
//   - Local development and CLI tools
//   - Single-instance deployments
//   - Persistent cache across restarts
//
// For multi-instance deployments, use [RedisStore] instead.
type FilesystemStore struct {
	dir string
}

// filesystemEntry wraps cached data with expiration metadata.
type filesystemEntry struct {
	Data      []byte    `json:"data"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// NewFilesystemStore creates a filesystem-based cache backend.
//
// If dir is empty, uses the default directory ~/.cache/stacktower/.
// The directory is created with mode 0755 if it doesn't exist.
func NewFilesystemStore(dir string) (*FilesystemStore, error) {
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
	return &FilesystemStore{dir: dir}, nil
}

// Dir returns the cache directory path.
func (s *FilesystemStore) Dir() string {
	return s.dir
}

// Get retrieves a cached value from the filesystem.
func (s *FilesystemStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	path := filepath.Join(s.dir, key)

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var entry filesystemEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Legacy format: raw data without wrapper
		return data, true, nil
	}

	// Check expiration
	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		return nil, false, ErrExpired
	}

	return entry.Data, true, nil
}

// Set stores a value in the filesystem with TTL.
func (s *FilesystemStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	entry := filesystemEntry{
		Data: value,
	}
	if ttl > 0 {
		entry.ExpiresAt = time.Now().Add(ttl)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	path := filepath.Join(s.dir, key)
	return os.WriteFile(path, data, 0o644)
}

// Delete removes an entry from the filesystem.
func (s *FilesystemStore) Delete(ctx context.Context, key string) error {
	path := filepath.Join(s.dir, key)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Close is a no-op for filesystem backend.
func (s *FilesystemStore) Close() error {
	return nil
}
