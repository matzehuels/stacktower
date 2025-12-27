package api

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/matzehuels/stacktower/pkg/core/dag"
	pkgio "github.com/matzehuels/stacktower/pkg/io"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// handleVisualize handles POST /api/v1/visualize
// This runs SYNCHRONOUSLY on the API server since it's just rendering
// from existing layout data (fast operation, no external calls).
// No auth required - it's CPU-bound only with no storage side effects.
func (s *Server) handleVisualize(w http.ResponseWriter, r *http.Request) {
	var req VisualizeRequest
	if err := s.decodeJSON(w, r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, errInvalidJSON(err))
		return
	}

	// Validate layout input requirement
	if len(req.Layout) == 0 {
		s.errorResponse(w, http.StatusBadRequest, errFieldRequired("layout"))
		return
	}

	// Parse optional graph (for popups/nebraska features)
	var g *dag.DAG
	if len(req.Graph) > 0 {
		var err error
		g, err = parseGraphFromJSON(req.Graph)
		if err != nil {
			s.errorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid graph: %v", err))
			return
		}
	}

	// Build pipeline options
	opts := pipeline.Options{
		VizType:   req.VizType,
		Formats:   req.Formats,
		Style:     req.Style,
		ShowEdges: req.ShowEdges,
		Popups:    req.Popups,
		Logger:    s.logger,
	}

	// Render SYNCHRONOUSLY via pipeline service
	artifacts, cached, err := s.pipeline.Visualize(r.Context(), req.Layout, g, opts)
	if err != nil {
		s.logger.Error("visualize failed", "error", err, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "visualization failed")
		return
	}

	// Encode artifacts as base64
	encoded := make(map[string]string)
	for format, data := range artifacts {
		encoded[format] = encodeBase64(data)
	}

	s.jsonResponse(w, http.StatusOK, VisualizeResponse{
		Status:    "completed",
		Artifacts: encoded,
		Cached:    cached,
	})
}

func parseGraphFromJSON(data []byte) (*dag.DAG, error) {
	return pkgio.ReadJSON(bytes.NewReader(data))
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
