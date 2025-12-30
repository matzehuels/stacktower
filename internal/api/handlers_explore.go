package api

import (
	"net/http"
	"strconv"
)

// =============================================================================
// Explore Types
// =============================================================================

// ExploreResponse is the response for GET /api/v1/explore.
type ExploreResponse struct {
	Entries []ExploreEntry `json:"entries"`
	Total   int64          `json:"total"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
}

// ExploreEntry is a grouped package entry in the explore view.
type ExploreEntry struct {
	Source     RenderSourceInfo `json:"source"`
	NodeCount  int              `json:"node_count"`
	EdgeCount  int              `json:"edge_count"`
	CreatedAt  string           `json:"created_at"`
	VizTypes   []ExploreVizType `json:"viz_types"`
	Popularity int              `json:"popularity"` // Users with this in library
	InLibrary  bool             `json:"in_library"` // Whether current user has this saved
}

// ExploreVizType represents a single viz type within an explore entry.
type ExploreVizType struct {
	VizType     string `json:"viz_type"`
	RenderID    string `json:"render_id"`
	GraphID     string `json:"graph_id,omitempty"`
	ArtifactSVG string `json:"artifact_svg,omitempty"`
	ArtifactPNG string `json:"artifact_png,omitempty"`
	ArtifactPDF string `json:"artifact_pdf,omitempty"`
}

// =============================================================================
// Explore Handler
// =============================================================================

// handleExplore handles GET /api/v1/explore.
// Returns public towers for discovery.
// Query params:
//   - language: filter by language
//   - sort_by: "popular" (default) or "recent"
//   - limit: max results (default 20, max 100)
//   - offset: pagination offset
func (s *Server) handleExplore(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()
	docstore := ctx.Backend.DocumentStore()

	// Get user ID if authenticated (for in_library field)
	userID := getUserIDOptional(r)

	// Parse query params
	language := r.URL.Query().Get("language")
	sortBy := r.URL.Query().Get("sort_by")
	if sortBy != "recent" {
		sortBy = "popular" // default
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	entries, total, err := docstore.ListExplore(r.Context(), language, sortBy, limit, offset)
	if err != nil {
		ctx.Logger.Error("failed to get explore entries", "error", err, "language", language)
		s.jsonResponse(w, http.StatusOK, ExploreResponse{
			Entries: []ExploreEntry{},
			Total:   0,
			Limit:   limit,
			Offset:  offset,
		})
		return
	}

	// Convert to API type
	apiEntries := make([]ExploreEntry, len(entries))
	for i, e := range entries {
		vizTypes := make([]ExploreVizType, len(e.VizTypes))
		for j, vt := range e.VizTypes {
			vizTypes[j] = ExploreVizType{
				VizType:     vt.VizType,
				RenderID:    vt.RenderID,
				GraphID:     vt.GraphID,
				ArtifactSVG: vt.ArtifactSVG,
				ArtifactPNG: vt.ArtifactPNG,
				ArtifactPDF: vt.ArtifactPDF,
			}
		}

		// Check if user has this in their library
		inLibrary := false
		if userID != "" {
			inLibrary, _ = docstore.IsInLibrary(r.Context(), userID, e.Source.Language, e.Source.Package)
		}

		apiEntries[i] = ExploreEntry{
			Source:     ToRenderSourceInfo(e.Source),
			NodeCount:  e.NodeCount,
			EdgeCount:  e.EdgeCount,
			CreatedAt:  e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			VizTypes:   vizTypes,
			Popularity: e.PopularityCount,
			InLibrary:  inLibrary,
		}
	}

	s.jsonResponse(w, http.StatusOK, ExploreResponse{
		Entries: apiEntries,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}
