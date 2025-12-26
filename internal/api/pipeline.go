package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/matzehuels/stacktower/pkg/cache"
	"github.com/matzehuels/stacktower/pkg/dag"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
	"github.com/matzehuels/stacktower/pkg/pipeline"
	"github.com/matzehuels/stacktower/pkg/queue"
)

// =============================================================================
// Parse endpoint - ASYNC (queues job)
// =============================================================================

// ParseRequest is the request body for POST /api/v1/parse.
type ParseRequest struct {
	Language         string `json:"language"`
	Package          string `json:"package,omitempty"`
	Manifest         string `json:"manifest,omitempty"`
	ManifestFilename string `json:"manifest_filename,omitempty"`
	MaxDepth         int    `json:"max_depth,omitempty"`
	MaxNodes         int    `json:"max_nodes,omitempty"`
	Normalize        bool   `json:"normalize,omitempty"`
	Enrich           bool   `json:"enrich,omitempty"`
	Refresh          bool   `json:"refresh,omitempty"`
}

// ParseResponse is the response for POST /api/v1/parse.
type ParseResponse struct {
	Status    string `json:"status"`           // "pending" or "completed"
	JobID     string `json:"job_id,omitempty"` // For async polling
	Graph     []byte `json:"graph,omitempty"`
	NodeCount int    `json:"node_count,omitempty"`
	EdgeCount int    `json:"edge_count,omitempty"`
	Cached    bool   `json:"cached,omitempty"`
	Error     string `json:"error,omitempty"`
}

// handleParse handles POST /api/v1/parse
// In production API, this QUEUES a job for async processing.
func (s *Server) handleParse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Auth required for parse (uses user quota, stores in user cache)
	sess := s.getSession(r)
	if sess == nil {
		s.errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	userID := fmt.Sprintf("github:%d", sess.User.ID)

	var req ParseRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Validate
	if req.Language == "" {
		s.errorResponse(w, http.StatusBadRequest, "language is required")
		return
	}
	if req.Package == "" && req.Manifest == "" {
		s.errorResponse(w, http.StatusBadRequest, "package or manifest is required")
		return
	}

	// Determine scope and cache key
	var scope cache.Scope
	var pkgOrManifest string
	if req.Manifest == "" {
		scope = cache.ScopeGlobal
		pkgOrManifest = req.Package
	} else {
		scope = cache.ScopeUser
		pkgOrManifest = cache.ContentHash([]byte(req.Manifest))[:16]
	}
	graphCacheKey := cache.GraphCacheKey(scope, userID, req.Language, pkgOrManifest, cache.GraphOptions{
		MaxDepth:  req.MaxDepth,
		MaxNodes:  req.MaxNodes,
		Normalize: req.Normalize,
	})

	ctx := r.Context()

	// Check cache first (fast path)
	if !req.Refresh {
		entry, _ := s.cache.GetGraphEntry(ctx, graphCacheKey)
		if entry != nil && !entry.IsExpired() {
			storedGraph, err := s.cache.GetGraph(ctx, entry.MongoID)
			if err == nil && storedGraph != nil {
				s.jsonResponse(w, http.StatusOK, ParseResponse{
					Status:    "completed",
					Graph:     storedGraph.Data,
					NodeCount: storedGraph.NodeCount,
					EdgeCount: storedGraph.EdgeCount,
					Cached:    true,
				})
				return
			}
		}
	}

	// Queue job for async processing
	jobID := uuid.New().String()
	job := &queue.Job{
		ID:   jobID,
		Type: string(queue.TypeParse),
		Payload: map[string]interface{}{
			"user_id":           userID,
			"language":          req.Language,
			"package":           req.Package,
			"manifest":          req.Manifest,
			"manifest_filename": req.ManifestFilename,
			"max_depth":         req.MaxDepth,
			"max_nodes":         req.MaxNodes,
			"normalize":         req.Normalize,
			"enrich":            req.Enrich,
			"refresh":           req.Refresh,
			"scope":             string(scope),
			"graph_cache_key":   graphCacheKey,
		},
		Status:    queue.StatusPending,
		CreatedAt: time.Now(),
	}

	if err := s.queue.Enqueue(ctx, job); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("enqueue job: %v", err))
		return
	}

	s.jsonResponse(w, http.StatusAccepted, ParseResponse{
		Status: "pending",
		JobID:  jobID,
	})
}

// =============================================================================
// Layout endpoint - ASYNC (queues job)
// =============================================================================

// LayoutRequest is the request body for POST /api/v1/layout.
type LayoutRequest struct {
	Graph     []byte  `json:"graph"`    // Inline graph JSON
	GraphID   string  `json:"graph_id"` // Or reference to stored graph
	VizType   string  `json:"viz_type,omitempty"`
	Width     float64 `json:"width,omitempty"`
	Height    float64 `json:"height,omitempty"`
	Ordering  string  `json:"ordering,omitempty"`
	Merge     bool    `json:"merge,omitempty"`
	Randomize bool    `json:"randomize,omitempty"`
	Seed      uint64  `json:"seed,omitempty"`
	Nebraska  bool    `json:"nebraska,omitempty"`
}

// LayoutResponse is the response for POST /api/v1/layout.
type LayoutResponse struct {
	Status string `json:"status"`           // "pending" or "completed"
	JobID  string `json:"job_id,omitempty"` // For async polling
	Layout []byte `json:"layout,omitempty"`
	Cached bool   `json:"cached,omitempty"`
	Error  string `json:"error,omitempty"`
}

// handleLayout handles POST /api/v1/layout
// In production API, this QUEUES a job for async processing.
func (s *Server) handleLayout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Auth required
	sess := s.getSession(r)
	if sess == nil {
		s.errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	userID := fmt.Sprintf("github:%d", sess.User.ID)

	var req LayoutRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Validate
	if len(req.Graph) == 0 && req.GraphID == "" {
		s.errorResponse(w, http.StatusBadRequest, "graph or graph_id is required")
		return
	}

	ctx := r.Context()

	// Queue job for async processing
	jobID := uuid.New().String()
	job := &queue.Job{
		ID:   jobID,
		Type: string(queue.TypeLayout),
		Payload: map[string]interface{}{
			"user_id":    userID,
			"graph_id":   req.GraphID,
			"graph_data": req.Graph,
			"viz_type":   req.VizType,
			"width":      req.Width,
			"height":     req.Height,
			"ordering":   req.Ordering,
			"merge":      req.Merge,
			"randomize":  req.Randomize,
			"seed":       req.Seed,
			"nebraska":   req.Nebraska,
		},
		Status:    queue.StatusPending,
		CreatedAt: time.Now(),
	}

	if err := s.queue.Enqueue(ctx, job); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("enqueue job: %v", err))
		return
	}

	s.jsonResponse(w, http.StatusAccepted, LayoutResponse{
		Status: "pending",
		JobID:  jobID,
	})
}

// =============================================================================
// Visualize endpoint - SYNC (runs directly on API server)
// =============================================================================

// VisualizeRequest is the request body for POST /api/v1/visualize.
type VisualizeRequest struct {
	Layout    []byte   `json:"layout"`
	Graph     []byte   `json:"graph,omitempty"` // Optional, for popups/nebraska
	VizType   string   `json:"viz_type,omitempty"`
	Formats   []string `json:"formats,omitempty"`
	Style     string   `json:"style,omitempty"`
	ShowEdges bool     `json:"show_edges,omitempty"`
	Popups    bool     `json:"popups,omitempty"`
}

// VisualizeResponse is the response for POST /api/v1/visualize.
type VisualizeResponse struct {
	Status    string            `json:"status"`
	Artifacts map[string]string `json:"artifacts,omitempty"` // format -> base64-encoded data
	Cached    bool              `json:"cached,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// handleVisualize handles POST /api/v1/visualize
// This runs SYNCHRONOUSLY on the API server since it's just rendering
// from existing layout data (fast operation, no external calls).
func (s *Server) handleVisualize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req VisualizeRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Validate
	if len(req.Layout) == 0 {
		s.errorResponse(w, http.StatusBadRequest, "layout is required")
		return
	}

	// Parse optional graph (for popups/nebraska features)
	var g *dag.DAG
	if len(req.Graph) > 0 {
		var err error
		g, err = parseGraphFromJSON(req.Graph)
		if err != nil {
			s.errorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid graph: %v", err))
			return
		}
	}

	// Build pipeline options
	opts := pipeline.Options{
		VizType:   req.VizType,
		Formats:   req.Formats,
		Style:     req.Style,
		ShowEdges: req.ShowEdges,
		Popups:    req.Popups,
		Logger:    s.logger,
	}

	// Render SYNCHRONOUSLY via pipeline service
	artifacts, cached, err := s.pipeline.Visualize(r.Context(), req.Layout, g, opts)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("visualize failed: %v", err))
		return
	}

	// Encode artifacts as base64
	encoded := make(map[string]string)
	for format, data := range artifacts {
		encoded[format] = encodeBase64(data)
	}

	s.jsonResponse(w, http.StatusOK, VisualizeResponse{
		Status:    "completed",
		Artifacts: encoded,
		Cached:    cached,
	})
}

// =============================================================================
// Helpers
// =============================================================================

func parseGraphFromJSON(data []byte) (*dag.DAG, error) {
	return pkgio.ReadJSON(bytes.NewReader(data))
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
