package pipeline

import (
	"encoding/json"
	"fmt"

	"github.com/matzehuels/stacktower/pkg/infra/storage"
)

// =============================================================================
// Unified Payload - used for all job types (parse, layout, render)
// =============================================================================

// JobPayload is the unified job payload that supports all pipeline stages.
// It embeds Options and adds worker-specific fields.
type JobPayload struct {
	Options

	// Cache keys (for linking results)
	GraphCacheKey string `json:"graph_cache_key,omitempty"`

	// For layout jobs: reference to existing graph
	GraphID   string `json:"graph_id,omitempty"`
	GraphData []byte `json:"graph_data,omitempty"`

	// Webhook callback
	Webhook string `json:"webhook,omitempty"`
}

// ToOptions returns the embedded Options.
func (p *JobPayload) ToOptions() Options {
	return p.Options
}

// FromOptions creates a JobPayload from Options.
// Useful for creating job payloads from API requests.
func FromOptions(opts Options) JobPayload {
	return JobPayload{Options: opts}
}

// ToMap converts the payload to a map for job queue serialization.
// Uses JSON marshaling for simplicity and correctness.
func (p *JobPayload) ToMap() (map[string]interface{}, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	return m, nil
}

// MarshalJSON returns the JSON encoding of the payload.
// This can be used directly when the queue accepts []byte.
func (p *JobPayload) MarshalJSON() ([]byte, error) {
	// Use an alias to avoid infinite recursion
	type PayloadAlias JobPayload
	return json.Marshal((*PayloadAlias)(p))
}

// IsPublic returns true if this is a public package (vs private manifest).
func (p *JobPayload) IsPublic() bool {
	return p.Options.Manifest == ""
}

// DetermineScope returns the appropriate scope based on payload content.
func (p *JobPayload) DetermineScope() storage.Scope {
	if p.Options.Scope != "" {
		return p.Options.Scope
	}
	if p.Options.Manifest != "" {
		return storage.ScopeUser
	}
	return storage.ScopeGlobal
}
