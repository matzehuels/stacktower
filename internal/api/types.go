// Package api provides HTTP REST API types for Stacktower.
//
// This file defines request/response types for all API endpoints.
// Types are organized by endpoint group and follow these patterns:
//
//   - Request types embed pipeline.Options where appropriate (DRY)
//   - Response types are API-specific and don't expose storage internals
//   - Transformation functions convert storage types to API types
package api

import (
	"time"

	"github.com/matzehuels/stacktower/pkg/infra/storage"
	"github.com/matzehuels/stacktower/pkg/pipeline"
)

// =============================================================================
// Render Endpoint Types
// =============================================================================

// RenderRequest is the request body for POST /api/v1/render.
// Embeds pipeline.Options for all rendering configuration.
type RenderRequest struct {
	pipeline.Options
}

// RenderResponse is the response for POST /api/v1/render.
type RenderResponse struct {
	Status   string        `json:"status"` // "completed" or "pending"
	RenderID string        `json:"render_id,omitempty"`
	JobID    string        `json:"job_id,omitempty"`
	Cached   bool          `json:"cached,omitempty"`
	Result   *RenderResult `json:"result,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// RenderResult contains the output of a completed render.
type RenderResult struct {
	Artifacts map[string]string `json:"artifacts"` // format -> artifact URL
	NodeCount int               `json:"node_count"`
	EdgeCount int               `json:"edge_count"`
	VizType   string            `json:"viz_type"`
	Source    RenderSourceInfo  `json:"source"`
	Layout    interface{}       `json:"layout,omitempty"` // Full layout data (blocks, edges, nebraska, etc.)
}

// NebraskaRankingAPI represents a maintainer's influence score (API type).
type NebraskaRankingAPI struct {
	Maintainer string               `json:"maintainer"`
	Score      float64              `json:"score"`
	Packages   []NebraskaPackageAPI `json:"packages"`
}

// NebraskaPackageAPI represents a package maintained by someone (API type).
type NebraskaPackageAPI struct {
	Package string `json:"package"`
	Role    string `json:"role"` // "owner", "lead", or "maintainer"
	URL     string `json:"url,omitempty"`
	Depth   int    `json:"depth,omitempty"`
}

// RenderSourceInfo describes the source of a render.
// This is the API representation of storage.RenderSource.
type RenderSourceInfo struct {
	Type     string `json:"type"` // "package" or "manifest"
	Language string `json:"language"`
	Package  string `json:"package,omitempty"`
	Repo     string `json:"repo,omitempty"`
}

// ToRenderSourceInfo converts a storage.RenderSource to API type.
// This is the canonical transformation from storage to API layer.
func ToRenderSourceInfo(s storage.RenderSource) RenderSourceInfo {
	return RenderSourceInfo{
		Type:     s.Type,
		Language: s.Language,
		Package:  s.Package,
		Repo:     s.Repo,
	}
}

// ToNebraskaRankingsAPI converts storage.NebraskaRanking slice to API types.
func ToNebraskaRankingsAPI(rankings []storage.NebraskaRanking) []NebraskaRankingAPI {
	if len(rankings) == 0 {
		return nil
	}
	result := make([]NebraskaRankingAPI, len(rankings))
	for i, r := range rankings {
		pkgs := make([]NebraskaPackageAPI, len(r.Packages))
		for j, p := range r.Packages {
			pkgs[j] = NebraskaPackageAPI{
				Package: p.Package,
				Role:    p.Role,
				URL:     p.URL,
				Depth:   p.Depth,
			}
		}
		result[i] = NebraskaRankingAPI{
			Maintainer: r.Maintainer,
			Score:      r.Score,
			Packages:   pkgs,
		}
	}
	return result
}

// =============================================================================
// Job Endpoint Types
// =============================================================================

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

// =============================================================================
// History Endpoint Types
// =============================================================================

// HistoryResponse is the response for GET /api/v1/history.
type HistoryResponse struct {
	Renders []*RenderHistoryItem `json:"renders"`
	Total   int64                `json:"total"`
	Limit   int                  `json:"limit"`
	Offset  int                  `json:"offset"`
}

// VizTypeRender represents a single visualization type's artifacts.
type VizTypeRender struct {
	VizType   string            `json:"viz_type"`
	Artifacts map[string]string `json:"artifacts"` // svg, png, pdf URLs
}

// RenderHistoryItem is a single item in the user's render history.
// Each item represents a package with all available viz type renders.
type RenderHistoryItem struct {
	ID        string           `json:"id"`
	Source    RenderSourceInfo `json:"source"`
	NodeCount int              `json:"node_count"`
	EdgeCount int              `json:"edge_count"`
	GraphURL  string           `json:"graph_url"` // Shared JSON graph URL
	Renders   []VizTypeRender  `json:"renders"`   // Available viz types with their artifacts
	Layout    interface{}      `json:"layout,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
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
// Note: This endpoint runs synchronously (no job queuing) so it uses
// a simplified request structure instead of embedding pipeline.Options.
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

// =============================================================================
// Repository Endpoint Types
// =============================================================================

// RepoAnalyzeRequest is the request body for POST /api/v1/repos/{owner}/{repo}/analyze.
type RepoAnalyzeRequest struct {
	ManifestPath string   `json:"manifest_path"`
	Formats      []string `json:"formats,omitempty"`
}
