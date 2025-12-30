package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/matzehuels/stacktower/pkg/infra/session"
)

// =============================================================================
// Session and Cookie Helpers
// =============================================================================

// sessionCookie creates a consistent session cookie with all security attributes.
// Uses SameSite=Strict for maximum CSRF protection - the cookie is only sent
// for same-site requests. This works because:
//   - OAuth callback redirects work (they're navigations, not cross-site requests)
//   - API calls from our frontend work (same origin)
//   - Cross-site form submissions are blocked (good!)
func (s *Server) sessionCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     "session",
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	}
}

// getSession retrieves the session from the request cookie.
func (s *Server) getSession(r *http.Request) *session.Session {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}

	sess, err := s.sessions.Get(r.Context(), cookie.Value)
	if err != nil || sess == nil {
		return nil
	}

	return sess
}

// =============================================================================
// CORS Helpers
// =============================================================================

// isAllowedOrigin checks if the origin is in the allowed list.
func (s *Server) isAllowedOrigin(origin string) bool {
	for _, allowed := range s.allowOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// =============================================================================
// Middleware Helpers
// =============================================================================

// apiVersionHeader adds API version headers to all responses.
func (s *Server) apiVersionHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-API-Version", apiVersion)
		next.ServeHTTP(w, r)
	})
}

// =============================================================================
// ID Generation
// =============================================================================

// generateJobID generates a unique job ID using UUID v4.
func generateJobID() string {
	return uuid.New().String()
}
