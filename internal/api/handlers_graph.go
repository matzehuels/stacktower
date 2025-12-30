package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

// handleGetGraph handles GET /api/v1/graphs/{graphID}.
// Returns the graph data (nodes + edges) as JSON.
// Uses optional auth - global graphs are accessible without auth.
//
// Authorization strategy: Returns 403 Forbidden for access denied because graphs
// may be shared resources (ScopeGlobal). We want to distinguish "you don't have
// permission" from "this doesn't exist":
//   - Global graphs (ScopeGlobal) are accessible to all users (including unauthenticated)
//   - User graphs (ScopeUser) are only accessible to the owner
func (s *Server) handleGetGraph(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()
	userID := getUserIDOptional(r)
	graphID := chi.URLParam(r, "graphID")

	if graphID == "" {
		s.errorResponse(w, http.StatusBadRequest, msgFieldRequired("graph ID"))
		return
	}

	// Use scoped method - enforces authorization
	graph, err := ctx.Backend.DocumentStore().GetGraphDocScoped(r.Context(), graphID, userID)
	if errors.Is(err, storage.ErrAccessDenied) {
		s.errorResponse(w, http.StatusForbidden, errMsgAccessDenied)
		return
	}
	if err != nil {
		ctx.Logger.Error("failed to get graph", "error", err, "graph_id", graphID, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusNotFound, msgResourceNotFound("graph"))
		return
	}
	if graph == nil {
		s.errorResponse(w, http.StatusNotFound, msgResourceNotFound("graph"))
		return
	}

	// Convert BSON document to JSON-safe format and serialize
	// (primitive.D -> map, primitive.A -> slice, etc.)
	jsonSafe := storage.ToJSONSafe(graph.Data)
	jsonData, err := json.Marshal(jsonSafe)
	if err != nil {
		ctx.Logger.Error("failed to serialize graph", "error", err, "graph_id", graphID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "Failed to serialize graph data")
		return
	}

	// Return the graph data as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(jsonData)
}
