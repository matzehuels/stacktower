package storage

// =============================================================================
// Render History Utilities
// =============================================================================
//
// User render history is stored directly via DocumentStore methods.
// Use Keys.RenderDocumentID() to generate deterministic IDs for renders.
//
// Example lookup:
//
//	renderID := storage.Keys.RenderDocumentID(userID, language, pkg, vizType)
//	render, err := docstore.GetRenderDocScoped(ctx, renderID, userID)
//
// Example check if render has all formats:
//
//	if render != nil && storage.RenderHasFormats(render, []string{"svg", "png"}) {
//	    // All formats available
//	}

// RenderHasFormats checks if a render has all requested artifact formats.
func RenderHasFormats(render *Render, formats []string) bool {
	if render == nil {
		return false
	}
	for _, format := range formats {
		switch format {
		case "svg":
			if render.Artifacts.SVG == "" {
				return false
			}
		case "png":
			if render.Artifacts.PNG == "" {
				return false
			}
		case "pdf":
			if render.Artifacts.PDF == "" {
				return false
			}
		}
	}
	return true
}

// BuildArtifactURLs creates API URLs for a render's artifacts.
// Note: Nebraska rankings are returned directly in the render response (parsed from Render.Layout),
// so there's no need for a separate layout artifact URL.
func BuildArtifactURLs(artifacts RenderArtifacts, graphID string) map[string]string {
	urls := make(map[string]string)
	if artifacts.SVG != "" {
		urls["svg"] = "/api/v1/artifacts/" + artifacts.SVG
	}
	if artifacts.PNG != "" {
		urls["png"] = "/api/v1/artifacts/" + artifacts.PNG
	}
	if artifacts.PDF != "" {
		urls["pdf"] = "/api/v1/artifacts/" + artifacts.PDF
	}
	if graphID != "" {
		urls["json"] = "/api/v1/graphs/" + graphID
	}
	return urls
}
