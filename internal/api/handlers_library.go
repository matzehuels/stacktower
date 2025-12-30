package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

// =============================================================================
// Library Endpoint
// =============================================================================

// LibraryResponse is the response for GET /api/v1/library.
type LibraryResponse struct {
	Packages []*LibraryItem `json:"packages"` // Public packages in user's library
	Repos    []*RepoItem    `json:"repos"`    // Private repo renders
	Total    int64          `json:"total"`
	Limit    int            `json:"limit"`
	Offset   int            `json:"offset"`
}

// LibraryItem represents a package in the user's library.
type LibraryItem struct {
	Language  string           `json:"language"`
	Package   string           `json:"package"`
	SavedAt   time.Time        `json:"saved_at"`
	VizTypes  []ExploreVizType `json:"viz_types,omitempty"`
	NodeCount int              `json:"node_count,omitempty"`
	EdgeCount int              `json:"edge_count,omitempty"`
}

// RepoItem represents a private repo render.
type RepoItem struct {
	ID        string           `json:"id"`
	Source    RenderSourceInfo `json:"source"`
	NodeCount int              `json:"node_count"`
	EdgeCount int              `json:"edge_count"`
	GraphURL  string           `json:"graph_url"`
	Renders   []VizTypeRender  `json:"renders"`
	CreatedAt time.Time        `json:"created_at"`
}

// handleLibrary handles GET /api/v1/library.
// Returns the user's saved packages plus private repo renders.
func (s *Server) handleLibrary(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()
	userID := getUserID(r)
	docstore := ctx.Backend.DocumentStore()

	p := parsePagination(r, DefaultPageSize, MaxPageSize)

	// Get user's library entries
	entries, total, err := docstore.ListLibrary(r.Context(), userID, p.Limit, p.Offset)
	if err != nil {
		ctx.Logger.Error("failed to list library", "error", err, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "Failed to retrieve library")
		return
	}

	// Convert entries to LibraryItems with viz type info
	packages := make([]*LibraryItem, 0, len(entries))
	for _, entry := range entries {
		item := &LibraryItem{
			Language: entry.Language,
			Package:  entry.Package,
			SavedAt:  entry.SavedAt,
		}

		// Fetch canonical renders to get viz types
		vizTypes := []ExploreVizType{}
		for _, vt := range []string{"tower", "nodelink"} {
			render, _ := docstore.GetCanonicalRender(r.Context(), entry.Language, entry.Package, vt)
			if render != nil {
				artifactSVG := ""
				if render.Artifacts.SVG != "" {
					artifactSVG = "/api/v1/artifacts/" + render.Artifacts.SVG
				}
				artifactPNG := ""
				if render.Artifacts.PNG != "" {
					artifactPNG = "/api/v1/artifacts/" + render.Artifacts.PNG
				}
				artifactPDF := ""
				if render.Artifacts.PDF != "" {
					artifactPDF = "/api/v1/artifacts/" + render.Artifacts.PDF
				}
				vizTypes = append(vizTypes, ExploreVizType{
					VizType:     vt,
					RenderID:    render.ID,
					GraphID:     render.GraphID,
					ArtifactSVG: artifactSVG,
					ArtifactPNG: artifactPNG,
					ArtifactPDF: artifactPDF,
				})
				if item.NodeCount == 0 {
					item.NodeCount = render.NodeCount
					item.EdgeCount = render.EdgeCount
				}
			}
		}
		item.VizTypes = vizTypes
		packages = append(packages, item)
	}

	// Get private repo renders
	privateRenders, _, err := docstore.ListPrivateRenders(r.Context(), userID, p.Limit, 0)
	if err != nil {
		ctx.Logger.Error("failed to list private renders", "error", err, "user_id", userID, "request_id", getRequestID(r))
		privateRenders = []*storage.Render{}
	}

	repos := make([]*RepoItem, 0, len(privateRenders))
	for _, render := range privateRenders {
		repos = append(repos, &RepoItem{
			ID:        render.ID,
			Source:    ToRenderSourceInfo(render.Source),
			NodeCount: render.NodeCount,
			EdgeCount: render.EdgeCount,
			GraphURL:  buildGraphURL(render.GraphID),
			Renders: []VizTypeRender{{
				VizType:   render.LayoutOptions.VizType,
				Artifacts: buildArtifactURLs(render.Artifacts),
			}},
			CreatedAt: render.CreatedAt,
		})
	}

	s.jsonResponse(w, http.StatusOK, LibraryResponse{
		Packages: packages,
		Repos:    repos,
		Total:    total,
		Limit:    p.Limit,
		Offset:   p.Offset,
	})
}

// handleSaveToLibrary handles PUT /api/v1/library/{language}/{package}.
// Saves a package to the user's library.
func (s *Server) handleSaveToLibrary(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()
	userID := getUserID(r)
	language := chi.URLParam(r, "language")
	pkg := chi.URLParam(r, "package")

	if language == "" || pkg == "" {
		s.errorResponse(w, http.StatusBadRequest, "Language and package are required")
		return
	}

	docstore := ctx.Backend.DocumentStore()

	// Check if package exists (has canonical renders)
	render, err := docstore.GetCanonicalRender(r.Context(), language, pkg, "tower")
	if err != nil {
		ctx.Logger.Error("failed to check canonical render", "error", err, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "Failed to save to library")
		return
	}
	if render == nil {
		// Try nodelink
		render, _ = docstore.GetCanonicalRender(r.Context(), language, pkg, "nodelink")
	}
	if render == nil {
		s.errorResponse(w, http.StatusNotFound, "Package not found")
		return
	}

	// Save to library
	if err := docstore.SaveToLibrary(r.Context(), userID, language, pkg); err != nil {
		ctx.Logger.Error("failed to save to library", "error", err, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "Failed to save to library")
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]bool{"saved": true})
}

// handleRemoveFromLibrary handles DELETE /api/v1/library/{language}/{package}.
// Removes a package from the user's library.
func (s *Server) handleRemoveFromLibrary(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()
	userID := getUserID(r)
	language := chi.URLParam(r, "language")
	pkg := chi.URLParam(r, "package")

	if language == "" || pkg == "" {
		s.errorResponse(w, http.StatusBadRequest, "Language and package are required")
		return
	}

	docstore := ctx.Backend.DocumentStore()

	if err := docstore.RemoveFromLibrary(r.Context(), userID, language, pkg); err != nil {
		ctx.Logger.Error("failed to remove from library", "error", err, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "Failed to remove from library")
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]bool{"removed": true})
}
