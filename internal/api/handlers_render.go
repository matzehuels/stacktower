package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/matzehuels/stacktower/pkg/infra/queue"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// handleRender handles POST /api/v1/render.
// This ALWAYS queues a job for async processing (full pipeline: parse → layout → visualize).
// Auth enforced by requireAuth middleware.
func (s *Server) handleRender(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()
	userID := getUserID(r)

	var req RenderRequest
	if err := s.decodeJSON(w, r, &req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, msgInvalidRequest(err))
		return
	}

	// Set tower-specific defaults and validate
	setWebAPIDefaults(&req.Options)
	if err := req.Options.ValidateAndSetDefaults(); err != nil {
		s.errorResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Options.UserID = userID

	// Fast path: check cached render
	if !req.Refresh {
		if record := ctx.Pipeline.GetCachedRenderRecord(r.Context(), req.Options); record != nil {
			s.jsonResponse(w, http.StatusOK, renderRecordToResponse(record, true))
			return
		}
	}

	// Queue job for async processing
	job := s.enqueueJob(ctx, w, r, queue.TypeRender, JobRequest{
		UserID:  userID,
		TraceID: getRequestID(r),
		Options: req.Options,
	})
	if job == nil {
		return // Error response already sent
	}

	s.jsonResponse(w, http.StatusAccepted, RenderResponse{
		Status: "pending",
		JobID:  job.ID,
	})
}

// handleGetRender handles GET /api/v1/render/{renderID}.
// Uses optionalAuth middleware to allow both authenticated and unauthenticated access.
//
// Authorization strategy:
// - Canonical renders (user_id = "") are accessible to everyone (public)
// - User-scoped renders are only accessible to their owner (requires auth)
// Returns 403 Forbidden for access denied (not 404) for clear feedback on ownership issues.
func (s *Server) handleGetRender(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()
	userID := getUserID(r) // May be empty for unauthenticated users
	renderID := chi.URLParam(r, "renderID")

	if renderID == "" {
		s.errorResponse(w, http.StatusBadRequest, msgFieldRequired("render ID"))
		return
	}

	// Get the render without scoping (to check if it's canonical/public)
	render, err := ctx.Backend.DocumentStore().GetRenderDoc(r.Context(), renderID)
	if err != nil {
		ctx.Logger.Error("failed to get render", "error", err, "render_id", renderID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, errMsgRenderGetFailed)
		return
	}
	if render == nil {
		s.errorResponse(w, http.StatusNotFound, msgResourceNotFound("render"))
		return
	}

	// Check authorization:
	// - Canonical renders (user_id = "") are public and accessible to everyone
	// - User-scoped renders require authentication and ownership match
	if render.UserID != "" {
		if userID == "" {
			// User-scoped render accessed by unauthenticated user
			s.errorResponse(w, http.StatusForbidden, "This visualization is private. Please sign in to view it.")
			return
		}
		if render.UserID != userID {
			// User-scoped render accessed by different user
			s.errorResponse(w, http.StatusForbidden, errMsgAccessDenied)
			return
		}
	}

	s.jsonResponse(w, http.StatusOK, storageRenderToResponse(render))
}

// handleDeleteRender handles DELETE /api/v1/render/{renderID}.
// Auth enforced by requireAuth middleware.
//
// Authorization strategy: Returns 403 Forbidden for access denied (not 404) because
// renders are user-owned resources where we want clear feedback on ownership issues.
func (s *Server) handleDeleteRender(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()
	userID := getUserID(r)
	renderID := chi.URLParam(r, "renderID")

	if renderID == "" {
		s.errorResponse(w, http.StatusBadRequest, msgFieldRequired("render ID"))
		return
	}

	err := ctx.Backend.DocumentStore().DeleteRenderDocScoped(r.Context(), renderID, userID)
	if errors.Is(err, storage.ErrAccessDenied) {
		s.errorResponse(w, http.StatusForbidden, errMsgAccessDenied)
		return
	}
	if err != nil {
		ctx.Logger.Error("failed to delete render", "error", err, "render_id", renderID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, errMsgRenderDeleteFailed)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// =============================================================================
// Defaults and Response Transformations
// =============================================================================

// setWebAPIDefaults sets sensible defaults for web API render requests.
func setWebAPIDefaults(opts *pipeline.Options) {
	if opts.VizType == "" {
		opts.VizType = pipeline.VizTypeTower
	}
	// Ensure JSON format is always included (needed for dependency list with brittle flags)
	if len(opts.Formats) == 0 {
		opts.Formats = []string{"svg", "json"}
	} else if !containsFormat(opts.Formats, "json") {
		opts.Formats = append(opts.Formats, "json")
	}
	// Tower-specific defaults for better web visualization
	if opts.VizType == pipeline.VizTypeTower {
		opts.Randomize = true // Vary block widths for visual interest
		opts.Merge = true     // Merge subdividers for cleaner towers
		opts.Popups = false   // Disable popups - frontend handles interaction via dependency list
	}
}

// containsFormat checks if a format is in the list.
func containsFormat(formats []string, target string) bool {
	for _, f := range formats {
		if f == target {
			return true
		}
	}
	return false
}

// renderRecordToResponse converts a pipeline.RenderRecord to RenderResponse.
func renderRecordToResponse(record *pipeline.RenderRecord, cached bool) *RenderResponse {
	// Parse layout data to interface{} for JSON response
	var layout interface{}
	if len(record.LayoutData) > 0 {
		_ = json.Unmarshal(record.LayoutData, &layout)
	}

	return &RenderResponse{
		Status:   "completed",
		RenderID: record.RenderID,
		Cached:   cached,
		Result: &RenderResult{
			Artifacts: record.Artifacts,
			NodeCount: record.NodeCount,
			EdgeCount: record.EdgeCount,
			VizType:   record.VizType,
			Source:    ToRenderSourceInfo(record.Source),
			Layout:    layout,
		},
	}
}

// storageRenderToResponse converts a storage.Render to RenderResponse.
func storageRenderToResponse(render *storage.Render) *RenderResponse {
	return &RenderResponse{
		Status:   "completed",
		RenderID: render.ID,
		Result: &RenderResult{
			Artifacts: storage.BuildArtifactURLs(render.Artifacts, render.GraphID),
			NodeCount: render.NodeCount,
			EdgeCount: render.EdgeCount,
			VizType:   render.LayoutOptions.VizType,
			Source:    ToRenderSourceInfo(render.Source),
			Layout:    storage.ToJSONSafe(render.Layout),
		},
	}
}

// Ensure queue types are used
var _ = queue.TypeRender
