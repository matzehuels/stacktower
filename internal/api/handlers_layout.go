package api

import (
	"net/http"
	"time"

	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// handleLayout handles POST /api/v1/layout
// Auth and rate limiting handled by middleware.
func (s *Server) handleLayout(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req LayoutRequest
	if err := s.decodeJSON(w, r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, errInvalidJSON(err))
		return
	}

	// Validate graph input requirement
	if len(req.Graph) == 0 && req.GraphID == "" {
		s.errorResponse(w, http.StatusBadRequest, errFieldRequired("graph or graph_id"))
		return
	}

	// Use centralized validation for options
	if err := req.Options.ValidateAndSetDefaults(); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()

	// Queue job for async processing
	jobID := generateJobID()
	req.Options.UserID = userID
	payload := &pipeline.JobPayload{
		Options:   req.Options,
		GraphID:   req.GraphID,
		GraphData: req.Graph,
	}

	payloadMap, err := payload.ToMap()
	if err != nil {
		s.logger.Error("failed to serialize payload", "error", err, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	job := &queue.Job{
		ID:        jobID,
		Type:      string(queue.TypeLayout),
		Payload:   payloadMap,
		Status:    queue.StatusPending,
		CreatedAt: time.Now(),
	}

	if err := s.queue.Enqueue(ctx, job); err != nil {
		s.logger.Error("failed to enqueue layout job", "error", err, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to start layout job")
		return
	}

	s.jsonResponse(w, http.StatusAccepted, LayoutResponse{
		Status: "pending",
		JobID:  jobID,
	})
}
