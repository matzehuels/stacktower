package api

import (
	"net/http"
	"strconv"
)

// Pagination defaults and limits.
// These values balance API responsiveness with usability:
// - DefaultPageSize (20): Small enough for quick responses, large enough to be useful
// - MaxPageSize (100): Prevents excessive memory usage and response times
// - MaxJobsPageSize (200): Jobs are lightweight, allowing larger pages for admin views
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
	MaxJobsPageSize = 200
)

// Pagination represents parsed pagination parameters.
type Pagination struct {
	Limit  int
	Offset int
}

// parsePagination extracts limit and offset from query parameters.
// Uses defaults if not provided, and caps at maxLimit.
func parsePagination(r *http.Request, defaultLimit, maxLimit int) Pagination {
	p := Pagination{
		Limit:  defaultLimit,
		Offset: 0,
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			p.Limit = parsed
			if p.Limit > maxLimit {
				p.Limit = maxLimit
			}
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			p.Offset = parsed
		}
	}

	return p
}
