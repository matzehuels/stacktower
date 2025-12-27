package api

import (
	"bytes"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

// handleGetArtifact handles GET /api/v1/artifacts/{artifactID}
// Auth handled by middleware.
func (s *Server) handleGetArtifact(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	artifactID := chi.URLParam(r, "artifactID")

	if artifactID == "" {
		s.errorResponse(w, http.StatusBadRequest, errFieldRequired("artifact ID"))
		return
	}

	// Use scoped method - enforces authorization via parent render
	data, err := s.backend.DocumentStore().GetArtifactScoped(r.Context(), artifactID, userID)
	if errors.Is(err, storage.ErrAccessDenied) {
		s.errorResponse(w, http.StatusForbidden, "access denied")
		return
	}
	if err != nil {
		s.logger.Debug("artifact not found", "artifact_id", artifactID, "user_id", userID, "error", err, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusNotFound, errResourceNotFound("artifact"))
		return
	}

	// Detect content type from data
	contentType := http.DetectContentType(data)
	if bytes.HasPrefix(data, []byte("<?xml")) || bytes.HasPrefix(data, []byte("<svg")) {
		contentType = "image/svg+xml"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if _, err := w.Write(data); err != nil {
		s.logger.Error("failed to write artifact response",
			"error", err,
			"artifact_id", artifactID,
			"request_id", getRequestID(r))
	}
}
