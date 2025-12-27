package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// handleRender handles POST /api/v1/render.
// This ALWAYS queues a job for async processing (full pipeline: parse → layout → visualize).
// Auth and rate limiting handled by middleware.
func (s *Server) handleRender(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	// Parse request
	var req RenderRequest
	if err := s.decodeJSON(w, r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, errInvalidRequest(err))
		return
	}

	if err := req.Options.ValidateAndSetDefaults(); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()

	// Determine scope and cache key
	var scope storage.Scope
	var pkgOrManifest string
	if req.Manifest == "" { // public package
		scope = storage.ScopeGlobal
		pkgOrManifest = req.Package
	} else {
		scope = storage.ScopeUser
		pkgOrManifest = storage.Hash([]byte(req.Manifest))[:16]
	}
	graphCacheKey := storage.GraphCacheKey(scope, userID, req.Language, pkgOrManifest, req.GraphOptions())

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
		s.logger.Error("failed to enqueue render job", "error", err, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to start render job")
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
	graphEntry, _ := s.backend.Index().GetGraphEntry(ctx, graphCacheKey)
	if graphEntry == nil || graphEntry.IsExpired() {
		return nil
	}

	storedGraph, err := s.backend.DocumentStore().GetGraphDoc(ctx, graphEntry.DocumentID)
	if err != nil || storedGraph == nil {
		return nil
	}

	// Check for cached render
	layoutOpts := req.LayoutOptions()
	renderCacheKey := storage.RenderCacheKey(userID, storedGraph.ContentHash, layoutOpts)

	renderEntry, _ := s.backend.Index().GetRenderEntry(ctx, renderCacheKey)
	if renderEntry == nil || renderEntry.IsExpired() {
		return nil
	}

	storedRender, err := s.backend.DocumentStore().GetRenderDoc(ctx, renderEntry.DocumentID)
	if err != nil || storedRender == nil {
		return nil
	}

	// Build cached response
	resp := s.renderToResponse(storedRender)
	resp.Cached = true
	return &resp
}

// enqueueRenderJob creates a full render job (parse → layout → visualize).
func (s *Server) enqueueRenderJob(ctx context.Context, userID string, req *RenderRequest, scope storage.Scope, graphCacheKey string) (string, error) {
	jobID := generateJobID()

	// Create payload from request options
	req.Options.UserID = userID
	req.Options.Scope = scope
	payload := &pipeline.JobPayload{
		Options:       req.Options,
		GraphCacheKey: graphCacheKey,
	}

	payloadMap, err := payload.ToMap()
	if err != nil {
		return "", fmt.Errorf("serialize payload: %w", err)
	}

	job := &queue.Job{
		ID:        jobID,
		Type:      string(queue.TypeRender),
		Payload:   payloadMap,
		Status:    queue.StatusPending,
		CreatedAt: time.Now(),
	}

	if err := s.queue.Enqueue(ctx, job); err != nil {
		return "", err
	}

	return jobID, nil
}

// handleGetRender handles GET /api/v1/render/{renderID}
// Auth handled by middleware.
func (s *Server) handleGetRender(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	renderID := chi.URLParam(r, "renderID")

	if renderID == "" {
		s.errorResponse(w, http.StatusBadRequest, errFieldRequired("render ID"))
		return
	}

	// Use scoped method - enforces authorization
	render, err := s.backend.DocumentStore().GetRenderDocScoped(r.Context(), renderID, userID)
	if errors.Is(err, storage.ErrAccessDenied) {
		s.errorResponse(w, http.StatusForbidden, "access denied")
		return
	}
	if err != nil {
		s.logger.Error("failed to get render", "error", err, "render_id", renderID, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to retrieve render")
		return
	}
	if render == nil {
		s.errorResponse(w, http.StatusNotFound, errResourceNotFound("render"))
		return
	}

	s.jsonResponse(w, http.StatusOK, s.renderToResponse(render))
}

// handleDeleteRender handles DELETE /api/v1/render/{renderID}
// Auth handled by middleware.
func (s *Server) handleDeleteRender(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	renderID := chi.URLParam(r, "renderID")

	if renderID == "" {
		s.errorResponse(w, http.StatusBadRequest, errFieldRequired("render ID"))
		return
	}

	// Use scoped delete - enforces ownership
	err := s.backend.DocumentStore().DeleteRenderDocScoped(r.Context(), renderID, userID)
	if errors.Is(err, storage.ErrAccessDenied) {
		s.errorResponse(w, http.StatusForbidden, "access denied")
		return
	}
	if err != nil {
		s.logger.Error("failed to delete render", "error", err, "render_id", renderID, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to delete render")
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// renderToResponse converts a Render to a RenderResponse.
func (s *Server) renderToResponse(render *storage.Render) RenderResponse {
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
