// Package queue provides an abstraction for job queues that can be implemented
// with different backends (in-memory, Redis, RabbitMQ, SQS, etc.).
//
// The queue package enables decoupling of API servers from worker processes,
// allowing long-running dependency resolution and rendering jobs to be processed
// asynchronously. This design supports horizontal scaling of both API and worker tiers.
//
// # Architecture
//
// The queue abstraction consists of three main components:
//
//  1. Queue: Manages job submission and retrieval
//  2. Job: Represents a unit of work with type, payload, and status
//  3. Backend implementations: In-memory (for local testing) or distributed (for production)
//
// # Usage
//
// Submit a job to the queue:
//
//	q := memory.New()
//	job := &queue.Job{
//	    ID:      "job-123",
//	    Type:    "parse",
//	    Payload: map[string]interface{}{"package": "requests", "language": "python"},
//	    Status:  queue.StatusPending,
//	}
//	if err := q.Enqueue(ctx, job); err != nil {
//	    log.Fatal(err)
//	}
//
// Worker dequeues and processes jobs:
//
//	for {
//	    job, err := q.Dequeue(ctx, "parse", "render")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    // Process job...
//	    job.Status = queue.StatusCompleted
//	    q.UpdateStatus(ctx, job.ID, job.Status, result)
//	}
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Status represents the current state of a job.
type Status string

const (
	// StatusPending indicates the job is queued but not yet started.
	StatusPending Status = "pending"

	// StatusRunning indicates the job is currently being processed.
	StatusRunning Status = "running"

	// StatusCompleted indicates the job finished successfully.
	StatusCompleted Status = "completed"

	// StatusFailed indicates the job encountered an error.
	StatusFailed Status = "failed"

	// StatusCancelled indicates the job was cancelled by the user.
	StatusCancelled Status = "cancelled"
)

// Job represents a unit of work in the queue.
// Jobs are created by the API layer and processed by workers.
type Job struct {
	// ID is a unique identifier for the job (e.g., UUID).
	ID string `json:"id"`

	// Type identifies the kind of work to perform (e.g., "parse", "render", "parse-and-render").
	Type string `json:"type"`

	// Payload contains job-specific parameters as JSON-serializable data.
	// For parse jobs: {"language": "python", "package": "requests", "max_depth": 10}
	// For render jobs: {"input_path": "graph.json", "formats": ["svg", "pdf"]}
	Payload map[string]interface{} `json:"payload"`

	// Status tracks the job's current state.
	Status Status `json:"status"`

	// Result contains the job output after completion (e.g., file paths, URLs).
	// Only populated when Status is StatusCompleted.
	Result map[string]interface{} `json:"result,omitempty"`

	// Error contains the error message if Status is StatusFailed.
	Error string `json:"error,omitempty"`

	// CreatedAt is when the job was submitted.
	CreatedAt time.Time `json:"created_at"`

	// StartedAt is when the job began processing (nil if not started).
	StartedAt *time.Time `json:"started_at,omitempty"`

	// CompletedAt is when the job finished (nil if not completed).
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// WebhookURL is an optional callback URL to notify when the job completes.
	WebhookURL string `json:"webhook_url,omitempty"`
}

// Duration returns the time elapsed from creation to completion.
// Returns 0 if the job is not yet completed.
func (j *Job) Duration() time.Duration {
	if j.CompletedAt == nil {
		return 0
	}
	return j.CompletedAt.Sub(j.CreatedAt)
}

// ProcessingDuration returns the time spent in the running state.
// Returns 0 if the job hasn't started or is still running.
func (j *Job) ProcessingDuration() time.Duration {
	if j.StartedAt == nil {
		return 0
	}
	end := time.Now()
	if j.CompletedAt != nil {
		end = *j.CompletedAt
	}
	return end.Sub(*j.StartedAt)
}

// MarshalPayload converts a struct to a JSON payload map.
// This is a convenience function for creating job payloads from typed structs.
func MarshalPayload(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	return payload, nil
}

// UnmarshalPayload converts a JSON payload map to a typed struct.
// This is a convenience function for extracting typed data from job payloads.
func UnmarshalPayload(payload map[string]interface{}, v interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	return nil
}

// Queue is the interface for job queue implementations.
// Implementations must be safe for concurrent use by multiple goroutines.
type Queue interface {
	// Enqueue adds a job to the queue.
	// The job's Status should be StatusPending when enqueued.
	Enqueue(ctx context.Context, job *Job) error

	// Dequeue retrieves and locks the next available job matching any of the given types.
	// The job's Status is automatically updated to StatusRunning.
	// Returns nil if no jobs are available (non-blocking).
	// Workers should call UpdateStatus when the job completes or fails.
	Dequeue(ctx context.Context, jobTypes ...string) (*Job, error)

	// Get retrieves a job by ID without modifying its state.
	// Returns an error if the job doesn't exist.
	Get(ctx context.Context, jobID string) (*Job, error)

	// UpdateStatus updates a job's status and optionally stores a result or error.
	// For StatusCompleted: result should contain output data (e.g., {"output_path": "/path/to/file"})
	// For StatusFailed: errorMsg should contain the error description
	UpdateStatus(ctx context.Context, jobID string, status Status, result map[string]interface{}, errorMsg string) error

	// Cancel marks a job as cancelled if it hasn't started processing yet.
	// Returns an error if the job is already running or completed.
	Cancel(ctx context.Context, jobID string) error

	// List returns all jobs, optionally filtered by status.
	// If statuses is empty, returns all jobs.
	// Results are ordered by creation time (newest first).
	List(ctx context.Context, statuses ...Status) ([]*Job, error)

	// Delete removes a job from the queue.
	// This is typically used for cleanup of old completed jobs.
	Delete(ctx context.Context, jobID string) error

	// Close releases any resources held by the queue (connections, goroutines, etc.).
	Close() error
}
