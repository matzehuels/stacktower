package api

import (
	"fmt"
	"net/http"
)

// handleHistory handles GET /api/v1/history.
// Auth handled by middleware.
func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	// Parse pagination using helper.
	p := parsePagination(r, DefaultPageSize, MaxPageSize)

	renders, total, err := s.backend.DocumentStore().ListRenderDocs(r.Context(), userID, p.Limit, p.Offset)
	if err != nil {
		s.logger.Error("failed to list renders", "error", err, "user_id", userID, "request_id", getRequestID(r))
		s.errorResponse(w, http.StatusInternalServerError, "failed to retrieve render history")
		return
	}

	items := make([]*RenderHistoryItem, len(renders))
	for i, render := range renders {
		artifacts := make(map[string]string)
		if render.Artifacts.SVG != "" {
			artifacts["svg"] = fmt.Sprintf("/api/v1/artifacts/%s", render.Artifacts.SVG)
		}
		if render.Artifacts.PNG != "" {
			artifacts["png"] = fmt.Sprintf("/api/v1/artifacts/%s", render.Artifacts.PNG)
		}
		if render.Artifacts.PDF != "" {
			artifacts["pdf"] = fmt.Sprintf("/api/v1/artifacts/%s", render.Artifacts.PDF)
		}

		items[i] = &RenderHistoryItem{
			ID: render.ID,
			Source: RenderSourceInfo{
				Type:     render.Source.Type,
				Language: render.Source.Language,
				Package:  render.Source.Package,
				Repo:     render.Source.Repo,
			},
			VizType:   render.LayoutOptions.VizType,
			NodeCount: render.NodeCount,
			EdgeCount: render.EdgeCount,
			Artifacts: artifacts,
			CreatedAt: render.CreatedAt,
		}
	}

	s.jsonResponse(w, http.StatusOK, HistoryResponse{
		Renders: items,
		Total:   total,
		Limit:   p.Limit,
		Offset:  p.Offset,
	})
}
