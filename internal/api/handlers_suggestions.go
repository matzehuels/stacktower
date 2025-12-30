package api

import (
	"net/http"
)

// PackageSuggestionsResponse is the response for GET /api/v1/packages/suggestions.
type PackageSuggestionsResponse struct {
	Suggestions []PackageSuggestion `json:"suggestions"`
}

// PackageSuggestion is a single package suggestion for autocomplete.
type PackageSuggestion struct {
	Package    string `json:"package"`
	Language   string `json:"language"`
	Popularity int    `json:"popularity"`
}

// handlePackageSuggestions handles GET /api/v1/packages/suggestions.
// This returns package suggestions for autocomplete based on the global render history.
// No auth required - this is public data for better UX.
func (s *Server) handlePackageSuggestions(w http.ResponseWriter, r *http.Request) {
	ctx := s.handlerContext()

	language := r.URL.Query().Get("language")
	query := r.URL.Query().Get("q")

	// Default limit of 10, max 20
	limit := 10
	if query == "" {
		// Without a query, return fewer suggestions (popular packages)
		limit = 5
	}

	suggestions, err := ctx.Backend.DocumentStore().ListPackageSuggestions(r.Context(), language, query, limit)
	if err != nil {
		ctx.Logger.Error("failed to list package suggestions", "error", err, "language", language, "query", query)
		s.jsonResponse(w, http.StatusOK, PackageSuggestionsResponse{Suggestions: []PackageSuggestion{}})
		return
	}

	// Convert storage type to API type
	apiSuggestions := make([]PackageSuggestion, len(suggestions))
	for i, s := range suggestions {
		apiSuggestions[i] = PackageSuggestion{
			Package:    s.Package,
			Language:   s.Language,
			Popularity: s.Popularity,
		}
	}

	s.jsonResponse(w, http.StatusOK, PackageSuggestionsResponse{Suggestions: apiSuggestions})
}
