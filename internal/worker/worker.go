// Package worker provides background job processing for Stacktower.
//
// The worker processes jobs from a queue (Redis) and executes pipeline operations
// (parse, layout, render) asynchronously. It supports configurable concurrency
// and graceful shutdown.
//
// # Usage
//
//	w := worker.New(queue, backend, worker.Config{
//	    Concurrency:  4,
//	    PollInterval: time.Second,
//	    Logger:       logger,
//	})
//
//	// Start processing (blocks until context is cancelled)
//	if err := w.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
package worker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// Config holds worker configuration.
type Config struct {
	Concurrency  int
	PollInterval time.Duration
	Logger       *infra.Logger
}

// JobHandler processes a job and returns the result.
type JobHandler func(ctx context.Context, job *queue.Job) (map[string]interface{}, error)

// Worker processes jobs from a queue.
type Worker struct {
	queue    queue.Queue
	backend  *storage.DistributedBackend
	pipeline *pipeline.Service
	config   Config
	handlers map[string]JobHandler // Job type -> handler

	// Shutdown coordination.
	inFlight   atomic.Int32
	cancelFunc context.CancelFunc
	done       chan struct{}
}

// New creates a new worker.
func New(q queue.Queue, backend *storage.DistributedBackend, config Config) *Worker {
	if config.Concurrency <= 0 {
		config.Concurrency = 1
	}
	if config.PollInterval == 0 {
		config.PollInterval = 1 * time.Second
	}
	if config.Logger == nil {
		config.Logger = infra.DefaultLogger()
	}

	w := &Worker{
		queue:    q,
		backend:  backend,
		pipeline: pipeline.NewService(backend),
		config:   config,
		done:     make(chan struct{}),
	}

	// Register job handlers using centralized job types.
	w.handlers = make(map[string]JobHandler)
	for _, jt := range queue.SupportedJobTypes {
		switch jt {
		case queue.TypeParse:
			w.handlers[string(jt)] = w.processParse
		case queue.TypeLayout:
			w.handlers[string(jt)] = w.processLayout
		case queue.TypeRender:
			w.handlers[string(jt)] = w.processRender
		}
	}

	return w
}

// Start begins processing jobs. Blocks until context is cancelled.
func (w *Worker) Start(ctx context.Context) error {
	w.config.Logger.Info("worker starting", "concurrency", w.config.Concurrency)

	// Create cancellable context for shutdown coordination.
	ctx, w.cancelFunc = context.WithCancel(ctx)

	var wg sync.WaitGroup
	for i := 0; i < w.config.Concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			w.workerLoop(ctx, id)
		}(i)
	}

	wg.Wait()
	close(w.done)
	w.config.Logger.Info("worker stopped")
	return nil
}

// Shutdown gracefully stops the worker, waiting for in-flight jobs to complete.
// The provided context controls the maximum time to wait for drain.
func (w *Worker) Shutdown(ctx context.Context) error {
	w.config.Logger.Info("worker shutdown initiated")

	// Signal workers to stop accepting new jobs.
	if w.cancelFunc != nil {
		w.cancelFunc()
	}

	// Wait for in-flight jobs to drain or timeout.
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			w.config.Logger.Info("worker shutdown complete")
			return nil
		case <-ctx.Done():
			inFlight := w.inFlight.Load()
			if inFlight > 0 {
				w.config.Logger.Warn("shutdown timeout with jobs in flight", "in_flight", inFlight)
			}
			return ctx.Err()
		case <-ticker.C:
			// Continue waiting.
		}
	}
}

// InFlight returns the number of jobs currently being processed.
func (w *Worker) InFlight() int32 {
	return w.inFlight.Load()
}

func (w *Worker) workerLoop(ctx context.Context, workerID int) {
	// Use centralized job types list.
	jobTypes := queue.SupportedJobTypeStrings()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, err := w.queue.Dequeue(ctx, jobTypes...)
		if err != nil {
			w.config.Logger.Error("dequeue error", "error", err)
			time.Sleep(w.config.PollInterval)
			continue
		}
		if job == nil {
			time.Sleep(w.config.PollInterval)
			continue
		}

		w.config.Logger.Info("processing", "job_id", job.ID, "type", job.Type)
		w.processJob(ctx, job)
	}
}

func (w *Worker) processJob(ctx context.Context, job *queue.Job) {
	w.inFlight.Add(1)
	defer w.inFlight.Add(-1)

	// Look up handler from registry.
	handler, ok := w.handlers[job.Type]
	if !ok {
		err := fmt.Errorf("unknown job type: %s", job.Type)
		w.config.Logger.Error("job failed", "job_id", job.ID, "error", err)
		if updateErr := w.queue.UpdateStatus(ctx, job.ID, queue.StatusFailed, nil, err.Error()); updateErr != nil {
			w.config.Logger.Error("failed to update job status", "job_id", job.ID, "error", updateErr)
		}
		return
	}

	// Execute handler.
	result, err := handler(ctx, job)
	if err != nil {
		w.config.Logger.Error("job failed", "job_id", job.ID, "error", err)
		if updateErr := w.queue.UpdateStatus(ctx, job.ID, queue.StatusFailed, nil, err.Error()); updateErr != nil {
			w.config.Logger.Error("failed to update job status", "job_id", job.ID, "error", updateErr)
		}
		return
	}

	w.config.Logger.Info("job completed", "job_id", job.ID)
	if updateErr := w.queue.UpdateStatus(ctx, job.ID, queue.StatusCompleted, result, ""); updateErr != nil {
		w.config.Logger.Error("failed to update job status", "job_id", job.ID, "error", updateErr)
	}
}
