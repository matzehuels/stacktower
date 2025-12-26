package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

// MemoryStorage is an in-memory artifact store.
// Use for development, testing, and local CLI usage.
// Artifacts are lost when the process exits.
type MemoryStorage struct {
	mu        sync.RWMutex
	artifacts map[string][]byte
}

// NewMemoryStorage creates a new in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{artifacts: make(map[string][]byte)}
}

func (s *MemoryStorage) Store(ctx context.Context, jobID, filename string, data io.Reader) (string, error) {
	if err := ValidateFilename(filename); err != nil {
		return "", err
	}
	content, err := io.ReadAll(data)
	if err != nil {
		return "", fmt.Errorf("read data: %w", err)
	}
	path := JoinPath(jobID, filename)
	s.mu.Lock()
	s.artifacts[path] = content
	s.mu.Unlock()
	return path, nil
}

func (s *MemoryStorage) Retrieve(ctx context.Context, path string) (io.ReadCloser, error) {
	s.mu.RLock()
	data, ok := s.artifacts[path]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%s: %w", path, ErrNotFound)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (s *MemoryStorage) Exists(ctx context.Context, path string) (bool, error) {
	s.mu.RLock()
	_, ok := s.artifacts[path]
	s.mu.RUnlock()
	return ok, nil
}

func (s *MemoryStorage) Delete(ctx context.Context, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.artifacts[path]; !ok {
		return fmt.Errorf("%s: %w", path, ErrNotFound)
	}
	delete(s.artifacts, path)
	return nil
}

func (s *MemoryStorage) List(ctx context.Context, jobID string) ([]string, error) {
	prefix := jobID + "/"
	s.mu.RLock()
	var paths []string
	for path := range s.artifacts {
		if strings.HasPrefix(path, prefix) {
			paths = append(paths, path)
		}
	}
	s.mu.RUnlock()
	sort.Strings(paths)
	return paths, nil
}

func (s *MemoryStorage) URL(ctx context.Context, path string) (string, error) {
	return "", fmt.Errorf("URL not supported for memory storage")
}

func (s *MemoryStorage) DeleteJob(ctx context.Context, jobID string) error {
	prefix := jobID + "/"
	s.mu.Lock()
	defer s.mu.Unlock()
	for path := range s.artifacts {
		if strings.HasPrefix(path, prefix) {
			delete(s.artifacts, path)
		}
	}
	return nil
}

func (s *MemoryStorage) Close() error { return nil }

// GetBytes returns the artifact as bytes (for testing).
func (s *MemoryStorage) GetBytes(ctx context.Context, path string) ([]byte, error) {
	s.mu.RLock()
	data, ok := s.artifacts[path]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%s: %w", path, ErrNotFound)
	}
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

var _ Storage = (*MemoryStorage)(nil)
