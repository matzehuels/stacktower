package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/matzehuels/stacktower/pkg/infra"
	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/io"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// =============================================================================
// Job Handlers
// =============================================================================

func (w *Worker) processParse(ctx context.Context, job *queue.Job) (map[string]interface{}, error) {
	_, opts, logger, err := w.setupJob(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("setup job: %w", err)
	}
	opts.Logger = logger

	// Pipeline handles caching internally
	g, graphData, cacheKey, cacheHit, err := w.pipeline.Parse(ctx, opts)
	if err != nil {
		return nil, err
	}

	graphID := w.pipeline.GetGraphDocumentID(ctx, cacheKey)

	// Return graph data as nested JSON (not base64)
	var graph interface{}
	if err := json.Unmarshal(graphData, &graph); err != nil {
		return nil, fmt.Errorf("parse graph data: %w", err)
	}

	return map[string]interface{}{
		"graph_id":   graphID,
		"graph_hash": storage.Hash(graphData),
		"graph_data": graph,
		"node_count": g.NodeCount(),
		"edge_count": g.EdgeCount(),
		"cache_hit":  cacheHit,
	}, nil
}

func (w *Worker) processLayout(ctx context.Context, job *queue.Job) (map[string]interface{}, error) {
	payload, opts, logger, err := w.setupJob(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("setup job: %w", err)
	}
	opts.Logger = logger

	// Get graph data with authorization check
	graphData, err := w.getGraphData(ctx, payload)
	if err != nil {
		return nil, err
	}

	g, err := io.ReadJSON(bytes.NewReader(graphData))
	if err != nil {
		return nil, fmt.Errorf("parse graph: %w", err)
	}

	// Pipeline handles caching internally
	layoutData, _, cacheHit, err := w.pipeline.Layout(ctx, g, opts)
	if err != nil {
		return nil, err
	}

	// Return layout as nested JSON
	var layout interface{}
	if err := json.Unmarshal(layoutData, &layout); err != nil {
		return nil, fmt.Errorf("parse layout: %w", err)
	}

	return map[string]interface{}{
		"layout_data": layout,
		"node_count":  g.NodeCount(),
		"edge_count":  g.EdgeCount(),
		"cache_hit":   cacheHit,
	}, nil
}

func (w *Worker) processRender(ctx context.Context, job *queue.Job) (map[string]interface{}, error) {
	payload, opts, logger, err := w.setupJob(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("setup job: %w", err)
	}
	opts.Logger = logger

	// Execute full pipeline (parse → layout → render)
	result, cacheKey, cacheHit, err := w.pipeline.ExecuteFull(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Get graph ID and hash
	graphID := w.pipeline.GetGraphDocumentID(ctx, cacheKey)
	graphHash := pipeline.ComputeGraphHash(result)

	// Store render record via pipeline service
	record, err := w.pipeline.StoreRenderResult(ctx, pipeline.RenderOutput{
		GraphID:   graphID,
		GraphHash: graphHash,
		Result:    result,
		Options:   opts,
		CacheHit:  cacheHit,
		TraceID:   payload.TraceID,
		Repo:      payload.Repo,
	})
	if err != nil {
		return nil, err
	}

	response := map[string]interface{}{
		"render_id":  record.RenderID,
		"graph_id":   graphID,
		"graph_hash": graphHash,
		"node_count": record.NodeCount,
		"edge_count": record.EdgeCount,
		"artifacts":  record.Artifacts, // Already contains URLs from BuildArtifactURLs
		"viz_type":   record.VizType,
		"cache_hit":  record.CacheHit,
		"source": map[string]interface{}{
			"type":     record.Source.Type,
			"language": record.Source.Language,
			"package":  record.Source.Package,
			"repo":     record.Source.Repo,
		},
	}

	// Include full layout data (contains nebraska rankings, blocks, edges, etc.)
	if len(record.LayoutData) > 0 {
		var layout interface{}
		if err := json.Unmarshal(record.LayoutData, &layout); err != nil {
			logger.Error("failed to unmarshal layout data", "error", err, "render_id", record.RenderID)
		} else {
			response["layout"] = layout
		}
	}

	return response, nil
}

// =============================================================================
// Helpers
// =============================================================================

// setupJob extracts payload and options from a job, setting up logging context.
// Returns an error if the payload cannot be unmarshaled (prevents nil pointer issues).
func (w *Worker) setupJob(ctx context.Context, job *queue.Job) (*pipeline.JobPayload, pipeline.Options, *infra.Logger, error) {
	var p pipeline.JobPayload
	if err := unmarshalPayload(job.Payload, &p); err != nil {
		return nil, pipeline.Options{}, nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	ctx = infra.WithTraceID(ctx, p.TraceID)
	logger := infra.LoggerWithTraceID(ctx, w.config.Logger)

	opts := p.ToOptions()
	opts.ValidateAndSetDefaults()

	return &p, opts, logger, nil
}

// getGraphData retrieves graph data from either a stored graph ID or inline data.
func (w *Worker) getGraphData(ctx context.Context, p *pipeline.JobPayload) ([]byte, error) {
	if p.GraphID != "" {
		// Verify user has access to this graph
		stored, err := w.backend.DocumentStore().GetGraphDocScoped(ctx, p.GraphID, p.UserID)
		if errors.Is(err, storage.ErrAccessDenied) {
			return nil, fmt.Errorf("access denied to graph: %s", p.GraphID)
		}
		if err != nil || stored == nil {
			return nil, fmt.Errorf("graph not found: %s", p.GraphID)
		}
		// Convert BSON document to JSON-safe format and serialize
		// (primitive.D -> map, primitive.A -> slice, etc.)
		jsonSafe := storage.ToJSONSafe(stored.Data)
		jsonData, err := json.Marshal(jsonSafe)
		if err != nil {
			return nil, fmt.Errorf("serialize graph data: %w", err)
		}
		return jsonData, nil
	}

	if len(p.GraphData) > 0 {
		return p.GraphData, nil
	}

	return nil, fmt.Errorf("graph_id or graph_data required")
}

// unmarshalPayload converts a map payload to a typed struct.
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
