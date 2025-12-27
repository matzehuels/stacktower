package api

import (
	"time"

	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// APIError represents a structured error response.
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Common error codes for client-side handling.
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

// RenderRequest is the request body for POST /api/v1/render.
// It embeds pipeline.Options for all rendering configuration.
// Access validation via req.Options.ValidateAndSetDefaults().
type RenderRequest struct {
	pipeline.Options
}

// GraphOptions extracts graph-related options for cache keys.
func (r *RenderRequest) GraphOptions() storage.GraphOptions {
	return storage.GraphOptions{
		MaxDepth:  r.Options.MaxDepth,
		MaxNodes:  r.Options.MaxNodes,
		Normalize: r.Options.Normalize,
	}
}

// LayoutOptions extracts layout-related options for cache keys.
func (r *RenderRequest) LayoutOptions() storage.LayoutOptions {
	return storage.LayoutOptions{
		VizType:   r.Options.VizType,
		Width:     r.Options.Width,
		Height:    r.Options.Height,
		Ordering:  r.Options.Ordering,
		Merge:     r.Options.Merge,
		Randomize: r.Options.Randomize,
		Seed:      r.Options.Seed,
	}
}

// RenderResponse is the response for POST /api/v1/render.
type RenderResponse struct {
	Status       string        `json:"status"` // "completed", "pending"
	RenderID     string        `json:"render_id,omitempty"`
	JobID        string        `json:"job_id,omitempty"`
	Cached       bool          `json:"cached,omitempty"`
	Stale        bool          `json:"stale,omitempty"`      // Data may be outdated
	Refreshing   bool          `json:"refreshing,omitempty"` // Background refresh in progress
	RefreshJobID string        `json:"refresh_job_id,omitempty"`
	Result       *RenderResult `json:"result,omitempty"`
	Error        string        `json:"error,omitempty"`
}

// RenderResult contains the output of a completed render.
type RenderResult struct {
	Artifacts map[string]string `json:"artifacts"` // format -> artifact URL
	NodeCount int               `json:"node_count"`
	EdgeCount int               `json:"edge_count"`
	Source    RenderSourceInfo  `json:"source"`
}

// RenderSourceInfo describes what was rendered.
type RenderSourceInfo struct {
	Type     string `json:"type"` // "package" or "manifest"
	Language string `json:"language"`
	Package  string `json:"package,omitempty"`
	Repo     string `json:"repo,omitempty"`
}

// JobResponse is the response after submitting an async job.
type JobResponse struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// JobStatusResponse is the response for GET /api/v1/jobs/:id.
type JobStatusResponse struct {
	JobID       string                 `json:"job_id"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Duration    *time.Duration         `json:"duration,omitempty"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       *string                `json:"error,omitempty"`
}

// HistoryResponse is the response for GET /api/v1/history.
type HistoryResponse struct {
	Renders []*RenderHistoryItem `json:"renders"`
	Total   int64                `json:"total"`
	Limit   int                  `json:"limit"`
	Offset  int                  `json:"offset"`
}

// RenderHistoryItem is a single item in the user's render history.
type RenderHistoryItem struct {
	ID        string            `json:"id"`
	Source    RenderSourceInfo  `json:"source"`
	VizType   string            `json:"viz_type"`
	NodeCount int               `json:"node_count"`
	EdgeCount int               `json:"edge_count"`
	Artifacts map[string]string `json:"artifacts"`
	CreatedAt time.Time         `json:"created_at"`
}

// =============================================================================
// Parse Endpoint Types
// =============================================================================

// ParseRequest is the request body for POST /api/v1/parse.
// Embeds pipeline.Options for all parsing configuration.
type ParseRequest struct {
	pipeline.Options
}

// ParseResponse is the response for POST /api/v1/parse.
type ParseResponse struct {
	Status    string `json:"status"`           // "pending" or "completed"
	JobID     string `json:"job_id,omitempty"` // For async polling
	Graph     []byte `json:"graph,omitempty"`
	NodeCount int    `json:"node_count,omitempty"`
	EdgeCount int    `json:"edge_count,omitempty"`
	Cached    bool   `json:"cached,omitempty"`
	Error     string `json:"error,omitempty"`
}

// =============================================================================
// Layout Endpoint Types
// =============================================================================

// LayoutRequest is the request body for POST /api/v1/layout.
// Embeds pipeline.Options for layout configuration, plus graph data fields.
type LayoutRequest struct {
	pipeline.Options
	Graph   []byte `json:"graph"`    // Inline graph JSON
	GraphID string `json:"graph_id"` // Or reference to stored graph
}

// LayoutResponse is the response for POST /api/v1/layout.
type LayoutResponse struct {
	Status string `json:"status"`           // "pending" or "completed"
	JobID  string `json:"job_id,omitempty"` // For async polling
	Layout []byte `json:"layout,omitempty"`
	Cached bool   `json:"cached,omitempty"`
	Error  string `json:"error,omitempty"`
}

// =============================================================================
// Visualize Endpoint Types
// =============================================================================

// VisualizeRequest is the request body for POST /api/v1/visualize.
type VisualizeRequest struct {
	Layout    []byte   `json:"layout"`
	Graph     []byte   `json:"graph,omitempty"` // Optional, for popups/nebraska
	VizType   string   `json:"viz_type,omitempty"`
	Formats   []string `json:"formats,omitempty"`
	Style     string   `json:"style,omitempty"`
	ShowEdges bool     `json:"show_edges,omitempty"`
	Popups    bool     `json:"popups,omitempty"`
}

// VisualizeResponse is the response for POST /api/v1/visualize.
type VisualizeResponse struct {
	Status    string            `json:"status"`
	Artifacts map[string]string `json:"artifacts,omitempty"` // format -> base64-encoded data
	Cached    bool              `json:"cached,omitempty"`
	Error     string            `json:"error,omitempty"`
}
