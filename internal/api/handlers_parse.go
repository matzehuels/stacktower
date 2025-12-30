package api

import (
	"net/http"

	"github.com/matzehuels/stacktower/pkg/infra/queue"
)

// handleParse handles POST /api/v1/parse.
// Auth enforced by requireAuth middleware.
func (s *Server) handleParse(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()
	userID := getUserID(r)

	var req ParseRequest
	if err := s.decodeJSON(w, r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, msgInvalidJSON(err))
		return
	}

	if err := req.Options.ValidateAndSetDefaults(); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Options.UserID = userID

	// Fast path: check cached graph
	if !req.Refresh {
		if g, data, _, hit := ctx.Pipeline.GetCachedGraph(r.Context(), req.Options); hit {
			s.jsonResponse(w, http.StatusOK, ParseResponse{
				Status:    "completed",
				Graph:     data,
				NodeCount: g.NodeCount(),
				EdgeCount: g.EdgeCount(),
				Cached:    true,
			})
			return
		}
	}

	// Queue job for async processing
	job := s.enqueueJob(ctx, w, r, queue.TypeParse, JobRequest{
		UserID:  userID,
		TraceID: getRequestID(r),
		Options: req.Options,
	})
	if job == nil {
		return // Error response already sent
	}

	s.jsonResponse(w, http.StatusAccepted, ParseResponse{
		Status: "pending",
		JobID:  job.ID,
	})
}
