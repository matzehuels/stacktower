package api

import (
	"net/http"
	"strconv"

	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

// Pagination defaults and limits.
// These values balance API responsiveness with usability:
// - DefaultPageSize (20): Small enough for quick responses, large enough to be useful
// - MaxPageSize (100): Prevents excessive memory usage and response times
// - MaxJobsPageSize (200): Jobs are lightweight, allowing larger pages for admin views
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
	MaxJobsPageSize = 200
)

// Pagination represents parsed pagination parameters.
type Pagination struct {
	Limit  int
	Offset int
}

// parsePagination extracts limit and offset from query parameters.
// Uses defaults if not provided, and caps at maxLimit.
func parsePagination(r *http.Request, defaultLimit, maxLimit int) Pagination {
	p := Pagination{
		Limit:  defaultLimit,
		Offset: 0,
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			p.Limit = parsed
			if p.Limit > maxLimit {
				p.Limit = maxLimit
			}
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			p.Offset = parsed
		}
	}

	return p
}

// =============================================================================
// Artifact Helpers (delegate to storage.BuildArtifactURLs)
// =============================================================================

// buildArtifactURLs converts artifact IDs to API URLs.
// Delegates to storage.BuildArtifactURLs for centralized implementation.
func buildArtifactURLs(artifacts storage.RenderArtifacts) map[string]string {
	return storage.BuildArtifactURLs(artifacts, "")
}

// buildGraphURL returns the API URL for a graph's JSON data.
func buildGraphURL(graphID string) string {
	if graphID == "" {
		return ""
	}
	return "/api/v1/graphs/" + graphID
}

// =============================================================================
// Job Helpers
// =============================================================================

// enqueueJob creates and enqueues a job, handling errors consistently.
// Returns the job on success, or nil if an error occurred (response already sent).
// This centralizes job creation/enqueue logic to ensure consistent error handling.
//
// Uses HandlerContext for business dependencies (Queue, Logger) while keeping
// HTTP response handling on Server.
func (s *Server) enqueueJob(ctx *HandlerContext, w http.ResponseWriter, r *http.Request, jobType queue.Type, req JobRequest) *queue.Job {
	job, err := newJob(jobType, req)
	if err != nil {
		ctx.Logger.Error("failed to create job",
			"error", err,
			"type", jobType,
			"user_id", req.UserID,
			"request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, errMsgJobCreateFailed)
		return nil
	}

	if err := ctx.Queue.Enqueue(r.Context(), job); err != nil {
		ctx.Logger.Error("failed to enqueue job",
			"error", err,
			"type", jobType,
			"job_id", job.ID,
			"user_id", req.UserID,
			"request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, errMsgJobEnqueueFailed)
		return nil
	}

	return job
}
