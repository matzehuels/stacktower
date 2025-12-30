package api

import (
	"context"
	"net/http"
	"time"
)

// handleHealth returns a simple liveness check response.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleHealthReady checks if dependencies (Redis, MongoDB) are reachable.
// Use this for Kubernetes readiness probes.
func (s *Server) handleHealthReady(w http.ResponseWriter, r *http.Request) {
	hctx := s.handlerContext()
	rctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// Check queue (Redis) connectivity
	if err := hctx.Queue.Ping(rctx); err != nil {
		s.jsonResponse(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status": "unhealthy",
			"check":  "queue",
			"error":  "queue unavailable",
		})
		return
	}

	// Check backend (Redis Index + MongoDB) connectivity
	if err := hctx.Backend.Ping(rctx); err != nil {
		s.jsonResponse(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status": "unhealthy",
			"check":  "backend",
			"error":  "backend unavailable",
		})
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{
		"status": "ready",
		"time":   time.Now().Format(time.RFC3339),
	})
}
