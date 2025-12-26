package api

import (
	"time"

	"github.com/matzehuels/stacktower/internal/jobs"
	"github.com/matzehuels/stacktower/pkg/infra/cache"
)

// RenderRequest is the request body for POST /api/v1/render.
// It embeds RenderPayload and adds API-specific fields.
type RenderRequest struct {
	jobs.RenderPayload

	// Repo is the GitHub repository ("owner/repo") for manifest sources.
	Repo string `json:"repo,omitempty"`

	// Refresh forces bypassing all caches.
	Refresh bool `json:"refresh,omitempty"`
}

// IsPublic returns true if this is a public package (vs private manifest).
func (r *RenderRequest) IsPublic() bool {
	return r.Manifest == ""
}

// GraphOptions extracts graph-related options for cache keys.
func (r *RenderRequest) GraphOptions() cache.GraphOptions {
	return cache.GraphOptions{
		MaxDepth:  r.MaxDepth,
		MaxNodes:  r.MaxNodes,
		Normalize: r.Normalize,
	}
}

// LayoutOptions extracts layout-related options for cache keys.
func (r *RenderRequest) LayoutOptions() cache.LayoutOptions {
	return cache.LayoutOptions{
		VizType:   r.VizType,
		Width:     r.Width,
		Height:    r.Height,
		Ordering:  r.Ordering,
		Merge:     r.Merge,
		Randomize: r.Randomize,
		Seed:      r.Seed,
	}
}

// RenderOptions extracts render-related options.
func (r *RenderRequest) RenderOptions() cache.RenderOptions {
	return cache.RenderOptions{
		Formats:   r.Formats,
		Style:     r.Style,
		ShowEdges: r.ShowEdges,
		Nebraska:  r.Nebraska,
		Popups:    r.Popups,
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
