package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/matzehuels/stacktower/pkg/cache"
	"github.com/matzehuels/stacktower/pkg/queue"
)

// =============================================================================
// Render endpoint - ASYNC (queues job)
// =============================================================================

// handleRender handles POST /api/v1/render
// This ALWAYS queues a job for async processing (full pipeline: parse → layout → visualize).
func (s *Server) handleRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Authentication required
	sess := s.getSession(r)
	if sess == nil {
		s.errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	userID := fmt.Sprintf("github:%d", sess.User.ID)

	// Parse request
	var req RenderRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	if err := req.ValidateAndSetDefaults(); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()

	// Determine scope and cache key
	var scope cache.Scope
	var pkgOrManifest string
	if req.IsPublic() {
		scope = cache.ScopeGlobal
		pkgOrManifest = req.Package
	} else {
		scope = cache.ScopeUser
		pkgOrManifest = cache.ContentHash([]byte(req.Manifest))[:16]
	}
	graphCacheKey := cache.GraphCacheKey(scope, userID, req.Language, pkgOrManifest, req.GraphOptions())

	// Check cache first (fast path) - return cached render if available
	if !req.Refresh {
		if resp := s.checkCachedRender(ctx, userID, graphCacheKey, &req); resp != nil {
			s.jsonResponse(w, http.StatusOK, *resp)
			return
		}
	}

	// Queue job for async processing
	jobID, err := s.enqueueRenderJob(ctx, userID, &req, scope, graphCacheKey)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("enqueue job: %v", err))
		return
	}

	s.jsonResponse(w, http.StatusAccepted, RenderResponse{
		Status: "pending",
		JobID:  jobID,
	})
}

// checkCachedRender checks if there's a cached render result available.
func (s *Server) checkCachedRender(ctx context.Context, userID, graphCacheKey string, req *RenderRequest) *RenderResponse {
	// Check for cached graph
	graphEntry, _ := s.cache.GetGraphEntry(ctx, graphCacheKey)
	if graphEntry == nil || graphEntry.IsExpired() {
		return nil
	}

	storedGraph, err := s.cache.GetGraph(ctx, graphEntry.MongoID)
	if err != nil || storedGraph == nil {
		return nil
	}

	// Check for cached render
	layoutOpts := req.LayoutOptions()
	renderCacheKey := cache.RenderCacheKey(userID, storedGraph.ContentHash, layoutOpts)

	renderEntry, _ := s.cache.GetRenderEntry(ctx, renderCacheKey)
	if renderEntry == nil || renderEntry.IsExpired() {
		return nil
	}

	storedRender, err := s.cache.GetRender(ctx, renderEntry.MongoID)
	if err != nil || storedRender == nil {
		return nil
	}

	// Build cached response
	resp := s.renderToResponse(storedRender)
	resp.Cached = true
	return &resp
}

// enqueueRenderJob creates a full render job (parse → layout → visualize).
func (s *Server) enqueueRenderJob(ctx context.Context, userID string, req *RenderRequest, scope cache.Scope, graphCacheKey string) (string, error) {
	jobID := uuid.New().String()

	payload := map[string]interface{}{
		"user_id":           userID,
		"language":          req.Language,
		"package":           req.Package,
		"manifest":          req.Manifest,
		"manifest_filename": req.ManifestFilename,
		"repo":              req.Repo,
		"max_depth":         req.MaxDepth,
		"max_nodes":         req.MaxNodes,
		"normalize":         req.Normalize,
		"enrich":            req.Enrich,
		"scope":             string(scope),
		"graph_cache_key":   graphCacheKey,
		// Layout options
		"viz_type":  req.VizType,
		"width":     req.Width,
		"height":    req.Height,
		"ordering":  req.Ordering,
		"merge":     req.Merge,
		"randomize": req.Randomize,
		"seed":      req.Seed,
		// Render options
		"formats":    req.Formats,
		"style":      req.Style,
		"show_edges": req.ShowEdges,
		"nebraska":   req.Nebraska,
		"popups":     req.Popups,
	}

	job := &queue.Job{
		ID:        jobID,
		Type:      string(queue.TypeRender),
		Payload:   payload,
		Status:    queue.StatusPending,
		CreatedAt: time.Now(),
	}

	if err := s.queue.Enqueue(ctx, job); err != nil {
		return "", err
	}

	return jobID, nil
}

// =============================================================================
// History and render management endpoints
// =============================================================================

// handleHistory handles GET /api/v1/history
func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sess := s.getSession(r)
	if sess == nil {
		s.errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	userID := fmt.Sprintf("github:%d", sess.User.ID)

	// Parse pagination
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
		if limit > 100 {
			limit = 100
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	renders, total, err := s.cache.ListRenders(r.Context(), userID, limit, offset)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("list renders: %v", err))
		return
	}

	items := make([]*RenderHistoryItem, len(renders))
	for i, render := range renders {
		artifacts := make(map[string]string)
		if render.Artifacts.SVG != "" {
			artifacts["svg"] = fmt.Sprintf("/api/v1/artifacts/%s", render.Artifacts.SVG)
		}
		if render.Artifacts.PNG != "" {
			artifacts["png"] = fmt.Sprintf("/api/v1/artifacts/%s", render.Artifacts.PNG)
		}
		if render.Artifacts.PDF != "" {
			artifacts["pdf"] = fmt.Sprintf("/api/v1/artifacts/%s", render.Artifacts.PDF)
		}

		items[i] = &RenderHistoryItem{
			ID: render.ID,
			Source: RenderSourceInfo{
				Type:     render.Source.Type,
				Language: render.Source.Language,
				Package:  render.Source.Package,
				Repo:     render.Source.Repo,
			},
			VizType:   render.LayoutOptions.VizType,
			NodeCount: render.NodeCount,
			EdgeCount: render.EdgeCount,
			Artifacts: artifacts,
			CreatedAt: render.CreatedAt,
		}
	}

	s.jsonResponse(w, http.StatusOK, HistoryResponse{
		Renders: items,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}

// handleRenderByID handles GET/DELETE /api/v1/render/:id
func (s *Server) handleRenderByID(w http.ResponseWriter, r *http.Request) {
	sess := s.getSession(r)
	if sess == nil {
		s.errorResponse(w, http.StatusUnauthorized, "authentication required")
		return
	}
	userID := fmt.Sprintf("github:%d", sess.User.ID)

	renderID := strings.TrimPrefix(r.URL.Path, "/api/v1/render/")
	if renderID == "" {
		s.errorResponse(w, http.StatusBadRequest, "render ID required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		render, err := s.cache.GetRender(r.Context(), renderID)
		if err != nil {
			s.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("get render: %v", err))
			return
		}
		if render == nil || render.UserID != userID {
			s.errorResponse(w, http.StatusNotFound, "render not found")
			return
		}
		s.jsonResponse(w, http.StatusOK, s.renderToResponse(render))

	case http.MethodDelete:
		// Verify ownership first
		render, _ := s.cache.GetRender(r.Context(), renderID)
		if render == nil || render.UserID != userID {
			s.errorResponse(w, http.StatusNotFound, "render not found")
			return
		}
		if err := s.cache.DeleteRender(r.Context(), renderID); err != nil {
			s.errorResponse(w, http.StatusInternalServerError, fmt.Sprintf("delete render: %v", err))
			return
		}
		s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})

	default:
		s.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// renderToResponse converts a Render to a RenderResponse.
func (s *Server) renderToResponse(render *cache.Render) RenderResponse {
	artifacts := make(map[string]string)
	if render.Artifacts.SVG != "" {
		artifacts["svg"] = fmt.Sprintf("/api/v1/artifacts/%s", render.Artifacts.SVG)
	}
	if render.Artifacts.PNG != "" {
		artifacts["png"] = fmt.Sprintf("/api/v1/artifacts/%s", render.Artifacts.PNG)
	}
	if render.Artifacts.PDF != "" {
		artifacts["pdf"] = fmt.Sprintf("/api/v1/artifacts/%s", render.Artifacts.PDF)
	}

	return RenderResponse{
		Status:   "completed",
		RenderID: render.ID,
		Result: &RenderResult{
			Artifacts: artifacts,
			NodeCount: render.NodeCount,
			EdgeCount: render.EdgeCount,
			Source: RenderSourceInfo{
				Type:     render.Source.Type,
				Language: render.Source.Language,
				Package:  render.Source.Package,
				Repo:     render.Source.Repo,
			},
		},
	}
}

// handleArtifactByID handles GET /api/v1/artifacts/:id
func (s *Server) handleArtifactByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.errorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	artifactID := strings.TrimPrefix(r.URL.Path, "/api/v1/artifacts/")
	if artifactID == "" {
		s.errorResponse(w, http.StatusBadRequest, "artifact ID required")
		return
	}

	data, err := s.cache.GetArtifact(r.Context(), artifactID)
	if err != nil {
		s.errorResponse(w, http.StatusNotFound, fmt.Sprintf("artifact not found: %v", err))
		return
	}

	// Detect content type from data
	contentType := http.DetectContentType(data)
	if bytes.HasPrefix(data, []byte("<?xml")) || bytes.HasPrefix(data, []byte("<svg")) {
		contentType = "image/svg+xml"
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}
