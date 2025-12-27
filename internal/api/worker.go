package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/io"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// WorkerConfig holds worker configuration.
type WorkerConfig struct {
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
	config   WorkerConfig
	handlers map[string]JobHandler // Job type -> handler

	// Shutdown coordination.
	inFlight   atomic.Int32
	cancelFunc context.CancelFunc
	done       chan struct{}
}

// NewWorker creates a new worker.
func NewWorker(q queue.Queue, backend *storage.DistributedBackend, config WorkerConfig) *Worker {
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

	// Register job handlers. Adding new job types is now a single line change.
	w.handlers = map[string]JobHandler{
		string(queue.TypeParse):  w.processParse,
		string(queue.TypeLayout): w.processLayout,
		string(queue.TypeRender): w.processRender,
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
	jobTypes := []string{
		string(queue.TypeParse),
		string(queue.TypeLayout),
		string(queue.TypeRender),
	}

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

func (w *Worker) processParse(ctx context.Context, job *queue.Job) (map[string]interface{}, error) {
	var p pipeline.JobPayload
	if err := unmarshalPayload(job.Payload, &p); err != nil {
		return nil, err
	}

	opts := p.ToOptions()
	opts.ValidateAndSetDefaults()
	opts.Logger = w.config.Logger

	// Pipeline handles computation + content-based caching via storage.Backend.
	// It returns the cache key (inputHash) used.
	g, graphData, inputHash, cacheHit, err := w.pipeline.Parse(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Compute content hash for reference.
	graphHash := storage.Hash(graphData)

	// Look up the MongoDB ID from the cache entry (pipeline stored it).
	graphID := ""
	if entry, _ := w.backend.Index().GetGraphEntry(ctx, inputHash); entry != nil {
		graphID = entry.DocumentID
	}

	// If user wants a separate cache key, create a reference (no data duplication).
	if p.GraphCacheKey != "" && p.GraphCacheKey != inputHash && graphID != "" {
		if err := w.backend.Index().SetGraphEntry(ctx, p.GraphCacheKey, &storage.CacheEntry{
			DocumentID: graphID,
			ExpiresAt:  time.Now().Add(storage.GraphTTL),
		}); err != nil {
			w.config.Logger.Warn("failed to set graph cache entry", "cache_key", p.GraphCacheKey, "error", err)
		}
	}

	return map[string]interface{}{
		"graph_id":   graphID,
		"graph_hash": graphHash,
		"node_count": g.NodeCount(),
		"edge_count": g.EdgeCount(),
		"cache_hit":  cacheHit,
	}, nil
}

func (w *Worker) processLayout(ctx context.Context, job *queue.Job) (map[string]interface{}, error) {
	var p pipeline.JobPayload
	if err := unmarshalPayload(job.Payload, &p); err != nil {
		return nil, err
	}

	opts := p.ToOptions()
	opts.ValidateAndSetDefaults()
	opts.Logger = w.config.Logger

	// Get graph data.
	var graphData []byte
	if p.GraphID != "" {
		stored, err := w.backend.DocumentStore().GetGraphDoc(ctx, p.GraphID)
		if err != nil || stored == nil {
			return nil, fmt.Errorf("graph not found: %s", p.GraphID)
		}
		graphData = stored.Data
	} else if len(p.GraphData) > 0 {
		graphData = p.GraphData
	} else {
		return nil, fmt.Errorf("graph_id or graph_data required")
	}

	g, err := io.ReadJSON(bytes.NewReader(graphData))
	if err != nil {
		return nil, err
	}

	// Layout returns layoutKey as second param.
	layoutData, _, cacheHit, err := w.pipeline.Layout(ctx, g, opts)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"layout_data": layoutData,
		"viz_type":    opts.VizType,
		"node_count":  g.NodeCount(),
		"edge_count":  g.EdgeCount(),
		"cache_hit":   cacheHit}, nil
}

func (w *Worker) processRender(ctx context.Context, job *queue.Job) (map[string]interface{}, error) {
	var p pipeline.JobPayload
	if err := unmarshalPayload(job.Payload, &p); err != nil {
		return nil, err
	}

	opts := p.ToOptions()
	opts.ValidateAndSetDefaults()
	opts.Logger = w.config.Logger

	// Pipeline handles computation + content-based caching via storage.Backend.
	// Returns graph cache key (input hash) as second param.
	result, inputHash, cacheHit, err := w.pipeline.ExecuteFull(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Compute content hash for reference (no duplicate storage).
	graphData, _ := io.SerializeDAG(result.Graph)
	graphHash := storage.Hash(graphData)

	// Look up the graph ID from cache (pipeline stored it via Backend).
	graphID := ""
	if entry, _ := w.backend.Index().GetGraphEntry(ctx, inputHash); entry != nil {
		graphID = entry.DocumentID
	}

	// If user wants a separate cache key, create a reference (no data duplication).
	if p.GraphCacheKey != "" && p.GraphCacheKey != inputHash && graphID != "" {
		if err := w.backend.Index().SetGraphEntry(ctx, p.GraphCacheKey, &storage.CacheEntry{
			DocumentID: graphID,
			ExpiresAt:  time.Now().Add(storage.GraphTTL),
		}); err != nil {
			w.config.Logger.Warn("failed to set graph cache entry", "cache_key", p.GraphCacheKey, "error", err)
		}
	}

	// Store user render history (references graph, doesn't duplicate data).
	return w.storeRenderRecord(ctx, p, graphID, graphHash, result, opts, cacheHit)
}

// storeRenderRecord stores user render history. It references the graph by ID
// rather than duplicating data.
func (w *Worker) storeRenderRecord(ctx context.Context, p pipeline.JobPayload, graphID, graphHash string, result *pipeline.Result, opts pipeline.Options, cacheHit bool) (map[string]interface{}, error) {
	layoutOpts := storage.LayoutOptions{
		VizType: opts.VizType, Width: opts.Width, Height: opts.Height,
		Ordering: opts.Ordering, Merge: opts.Merge, Randomize: opts.Randomize, Seed: opts.Seed,
	}
	renderOpts := storage.RenderOptions{
		Formats: opts.Formats, Style: opts.Style, ShowEdges: opts.ShowEdges,
		Nebraska: opts.Nebraska, Popups: opts.Popups,
	}
	source := storage.RenderSource{
		Type: "package", Language: opts.Language, Package: opts.Package, Repo: p.Repo,
	}
	if opts.Manifest != "" {
		source.Type = "manifest"
		source.ManifestFilename = opts.ManifestFilename
	}

	render := &storage.Render{
		ID: uuid.New().String(), UserID: p.UserID, GraphID: graphID, GraphHash: graphHash,
		LayoutOptions: layoutOpts, RenderOptions: renderOpts, Layout: result.LayoutData,
		NodeCount: result.Graph.NodeCount(), EdgeCount: result.Graph.EdgeCount(), Source: source,
	}

	artifactURLs := make(map[string]string)
	for format, data := range result.Artifacts {
		id, err := w.backend.DocumentStore().StoreArtifact(ctx, render.ID, fmt.Sprintf("%s.%s", opts.VizType, format), data)
		if err != nil {
			return nil, err
		}
		artifactURLs[format] = "/api/v1/artifacts/" + id
		switch format {
		case "svg":
			render.Artifacts.SVG = id
		case "png":
			render.Artifacts.PNG = id
		case "pdf":
			render.Artifacts.PDF = id
		}
	}

	if err := w.backend.DocumentStore().StoreRenderDoc(ctx, render); err != nil {
		w.config.Logger.Warn("failed to store render doc", "render_id", render.ID, "error", err)
	}
	if err := w.backend.Index().SetRenderEntry(ctx, storage.RenderCacheKey(p.UserID, graphHash, layoutOpts), &storage.CacheEntry{
		DocumentID: render.ID, ExpiresAt: time.Now().Add(storage.RenderTTL),
	}); err != nil {
		w.config.Logger.Warn("failed to set render cache entry", "render_id", render.ID, "error", err)
	}

	return map[string]interface{}{
		"render_id":  render.ID,
		"graph_id":   graphID,
		"graph_hash": graphHash,
		"node_count": result.Graph.NodeCount(),
		"edge_count": result.Graph.EdgeCount(),
		"artifacts":  artifactURLs,
		"cache_hit":  cacheHit,
	}, nil
}

// unmarshalPayload converts a map payload to a typed struct.
// This uses JSON round-tripping which is intentional for handling
// the dynamic payload structure from the queue.
func unmarshalPayload(payload map[string]interface{}, v interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	return nil
}
