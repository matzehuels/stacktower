// Package storage provides an abstraction for storing job artifacts (graphs, rendered images, etc.).
//
// The storage abstraction enables decoupling of artifact storage from the worker implementation,
// allowing seamless migration from local filesystem (for testing) to cloud storage (for production).
//
// # Architecture
//
// The storage abstraction consists of:
//
//  1. Storage: Interface for storing and retrieving artifacts
//  2. Backend implementations: Filesystem (local), S3, GCS, Azure Blob, etc.
//
// # Usage
//
// Store a file:
//
//	store := filesystem.New("/var/stacktower/artifacts")
//	path, err := store.Store(ctx, "job-123", "graph.json", bytes.NewReader(data))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// path: "job-123/graph.json"
//
// Retrieve a file:
//
//	reader, err := store.Retrieve(ctx, "job-123/graph.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer reader.Close()
//	data, _ := io.ReadAll(reader)
//
// Get a public URL (for cloud storage):
//
//	url, err := store.URL(ctx, "job-123/graph.json")
//	// url: "https://storage.googleapis.com/bucket/job-123/graph.json"
package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"time"

	pkgerr "github.com/matzehuels/stacktower/pkg/errors"
)

// Sentinel errors for storage operations.
var (
	// ErrNotFound is returned when an artifact does not exist.
	ErrNotFound = pkgerr.ErrNotFound
)

// Storage is the interface for artifact storage implementations.
// Implementations must be safe for concurrent use by multiple goroutines.
type Storage interface {
	// Store saves data to storage under the given job ID and filename.
	// Returns the storage path (e.g., "job-123/graph.json").
	// The reader is consumed entirely and closed by this method.
	Store(ctx context.Context, jobID, filename string, data io.Reader) (string, error)

	// Retrieve returns a reader for the artifact at the given path.
	// The caller is responsible for closing the reader.
	// Returns an error if the artifact doesn't exist.
	Retrieve(ctx context.Context, path string) (io.ReadCloser, error)

	// Exists checks if an artifact exists at the given path.
	Exists(ctx context.Context, path string) (bool, error)

	// Delete removes an artifact at the given path.
	// Returns an error if the artifact doesn't exist.
	Delete(ctx context.Context, path string) error

	// List returns all artifact paths for a given job ID.
	// Results are sorted alphabetically.
	List(ctx context.Context, jobID string) ([]string, error)

	// URL returns a publicly accessible URL for the artifact.
	// For local filesystem storage, this may return a file:// URL or an error.
	// For cloud storage, this returns an HTTPS URL (may be signed/temporary).
	URL(ctx context.Context, path string) (string, error)

	// DeleteJob removes all artifacts for a given job ID.
	// This is used for cleanup when a job is deleted.
	DeleteJob(ctx context.Context, jobID string) error

	// Close releases any resources held by the storage backend.
	Close() error
}

// Metadata contains information about a stored artifact.
type Metadata struct {
	// Path is the storage path (e.g., "job-123/graph.json").
	Path string

	// Size is the file size in bytes.
	Size int64

	// ContentType is the MIME type (e.g., "application/json", "image/svg+xml").
	ContentType string

	// CreatedAt is when the artifact was stored.
	CreatedAt time.Time

	// URL is a publicly accessible URL (if available).
	URL string
}

// ParseJobID extracts the job ID from a storage path.
// Example: "job-123/graph.json" -> "job-123"
func ParseJobID(storagePath string) string {
	dir := path.Dir(storagePath)
	if dir == "." || dir == "/" {
		return ""
	}
	return dir
}

// JoinPath constructs a storage path from a job ID and filename.
// Example: ("job-123", "graph.json") -> "job-123/graph.json"
func JoinPath(jobID, filename string) string {
	return path.Join(jobID, filename)
}

// ContentTypeFromExtension returns a MIME type based on file extension.
func ContentTypeFromExtension(filename string) string {
	ext := path.Ext(filename)
	switch ext {
	case ".json":
		return "application/json"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".pdf":
		return "application/pdf"
	case ".txt", ".log":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

// ValidateFilename checks if a filename is safe for storage.
// It rejects paths with directory traversal attempts (../) and absolute paths.
func ValidateFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if path.IsAbs(filename) {
		return fmt.Errorf("filename cannot be an absolute path: %s", filename)
	}
	if path.Clean(filename) != filename {
		return fmt.Errorf("filename contains invalid path components: %s", filename)
	}
	return nil
}
