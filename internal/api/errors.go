package api

import (
	"fmt"
	"net/http"
)

// =============================================================================
// Error Codes - Machine-readable error identifiers for client-side handling
// =============================================================================

const (
	ErrCodeValidation     = "VALIDATION_ERROR"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
	ErrCodeForbidden      = "FORBIDDEN"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeRateLimited    = "RATE_LIMITED"
	ErrCodeInternal       = "INTERNAL_ERROR"
	ErrCodeServiceUnavail = "SERVICE_UNAVAILABLE"
	ErrCodeBadRequest     = "BAD_REQUEST"
)

// APIError represents a structured error response.
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// =============================================================================
// Error Message Helpers - Consistent error formatting across handlers
// =============================================================================
//
// Error message conventions:
//   - Use lowercase, no trailing period
//   - Be specific but concise
//   - Don't expose internal details to users
//   - Use these helpers for consistency

// msgInvalidJSON formats a JSON parsing error message.
func msgInvalidJSON(err error) string {
	return fmt.Sprintf("invalid JSON: %v", err)
}

// msgInvalidRequest formats a request validation error message.
func msgInvalidRequest(err error) string {
	return fmt.Sprintf("invalid request: %v", err)
}

// msgFieldRequired formats a missing required field error message.
func msgFieldRequired(field string) string {
	return fmt.Sprintf("%s is required", field)
}

// msgResourceNotFound formats a not found error message.
func msgResourceNotFound(resource string) string {
	return fmt.Sprintf("%s not found", resource)
}

// Standard error messages - use these instead of inline strings for consistency.
// ALL handler error messages should be defined here for maintainability.
const (
	// Auth errors
	errMsgAuthRequired   = "authentication required"
	errMsgAccessDenied   = "access denied"
	errMsgSessionInvalid = "valid session required"

	// Security errors
	errMsgCrossOrigin   = "cross-origin request blocked"
	errMsgOriginInvalid = "origin validation failed"

	// OAuth errors
	errMsgOAuthNotConfig = "GitHub OAuth not configured"
	errMsgInvalidState   = "invalid or expired state"
	errMsgAuthFailed     = "authentication failed"

	// Operation errors
	errMsgInternalError   = "internal server error"
	errMsgRateLimited     = "rate limit exceeded"
	errMsgVisualizeFailed = "visualization failed"

	// Job errors
	errMsgJobCreateFailed  = "failed to create job"
	errMsgJobEnqueueFailed = "failed to start job"
	errMsgJobDeleteFailed  = "failed to delete job"
	errMsgJobListFailed    = "failed to list jobs"

	// Render errors
	errMsgRenderGetFailed    = "failed to retrieve render"
	errMsgRenderDeleteFailed = "failed to delete render"

	// History errors
	errMsgHistoryFailed = "failed to retrieve render history"

	// Repository errors
	errMsgRepoFetchFailed      = "failed to fetch repositories"
	errMsgManifestFetchFailed  = "failed to fetch manifest"
	errMsgManifestDetectFailed = "failed to detect manifests"
	errMsgUnknownManifest      = "unknown manifest type"
	errMsgAnalysisFailed       = "failed to start analysis"
)

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
