package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GridFSStorage stores artifacts in MongoDB GridFS.
// Use for production deployments requiring durable, shared storage.
type GridFSStorage struct {
	bucket *gridfs.Bucket
}

// NewGridFSStorage creates a new GridFS storage using the given database.
func NewGridFSStorage(db *mongo.Database) (*GridFSStorage, error) {
	bucket, err := gridfs.NewBucket(db, options.GridFSBucket().SetName("artifacts"))
	if err != nil {
		return nil, fmt.Errorf("create GridFS bucket: %w", err)
	}
	return &GridFSStorage{bucket: bucket}, nil
}

// NewGridFSStorageWithBucket creates storage using an existing bucket.
func NewGridFSStorageWithBucket(bucket *gridfs.Bucket) *GridFSStorage {
	return &GridFSStorage{bucket: bucket}
}

func (s *GridFSStorage) Store(ctx context.Context, jobID, filename string, data io.Reader) (string, error) {
	if err := ValidateFilename(filename); err != nil {
		return "", err
	}

	path := JoinPath(jobID, filename)
	uploadOpts := options.GridFSUpload().SetMetadata(bson.M{
		"job_id":       jobID,
		"filename":     filename,
		"content_type": ContentTypeFromExtension(filename),
		"created_at":   time.Now(),
	})

	_, err := s.bucket.UploadFromStream(path, data, uploadOpts)
	if err != nil {
		return "", fmt.Errorf("upload to GridFS: %w", err)
	}
	return path, nil
}

func (s *GridFSStorage) Retrieve(ctx context.Context, path string) (io.ReadCloser, error) {
	stream, err := s.bucket.OpenDownloadStreamByName(path)
	if err != nil {
		if isGridFSNotFound(err) {
			return nil, fmt.Errorf("%s: %w", path, ErrNotFound)
		}
		return nil, fmt.Errorf("open GridFS stream: %w", err)
	}
	return stream, nil
}

func (s *GridFSStorage) Exists(ctx context.Context, path string) (bool, error) {
	cursor, err := s.bucket.Find(bson.M{"filename": path})
	if err != nil {
		return false, fmt.Errorf("find in GridFS: %w", err)
	}
	defer cursor.Close(ctx)
	return cursor.Next(ctx), nil
}

func (s *GridFSStorage) Delete(ctx context.Context, path string) error {
	cursor, err := s.bucket.Find(bson.M{"filename": path})
	if err != nil {
		return fmt.Errorf("find in GridFS: %w", err)
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		return fmt.Errorf("%s: %w", path, ErrNotFound)
	}

	var file struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	if err := cursor.Decode(&file); err != nil {
		return fmt.Errorf("decode file: %w", err)
	}
	return s.bucket.Delete(file.ID)
}

func (s *GridFSStorage) List(ctx context.Context, jobID string) ([]string, error) {
	filter := bson.M{"filename": bson.M{"$regex": "^" + jobID + "/"}}
	cursor, err := s.bucket.Find(filter, options.GridFSFind().SetSort(bson.D{{Key: "filename", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("find in GridFS: %w", err)
	}
	defer cursor.Close(ctx)

	var paths []string
	for cursor.Next(ctx) {
		var file struct {
			Filename string `bson:"filename"`
		}
		if err := cursor.Decode(&file); err != nil {
			continue
		}
		paths = append(paths, file.Filename)
	}
	return paths, nil
}

func (s *GridFSStorage) URL(ctx context.Context, path string) (string, error) {
	return "/api/v1/artifacts/" + path, nil
}

func (s *GridFSStorage) DeleteJob(ctx context.Context, jobID string) error {
	filter := bson.M{"filename": bson.M{"$regex": "^" + jobID + "/"}}
	cursor, err := s.bucket.Find(filter)
	if err != nil {
		return fmt.Errorf("find in GridFS: %w", err)
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var file struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&file); err != nil {
			continue
		}
		s.bucket.Delete(file.ID)
	}
	return nil
}

func (s *GridFSStorage) Close() error { return nil }

// GetBytes returns the artifact as bytes.
func (s *GridFSStorage) GetBytes(ctx context.Context, path string) ([]byte, error) {
	stream, err := s.bucket.OpenDownloadStreamByName(path)
	if err != nil {
		if isGridFSNotFound(err) {
			return nil, fmt.Errorf("%s: %w", path, ErrNotFound)
		}
		return nil, fmt.Errorf("open GridFS stream: %w", err)
	}
	defer stream.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, stream); err != nil {
		return nil, fmt.Errorf("read GridFS stream: %w", err)
	}
	return buf.Bytes(), nil
}

// StoreBytes stores bytes directly.
func (s *GridFSStorage) StoreBytes(ctx context.Context, jobID, filename string, data []byte) (string, error) {
	return s.Store(ctx, jobID, filename, bytes.NewReader(data))
}

func isGridFSNotFound(err error) bool {
	return err == gridfs.ErrFileNotFound ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "no documents")
}

var _ Storage = (*GridFSStorage)(nil)
