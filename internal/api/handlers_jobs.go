package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	jobID := chi.URLParam(r, "jobID")
	if jobID == "" {
		s.errorResponse(w, http.StatusBadRequest, errFieldRequired("job ID"))
		return
	}

	job, err := s.queue.Get(r.Context(), jobID)
	if err != nil {
		s.logger.Debug("failed to get job", "error", err, "job_id", jobID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusNotFound, errResourceNotFound("job"))
		return
	}

	// Verify ownership - return same error as not found to prevent enumeration
	jobUserID, ok := job.Payload["user_id"].(string)
	if !ok || jobUserID != userID {
		s.logger.Debug("job access denied", "job_id", jobID, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusNotFound, errResourceNotFound("job"))
		return
	}

	s.jsonResponse(w, http.StatusOK, s.jobToResponse(job))
}

func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	jobID := chi.URLParam(r, "jobID")
	if jobID == "" {
		s.errorResponse(w, http.StatusBadRequest, errFieldRequired("job ID"))
		return
	}

	// Verify ownership first - return same error as not found to prevent enumeration
	job, err := s.queue.Get(r.Context(), jobID)
	if err != nil {
		s.logger.Debug("failed to get job for deletion", "error", err, "job_id", jobID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusNotFound, errResourceNotFound("job"))
		return
	}

	jobUserID, ok := job.Payload["user_id"].(string)
	if !ok || jobUserID != userID {
		s.logger.Debug("job deletion access denied", "job_id", jobID, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusNotFound, errResourceNotFound("job"))
		return
	}

	// Try to cancel first (if pending)
	if err := s.queue.Cancel(r.Context(), jobID); err == nil {
		s.jsonResponse(w, http.StatusOK, map[string]string{"message": "job cancelled"})
		return
	}

	// Otherwise delete it
	if err := s.queue.Delete(r.Context(), jobID); err != nil {
		s.logger.Error("failed to delete job", "error", err, "job_id", jobID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to delete job")
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"message": "job deleted"})
}

// DefaultJobsPageSize is the default page size for job listings.
const DefaultJobsPageSize = 50

func (s *Server) handleJobsList(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	statusFilter := r.URL.Query().Get("status")
	var statuses []queue.Status
	if statusFilter != "" {
		statuses = []queue.Status{queue.Status(statusFilter)}
	}

	// Parse pagination
	p := parsePagination(r, DefaultJobsPageSize, MaxJobsPageSize)

	// Use ListByUser for efficient user-scoped queries (no in-memory filtering)
	userJobs, err := s.queue.ListByUser(r.Context(), userID, statuses...)
	if err != nil {
		s.logger.Error("failed to list jobs",
			"error", err,
			"user_id", userID,
			"request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to list jobs")
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
func (s *Server) checkRateLimit(w http.ResponseWriter, r *http.Request, userID string, opType storage.OperationType) bool {
	if err := s.backend.CheckRateLimit(r.Context(), userID, opType, s.quota); err != nil {
		if errors.Is(err, storage.ErrRateLimited) {
			s.errorResponse(w, http.StatusTooManyRequests, "rate limit exceeded - try again later")
		} else {
			s.logger.Error("rate limit check failed", "error", err, "user_id", userID, "op_type", opType, "request_id", getRequestID(r))
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
