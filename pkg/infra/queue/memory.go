package queue

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MemoryQueue is an in-memory implementation of Queue.
// Suitable for local development, testing, and single-server deployments.
type MemoryQueue struct {
	mu      sync.RWMutex
	jobs    map[string]*Job
	pending map[string][]string // jobType -> []jobID
	closed  bool
}

// NewMemoryQueue creates a new in-memory queue.
func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		jobs:    make(map[string]*Job),
		pending: make(map[string][]string),
	}
}

func (q *MemoryQueue) Enqueue(ctx context.Context, job *Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}
	if job.ID == "" {
		return fmt.Errorf("job ID cannot be empty")
	}
	if job.Type == "" {
		return fmt.Errorf("job type cannot be empty")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}
	if _, exists := q.jobs[job.ID]; exists {
		return fmt.Errorf("job %s already exists", job.ID)
	}

	q.jobs[job.ID] = job
	if job.Status == StatusPending {
		q.pending[job.Type] = append(q.pending[job.Type], job.ID)
	}
	return nil
}

func (q *MemoryQueue) Dequeue(ctx context.Context, jobTypes ...string) (*Job, error) {
	if len(jobTypes) == 0 {
		return nil, fmt.Errorf("at least one job type must be specified")
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil, fmt.Errorf("queue is closed")
	}

	for _, jobType := range jobTypes {
		pending := q.pending[jobType]
		if len(pending) == 0 {
			continue
		}

		jobID := pending[0]
		q.pending[jobType] = pending[1:]

		job, exists := q.jobs[jobID]
		if !exists {
			continue
		}

		now := time.Now()
		job.Status = StatusRunning
		job.StartedAt = &now
		return job, nil
	}

	return nil, nil
}

func (q *MemoryQueue) Get(ctx context.Context, jobID string) (*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	job, exists := q.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	return job, nil
}

func (q *MemoryQueue) UpdateStatus(ctx context.Context, jobID string, status Status, result map[string]interface{}, errorMsg string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	job, exists := q.jobs[jobID]
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}

	job.Status = status
	if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
		now := time.Now()
		job.CompletedAt = &now
	}
	if status == StatusCompleted && result != nil {
		job.Result = result
	}
	if status == StatusFailed && errorMsg != "" {
		job.Error = errorMsg
	}
	return nil
}

func (q *MemoryQueue) Cancel(ctx context.Context, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	job, exists := q.jobs[jobID]
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}

	if job.Status != StatusPending {
		return fmt.Errorf("cannot cancel job in status %s", job.Status)
	}

	pending := q.pending[job.Type]
	for i, id := range pending {
		if id == jobID {
			q.pending[job.Type] = append(pending[:i], pending[i+1:]...)
			break
		}
	}

	now := time.Now()
	job.Status = StatusCancelled
	job.CompletedAt = &now
	return nil
}

func (q *MemoryQueue) List(ctx context.Context, statuses ...Status) ([]*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	filterByStatus := len(statuses) > 0
	statusMap := make(map[Status]bool)
	for _, s := range statuses {
		statusMap[s] = true
	}

	var result []*Job
	for _, job := range q.jobs {
		if filterByStatus && !statusMap[job.Status] {
			continue
		}
		result = append(result, job)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (q *MemoryQueue) ListByUser(ctx context.Context, userID string, statuses ...Status) ([]*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	filterByStatus := len(statuses) > 0
	statusMap := make(map[Status]bool)
	for _, s := range statuses {
		statusMap[s] = true
	}

	var result []*Job
	for _, job := range q.jobs {
		// Check user_id in payload
		jobUserID, ok := job.Payload["user_id"].(string)
		if !ok || jobUserID != userID {
			continue
		}
		if filterByStatus && !statusMap[job.Status] {
			continue
		}
		result = append(result, job)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (q *MemoryQueue) Delete(ctx context.Context, jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	job, exists := q.jobs[jobID]
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}

	if job.Status == StatusPending {
		pending := q.pending[job.Type]
		for i, id := range pending {
			if id == jobID {
				q.pending[job.Type] = append(pending[:i], pending[i+1:]...)
				break
			}
		}
	}

	delete(q.jobs, jobID)
	return nil
}

func (q *MemoryQueue) Ping(ctx context.Context) error {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.closed {
		return fmt.Errorf("queue is closed")
	}
	return nil
}

func (q *MemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	return nil
}

var _ Queue = (*MemoryQueue)(nil)
