package api

import (
	"net/http"
	"time"

	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// handleParse handles POST /api/v1/parse
// Auth and rate limiting handled by middleware.
func (s *Server) handleParse(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req ParseRequest
	if err := s.decodeJSON(w, r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, errInvalidJSON(err))
		return
	}

	// Use centralized validation from pipeline.Options
	if err := req.Options.ValidateAndSetDefaults(); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Determine scope and cache key
	var scope storage.Scope
	var pkgOrManifest string
	if req.Manifest == "" {
		scope = storage.ScopeGlobal
		pkgOrManifest = req.Package
	} else {
		scope = storage.ScopeUser
		pkgOrManifest = storage.Hash([]byte(req.Manifest))[:16]
	}
	graphCacheKey := storage.GraphCacheKey(scope, userID, req.Language, pkgOrManifest, storage.GraphOptions{
		MaxDepth:  req.MaxDepth,
		MaxNodes:  req.MaxNodes,
		Normalize: req.Normalize,
	})

	ctx := r.Context()

	// Check cache first (fast path)
	if !req.Refresh {
		entry, _ := s.backend.Index().GetGraphEntry(ctx, graphCacheKey)
		if entry != nil && !entry.IsExpired() {
			storedGraph, err := s.backend.DocumentStore().GetGraphDoc(ctx, entry.DocumentID)
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
	jobID := generateJobID()
	req.Options.UserID = userID
	req.Options.Scope = scope
	payload := &pipeline.JobPayload{
		Options:       req.Options,
		GraphCacheKey: graphCacheKey,
	}

	payloadMap, err := payload.ToMap()
	if err != nil {
		s.logger.Error("failed to serialize payload", "error", err, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	job := &queue.Job{
		ID:        jobID,
		Type:      string(queue.TypeParse),
		Payload:   payloadMap,
		Status:    queue.StatusPending,
		CreatedAt: time.Now(),
	}

	if err := s.queue.Enqueue(ctx, job); err != nil {
		s.logger.Error("failed to enqueue parse job", "error", err, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to start parse job")
		return
	}

	s.jsonResponse(w, http.StatusAccepted, ParseResponse{
		Status: "pending",
		JobID:  jobID,
	})
}
