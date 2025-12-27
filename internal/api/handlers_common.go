package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/matzehuels/stacktower/pkg/infra/session"
)

// =============================================================================
// Error Message Helpers - Consistent error formatting across handlers
// =============================================================================

// errInvalidJSON formats a JSON parsing error message.
func errInvalidJSON(err error) string {
	return fmt.Sprintf("invalid JSON: %v", err)
}

// errInvalidRequest formats a request validation error message.
func errInvalidRequest(err error) string {
	return fmt.Sprintf("invalid request: %v", err)
}

// errFieldRequired formats a missing required field error message.
func errFieldRequired(field string) string {
	return fmt.Sprintf("%s is required", field)
}

// errResourceNotFound formats a not found error message.
func errResourceNotFound(resource string) string {
	return fmt.Sprintf("%s not found", resource)
}

func (s *Server) decodeJSON(w http.ResponseWriter, r *http.Request, v interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxRequestSize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Headers already sent, log the error for debugging
		s.logger.Error("failed to encode JSON response", "error", err, "status", status)
	}
}

func (s *Server) errorResponse(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, APIError{
		Code:    httpStatusToErrorCode(status),
		Message: message,
	})
}

// httpStatusToErrorCode maps HTTP status codes to error codes.
func httpStatusToErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return ErrCodeBadRequest
	case http.StatusUnauthorized:
		return ErrCodeUnauthorized
	case http.StatusForbidden:
		return ErrCodeForbidden
	case http.StatusNotFound:
		return ErrCodeNotFound
	case http.StatusTooManyRequests:
		return ErrCodeRateLimited
	case http.StatusServiceUnavailable:
		return ErrCodeServiceUnavail
	default:
		return ErrCodeInternal
	}
}

func (s *Server) isAllowedOrigin(origin string) bool {
	for _, allowed := range s.allowOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// sessionCookie creates a consistent session cookie with all security attributes.
func (s *Server) sessionCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     "session",
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
}

// apiVersionHeader adds API version headers to all responses.
func (s *Server) apiVersionHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-API-Version", apiVersion)
		next.ServeHTTP(w, r)
	})
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

// generateJobID generates a unique job ID using UUID v4.
func generateJobID() string {
	return uuid.New().String()
}
