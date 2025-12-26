package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FilesystemStorage stores artifacts on the local filesystem.
// Suitable for local development, single-server deployments, and CLI usage.
type FilesystemStorage struct {
	root string
}

// NewFilesystemStorage creates a new filesystem storage rooted at the given directory.
func NewFilesystemStorage(root string) (*FilesystemStorage, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute path: %w", err)
	}
	return &FilesystemStorage{root: absRoot}, nil
}

func (s *FilesystemStorage) Store(ctx context.Context, jobID, filename string, data io.Reader) (string, error) {
	if jobID == "" {
		return "", fmt.Errorf("job ID cannot be empty")
	}
	if err := ValidateFilename(filename); err != nil {
		return "", err
	}

	jobDir := filepath.Join(s.root, jobID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		return "", fmt.Errorf("create job directory: %w", err)
	}

	filePath := filepath.Join(jobDir, filename)
	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, data); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return JoinPath(jobID, filename), nil
}

func (s *FilesystemStorage) Retrieve(ctx context.Context, path string) (io.ReadCloser, error) {
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	filePath := filepath.Join(s.root, path)
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	if !strings.HasPrefix(absPath, s.root) {
		return nil, fmt.Errorf("path traversal detected: %s", path)
	}

	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("artifact not found: %s", path)
		}
		return nil, fmt.Errorf("open file: %w", err)
	}
	return f, nil
}

func (s *FilesystemStorage) Exists(ctx context.Context, path string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("path cannot be empty")
	}
	filePath := filepath.Join(s.root, path)
	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("stat file: %w", err)
}

func (s *FilesystemStorage) Delete(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	filePath := filepath.Join(s.root, path)
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	if !strings.HasPrefix(absPath, s.root) {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("artifact not found: %s", path)
		}
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}

func (s *FilesystemStorage) List(ctx context.Context, jobID string) ([]string, error) {
	if jobID == "" {
		return nil, fmt.Errorf("job ID cannot be empty")
	}

	jobDir := filepath.Join(s.root, jobID)
	if _, err := os.Stat(jobDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	var paths []string
	err := filepath.Walk(jobDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(relPath))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}

	sort.Strings(paths)
	return paths, nil
}

func (s *FilesystemStorage) URL(ctx context.Context, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	filePath := filepath.Join(s.root, path)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("artifact not found: %s", path)
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	return "file://" + absPath, nil
}

func (s *FilesystemStorage) DeleteJob(ctx context.Context, jobID string) error {
	if jobID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}
	jobDir := filepath.Join(s.root, jobID)
	if _, err := os.Stat(jobDir); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(jobDir)
}

func (s *FilesystemStorage) Close() error { return nil }

var _ Storage = (*FilesystemStorage)(nil)
