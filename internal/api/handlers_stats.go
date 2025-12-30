package api

import (
	"net/http"
)

// PublicStats represents aggregate platform statistics.
// These are shown on the landing page to demonstrate platform growth.
type PublicStats struct {
	TotalTowers       int64 `json:"total_renders"`      // Unique packages/manifests visualized
	TotalDependencies int64 `json:"total_dependencies"` // Unique dependencies across all graphs
	TotalUsers        int64 `json:"total_users"`        // Unique users
}

// handlePublicStats handles GET /api/v1/stats (no auth required).
// Returns aggregate platform statistics for the landing page.
func (s *Server) handlePublicStats(w http.ResponseWriter, r *http.Request) {
	hctx := s.handlerContext()
	rctx := r.Context()

	// Get unique towers (distinct language+package combinations, not counting viz types separately)
	totalTowers, err := hctx.Backend.DocumentStore().CountUniqueTowers(rctx)
	if err != nil {
		hctx.Logger.Error("failed to count unique towers", "error", err, "request_id", getRequestID(r))
		totalTowers = 0
	}

	// Get total unique users (from renders)
	totalUsers, err := hctx.Backend.DocumentStore().CountUniqueUsers(rctx)
	if err != nil {
		hctx.Logger.Error("failed to count users", "error", err, "request_id", getRequestID(r))
		totalUsers = 0
	}

	// Get unique dependencies (distinct node IDs across all graphs)
	totalDeps, err := hctx.Backend.DocumentStore().CountUniqueDependencies(rctx)
	if err != nil {
		hctx.Logger.Error("failed to count unique dependencies", "error", err, "request_id", getRequestID(r))
		totalDeps = 0
	}

	s.jsonResponse(w, http.StatusOK, PublicStats{
		TotalTowers:       totalTowers,
		TotalDependencies: totalDeps,
		TotalUsers:        totalUsers,
	})
}
