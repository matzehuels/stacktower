package sink

import (
	"github.com/matzehuels/stacktower/pkg/core/dag"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/feature"
	towerio "github.com/matzehuels/stacktower/pkg/core/render/tower/io"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/layout"
)

// JSONOption configures JSON rendering via [RenderJSON].
type JSONOption func(*jsonConfig)

type jsonConfig struct {
	opts []towerio.WriteOption
}

// WithJSONGraph attaches the DAG for metadata enrichment (URLs, brittle flags,
// auxiliary/synthetic flags). Without this, blocks will have minimal metadata.
func WithJSONGraph(g *dag.DAG) JSONOption {
	return func(c *jsonConfig) { c.opts = append(c.opts, towerio.WithGraph(g)) }
}

// WithJSONMerged marks that the layout uses merged subdividers. This ensures the
// JSON correctly represents subdivider relationships.
func WithJSONMerged() JSONOption {
	return func(c *jsonConfig) { c.opts = append(c.opts, towerio.WithMerged()) }
}

// WithJSONRandomize records the randomization seed in the JSON output, enabling
// reproducible re-rendering with the same visual jitter.
func WithJSONRandomize(seed uint64) JSONOption {
	return func(c *jsonConfig) { c.opts = append(c.opts, towerio.WithRandomize(seed)) }
}

// WithJSONStyle records the style name (e.g., "simple", "handdrawn") in the JSON output
// for documentation or round-trip rendering.
func WithJSONStyle(s string) JSONOption {
	return func(c *jsonConfig) { c.opts = append(c.opts, towerio.WithStyle(s)) }
}

// WithJSONNebraska includes Nebraska ranking data in the JSON output. Rankings should
// come from [feature.RankNebraska].
func WithJSONNebraska(rankings []feature.NebraskaRanking) JSONOption {
	return func(c *jsonConfig) { c.opts = append(c.opts, towerio.WithNebraska(rankings)) }
}

// RenderJSON exports the layout and associated metadata as a pretty-printed JSON document.
// This is the primary data interchange format for Stacktower, enabling:
//
//   - Integration with external visualization tools
//   - Caching computed layouts for fast re-rendering
//   - Round-trip rendering (re-import and render identically)
//
// The JSON includes:
//   - Block positions and dimensions
//   - Row orderings (for reconstructing the layout)
//   - Metadata (URLs, stars, dates, auxiliary/synthetic flags)
//   - Optional Nebraska rankings
//   - Render options (style, seed, merged flag) for reproducibility
//
// RenderJSON returns an error only if JSON marshaling fails (should not happen
// with well-formed layouts). It does not modify l or the DAG, and is safe to call
// concurrently.
//
// For more control over layout serialization, use [towerio.WriteLayout] directly.
func RenderJSON(l layout.Layout, opts ...JSONOption) ([]byte, error) {
	cfg := jsonConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	return towerio.WriteLayout(l, cfg.opts...)
}
