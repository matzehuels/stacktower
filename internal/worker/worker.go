// Package worker provides the job processing engine for Stacktower.
//
// Workers dequeue jobs from the queue and process them using pipeline.Service.
// They handle: parse, layout, and render jobs. Visualize runs sync on the API.
//
// # Architecture
//
// The worker separates two concerns:
//
//  1. Content deduplication: Handled by pipeline.Service via artifact.Backend.
//     Same inputs produce the same outputs - no recomputation needed.
//
//  2. User history: Handled by the worker via cache.Cache.
//     Stores Render records (who rendered what, when) with references to graphs.
//     No data duplication - just metadata and pointers.
//
// Workers can run in the same process as the API server (local mode) or as
// separate processes/containers (distributed mode).
package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"

	"github.com/matzehuels/stacktower/internal/jobs"
	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/infra/artifact"
	"github.com/matzehuels/stacktower/pkg/infra/cache"
	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/io"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// Config holds worker configuration.
type Config struct {
	Concurrency  int
	PollInterval time.Duration
	Logger       *log.Logger
}

// Worker processes jobs from a queue.
type Worker struct {
	queue    queue.Queue
	cache    cache.Cache
	pipeline *pipeline.Service
	config   Config
}

// New creates a new worker.
func New(q queue.Queue, c cache.Cache, config Config) *Worker {
	if config.Concurrency <= 0 {
		config.Concurrency = 1
	}
	if config.PollInterval == 0 {
		config.PollInterval = 1 * time.Second
	}
	if config.Logger == nil {
		config.Logger = log.NewWithOptions(os.Stderr, log.Options{Level: log.InfoLevel})
	}

	return &Worker{
		queue:    q,
		cache:    c,
		pipeline: pipeline.NewService(artifact.NewProdBackend(c)),
		config:   config,
	}
}

// Start begins processing jobs. Blocks until context is cancelled.
func (w *Worker) Start(ctx context.Context) error {
	w.config.Logger.Info("worker starting", "concurrency", w.config.Concurrency)

	var wg sync.WaitGroup
	for i := 0; i < w.config.Concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			w.loop(ctx, id)
		}(i)
	}

	wg.Wait()
	w.config.Logger.Info("worker stopped")
	return nil
}

func (w *Worker) loop(ctx context.Context, workerID int) {
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
		w.process(ctx, job)
	}
}

func (w *Worker) process(ctx context.Context, job *queue.Job) {
	var result map[string]interface{}
	var err error

	switch job.Type {
	case string(queue.TypeParse):
		result, err = w.processParse(ctx, job)
	case string(queue.TypeLayout):
		result, err = w.processLayout(ctx, job)
	case string(queue.TypeRender):
		result, err = w.processRender(ctx, job)
	default:
		err = fmt.Errorf("unknown job type: %s", job.Type)
	}

	if err != nil {
		w.config.Logger.Error("job failed", "job_id", job.ID, "error", err)
		w.queue.UpdateStatus(ctx, job.ID, queue.StatusFailed, nil, err.Error())
		return
	}

	w.config.Logger.Info("job completed", "job_id", job.ID)
	w.queue.UpdateStatus(ctx, job.ID, queue.StatusCompleted, result, "")
}

// =============================================================================
// Job handlers (no abstraction needed - just methods)
// =============================================================================

func (w *Worker) processParse(ctx context.Context, job *queue.Job) (map[string]interface{}, error) {
	var p struct {
		jobs.ParsePayload
		UserID        string `json:"user_id"`
		Scope         string `json:"scope"`
		GraphCacheKey string `json:"graph_cache_key"`
	}
	if err := unmarshal(job.Payload, &p); err != nil {
		return nil, err
	}
	if err := p.ValidateAndSetDefaults(); err != nil {
		return nil, err
	}

	opts := pipeline.Options{
		Language:         p.Language,
		Package:          p.Package,
		Manifest:         p.Manifest,
		ManifestFilename: p.ManifestFilename,
		MaxDepth:         p.MaxDepth,
		MaxNodes:         p.MaxNodes,
		Normalize:        p.Normalize,
		Enrich:           p.Enrich,
		Refresh:          p.Refresh,
		Logger:           w.config.Logger,
	}

	// Pipeline handles computation + content-based caching via artifact.Backend
	g, graphData, cacheHit, err := w.pipeline.Parse(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Compute content hash for reference
	graphHash := cache.ContentHash(graphData)

	// Compute the same input hash that pipeline used to find the graph ID
	inputHash := artifact.HashJSON(pipeline.ParseInputs{
		Language:         opts.Language,
		Package:          opts.Package,
		ManifestHash:     artifact.Hash([]byte(opts.Manifest)),
		ManifestFilename: opts.ManifestFilename,
		MaxDepth:         opts.MaxDepth,
		MaxNodes:         opts.MaxNodes,
		Normalize:        opts.Normalize,
		Enrich:           opts.Enrich,
	})

	// Look up the MongoDB ID from the cache entry (pipeline stored it)
	graphID := ""
	if entry, _ := w.cache.GetGraphEntry(ctx, inputHash); entry != nil {
		graphID = entry.MongoID
	}

	// If user wants a separate cache key, create a reference (no data duplication)
	if p.GraphCacheKey != "" && p.GraphCacheKey != inputHash && graphID != "" {
		w.cache.SetGraphEntry(ctx, p.GraphCacheKey, &cache.CacheEntry{
			MongoID:   graphID,
			ExpiresAt: time.Now().Add(cache.GraphTTL),
		})
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
	var p struct {
		jobs.LayoutPayload
		UserID    string `json:"user_id"`
		GraphID   string `json:"graph_id"`
		GraphData []byte `json:"graph_data"`
	}
	if err := unmarshal(job.Payload, &p); err != nil {
		return nil, err
	}
	if err := p.ValidateAndSetDefaults(); err != nil {
		return nil, err
	}

	// Get graph data
	var graphData []byte
	if p.GraphID != "" {
		stored, err := w.cache.GetGraph(ctx, p.GraphID)
		if err != nil || stored == nil {
			return nil, fmt.Errorf("graph not found: %s", p.GraphID)
		}
		graphData = stored.Data
	} else if p.GraphPath != "" {
		stored, err := w.cache.GetGraph(ctx, p.GraphPath)
		if err != nil || stored == nil {
			return nil, fmt.Errorf("graph not found: %s", p.GraphPath)
		}
		graphData = stored.Data
	} else if len(p.GraphData) > 0 {
		graphData = p.GraphData
	} else {
		return nil, fmt.Errorf("graph_id, graph_path, or graph_data required")
	}

	g, err := io.ReadJSON(bytes.NewReader(graphData))
	if err != nil {
		return nil, err
	}

	opts := pipeline.Options{
		VizType:   p.VizType,
		Width:     p.Width,
		Height:    p.Height,
		Ordering:  p.Ordering,
		Merge:     p.Merge,
		Randomize: p.Randomize,
		Seed:      p.Seed,
		Logger:    w.config.Logger,
	}

	layoutData, _, err := w.pipeline.Layout(ctx, g, opts)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"layout_data": layoutData,
		"viz_type":    p.VizType,
		"node_count":  g.NodeCount(),
		"edge_count":  g.EdgeCount(),
	}, nil
}

func (w *Worker) processRender(ctx context.Context, job *queue.Job) (map[string]interface{}, error) {
	var p struct {
		jobs.RenderPayload
		UserID        string `json:"user_id"`
		Scope         string `json:"scope"`
		GraphCacheKey string `json:"graph_cache_key"`
		Repo          string `json:"repo"`
	}
	if err := unmarshal(job.Payload, &p); err != nil {
		return nil, err
	}
	if err := p.ValidateAndSetDefaults(); err != nil {
		return nil, err
	}

	opts := p.ToPipelineOptions()
	opts.Logger = w.config.Logger

	// Pipeline handles computation + content-based caching via artifact.Backend
	result, cacheHit, err := w.pipeline.ExecuteFull(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Compute content hash for reference (no duplicate storage!)
	graphData, _ := serializeGraph(result.Graph)
	graphHash := cache.ContentHash(graphData)

	// Compute input hash to look up the graph ID from pipeline's storage
	inputHash := artifact.HashJSON(pipeline.ParseInputs{
		Language:         opts.Language,
		Package:          opts.Package,
		ManifestHash:     artifact.Hash([]byte(opts.Manifest)),
		ManifestFilename: opts.ManifestFilename,
		MaxDepth:         opts.MaxDepth,
		MaxNodes:         opts.MaxNodes,
		Normalize:        opts.Normalize,
		Enrich:           opts.Enrich,
	})

	// Look up the graph ID from cache (pipeline stored it via Backend)
	graphID := ""
	if entry, _ := w.cache.GetGraphEntry(ctx, inputHash); entry != nil {
		graphID = entry.MongoID
	}

	// If user wants a separate cache key, create a reference (no data duplication)
	if p.GraphCacheKey != "" && p.GraphCacheKey != inputHash && graphID != "" {
		w.cache.SetGraphEntry(ctx, p.GraphCacheKey, &cache.CacheEntry{
			MongoID:   graphID,
			ExpiresAt: time.Now().Add(cache.GraphTTL),
		})
	}

	// Store user render history (references graph, doesn't duplicate data)
	return w.storeRenderRecord(ctx, p.UserID, graphID, graphHash, result, p, cacheHit)
}

// storeRenderRecord stores user render history (references graph by ID, no data duplication).
func (w *Worker) storeRenderRecord(ctx context.Context, userID, graphID, graphHash string, result *pipeline.Result, p struct {
	jobs.RenderPayload
	UserID        string `json:"user_id"`
	Scope         string `json:"scope"`
	GraphCacheKey string `json:"graph_cache_key"`
	Repo          string `json:"repo"`
}, cacheHit bool) (map[string]interface{}, error) {
	layoutOpts := cache.LayoutOptions{
		VizType: p.VizType, Width: p.Width, Height: p.Height,
		Ordering: p.Ordering, Merge: p.Merge, Randomize: p.Randomize, Seed: p.Seed,
	}
	renderOpts := cache.RenderOptions{
		Formats: p.Formats, Style: p.Style, ShowEdges: p.ShowEdges,
		Nebraska: p.Nebraska, Popups: p.Popups,
	}
	source := cache.RenderSource{
		Type: "package", Language: p.Language, Package: p.Package, Repo: p.Repo,
	}
	if p.Manifest != "" {
		source.Type = "manifest"
		source.ManifestFilename = p.ManifestFilename
	}

	render := &cache.Render{
		ID: uuid.New().String(), UserID: userID, GraphID: graphID, GraphHash: graphHash,
		LayoutOptions: layoutOpts, RenderOptions: renderOpts, Layout: result.LayoutData,
		NodeCount: result.Graph.NodeCount(), EdgeCount: result.Graph.EdgeCount(), Source: source,
	}

	artifactURLs := make(map[string]string)
	for format, data := range result.Artifacts {
		id, err := w.cache.StoreArtifact(ctx, render.ID, fmt.Sprintf("%s.%s", p.VizType, format), data)
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

	_ = w.cache.StoreRender(ctx, render)
	w.cache.SetRenderEntry(ctx, cache.RenderCacheKey(userID, graphHash, layoutOpts), &cache.CacheEntry{
		MongoID: render.ID, ExpiresAt: time.Now().Add(cache.RenderTTL),
	})

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

// =============================================================================
// Helpers
// =============================================================================

func unmarshal(payload map[string]interface{}, v interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func serializeGraph(g *dag.DAG) ([]byte, error) {
	var buf bytes.Buffer
	if err := io.WriteJSON(g, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
