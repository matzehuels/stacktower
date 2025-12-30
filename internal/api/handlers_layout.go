package api

import (
	"net/http"

	"github.com/matzehuels/stacktower/pkg/infra/queue"
)

// handleLayout handles POST /api/v1/layout.
// Auth enforced by requireAuth middleware.
func (s *Server) handleLayout(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()
	userID := getUserID(r)

	var req LayoutRequest
	if err := s.decodeJSON(w, r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, msgInvalidJSON(err))
		return
	}

	// Validate graph input requirement
	if len(req.Graph) == 0 && req.GraphID == "" {
		s.errorResponse(w, http.StatusBadRequest, msgFieldRequired("graph or graph_id"))
		return
	}

	if err := req.Options.ValidateForLayout(); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Options.UserID = userID

	// Queue job for async processing
	job := s.enqueueJob(ctx, w, r, queue.TypeLayout, JobRequest{
		UserID:    userID,
		TraceID:   getRequestID(r),
		Options:   req.Options,
		GraphID:   req.GraphID,
		GraphData: req.Graph,
	})
	if job == nil {
		return // Error response already sent
	}

	s.jsonResponse(w, http.StatusAccepted, LayoutResponse{
		Status: "pending",
		JobID:  job.ID,
	})
}
