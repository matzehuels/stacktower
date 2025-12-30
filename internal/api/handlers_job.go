package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

// handleGetJob handles GET /api/v1/jobs/{jobID}.
// Auth enforced by requireAuth middleware.
//
// Authorization strategy: Returns 404 NotFound (not 403) to prevent enumeration attacks.
// We intentionally don't reveal whether a job exists if the user doesn't own it.
func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()
	userID := getUserID(r)
	jobID := chi.URLParam(r, "jobID")
	if jobID == "" {
		s.errorResponse(w, http.StatusBadRequest, msgFieldRequired("job ID"))
		return
	}

	job, err := ctx.Queue.Get(r.Context(), jobID)
	if err != nil {
		ctx.Logger.Debug("failed to get job", "error", err, "job_id", jobID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusNotFound, msgResourceNotFound("job"))
		return
	}

	// Verify ownership - return 404 (not 403) to prevent resource enumeration
	jobUserID, ok := job.Payload["user_id"].(string)
	if !ok || jobUserID != userID {
		ctx.Logger.Debug("job access denied", "job_id", jobID, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusNotFound, msgResourceNotFound("job"))
		return
	}

	s.jsonResponse(w, http.StatusOK, s.jobToResponse(job))
}

// handleDeleteJob handles DELETE /api/v1/jobs/{jobID}.
// Auth enforced by requireAuth middleware.
//
// Authorization strategy: Returns 404 NotFound (not 403) to prevent enumeration attacks.
// We intentionally don't reveal whether a job exists if the user doesn't own it.
func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()
	userID := getUserID(r)
	jobID := chi.URLParam(r, "jobID")
	if jobID == "" {
		s.errorResponse(w, http.StatusBadRequest, msgFieldRequired("job ID"))
		return
	}

	// Verify ownership first - return 404 (not 403) to prevent resource enumeration
	job, err := ctx.Queue.Get(r.Context(), jobID)
	if err != nil {
		ctx.Logger.Debug("failed to get job for deletion", "error", err, "job_id", jobID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusNotFound, msgResourceNotFound("job"))
		return
	}

	jobUserID, ok := job.Payload["user_id"].(string)
	if !ok || jobUserID != userID {
		ctx.Logger.Debug("job deletion access denied", "job_id", jobID, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusNotFound, msgResourceNotFound("job"))
		return
	}

	// Try to cancel first (if pending)
	if err := ctx.Queue.Cancel(r.Context(), jobID); err == nil {
		s.jsonResponse(w, http.StatusOK, map[string]string{"message": "job cancelled"})
		return
	}

	// Otherwise delete it
	if err := ctx.Queue.Delete(r.Context(), jobID); err != nil {
		ctx.Logger.Error("failed to delete job", "error", err, "job_id", jobID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, errMsgJobDeleteFailed)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"message": "job deleted"})
}

// DefaultJobsPageSize is the default page size for job listings.
const DefaultJobsPageSize = 50

// handleJobsList handles GET /api/v1/jobs.
// Auth enforced by requireAuth middleware.
func (s *Server) handleJobsList(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()
	userID := getUserID(r)
	statusFilter := r.URL.Query().Get("status")
	var statuses []queue.Status
	if statusFilter != "" {
		statuses = []queue.Status{queue.Status(statusFilter)}
	}

	// Parse pagination
	p := parsePagination(r, DefaultJobsPageSize, MaxJobsPageSize)

	// Use ListByUser for efficient user-scoped queries (no in-memory filtering)
	userJobs, err := ctx.Queue.ListByUser(r.Context(), userID, statuses...)
	if err != nil {
		ctx.Logger.Error("failed to list jobs",
			"error", err,
			"user_id", userID,
			"request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, errMsgJobListFailed)
		return
	}

	// Apply pagination to results
	total := len(userJobs)
	start := p.Offset
	if start > total {
		start = total
	}
	end := start + p.Limit
	if end > total {
		end = total
	}
	paginatedJobs := userJobs[start:end]

	responses := make([]JobStatusResponse, len(paginatedJobs))
	for i, job := range paginatedJobs {
		responses[i] = s.jobToResponse(job)
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"jobs":   responses,
		"total":  total,
		"limit":  p.Limit,
		"offset": p.Offset,
	})
}

// checkRateLimit checks if a user can perform an operation.
// Returns true if allowed, false if rate limited (and sends error response).
//
// Note: This method handles the HTTP response for rate limit errors to keep
// middleware code clean. For pure rate limit checks without HTTP handling,
// use ctx.Backend.CheckRateLimit directly.
func (s *Server) checkRateLimit(ctx *HandlerContext, w http.ResponseWriter, r *http.Request, userID string, opType storage.OperationType) bool {
	if err := ctx.Backend.CheckRateLimit(r.Context(), userID, opType, ctx.Quota); err != nil {
		if errors.Is(err, storage.ErrRateLimited) {
			s.errorResponse(w, http.StatusTooManyRequests, errMsgRateLimited)
		} else {
			ctx.Logger.Error("rate limit check failed", "error", err, "user_id", userID, "op_type", opType, "request_id", getRequestID(r))
			s.errorResponse(w, http.StatusInternalServerError, "rate limit check failed")
		}
		return false
	}
	return true
}

func (s *Server) jobToResponse(job *queue.Job) JobStatusResponse {
	resp := JobStatusResponse{
		JobID:     job.ID,
		Type:      job.Type,
		Status:    string(job.Status),
		CreatedAt: job.CreatedAt,
	}

	if job.StartedAt != nil {
		resp.StartedAt = job.StartedAt
	}
	if job.CompletedAt != nil {
		resp.CompletedAt = job.CompletedAt
		duration := job.Duration()
		resp.Duration = &duration
	}
	if job.Result != nil {
		resp.Result = job.Result
	}
	if job.Error != "" {
		resp.Error = &job.Error
	}

	return resp
}
