package api

import (
	"encoding/json"
	"net/http"
)

// =============================================================================
// JSON Request/Response Helpers
// =============================================================================

// decodeJSON decodes the request body into v with safety limits.
// It enforces maxRequestSize and rejects unknown fields.
func (s *Server) decodeJSON(w http.ResponseWriter, r *http.Request, v interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxRequestSize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

// jsonResponse writes a JSON response with the given status code.
func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Headers already sent, log the error for debugging
		s.logger.Error("failed to encode JSON response", "error", err, "status", status)
	}
}

// errorResponse writes a structured error response.
func (s *Server) errorResponse(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, APIError{
		Code:    httpStatusToErrorCode(status),
		Message: message,
	})
}
