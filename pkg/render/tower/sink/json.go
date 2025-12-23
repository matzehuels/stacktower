package sink

import (
	"encoding/json"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/feature"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
)

// JSONOption configures JSON rendering via [RenderJSON].
type JSONOption func(*jsonRenderer)

type jsonRenderer struct {
	graph     *dag.DAG
	merged    bool
	randomize bool
	seed      uint64
	style     string
	nebraska  []feature.NebraskaRanking
}

// WithJSONGraph attaches the DAG for metadata enrichment (URLs, brittle flags,
// auxiliary/synthetic flags). Without this, blocks will have minimal metadata.
func WithJSONGraph(g *dag.DAG) JSONOption { return func(r *jsonRenderer) { r.graph = g } }

// WithJSONMerged marks that the layout uses merged subdividers. This ensures the
// JSON correctly represents subdivider relationships.
func WithJSONMerged() JSONOption { return func(r *jsonRenderer) { r.merged = true } }

// WithJSONRandomize records the randomization seed in the JSON output, enabling
// reproducible re-rendering with the same visual jitter.
func WithJSONRandomize(seed uint64) JSONOption {
	return func(r *jsonRenderer) { r.randomize = true; r.seed = seed }
}

// WithJSONStyle records the style name (e.g., "simple", "handdrawn") in the JSON output
// for documentation or round-trip rendering.
func WithJSONStyle(s string) JSONOption { return func(r *jsonRenderer) { r.style = s } }

// WithJSONNebraska includes Nebraska ranking data in the JSON output. Rankings should
// come from [feature.RankNebraska].
func WithJSONNebraska(rankings []feature.NebraskaRanking) JSONOption {
	return func(r *jsonRenderer) { r.nebraska = rankings }
}

type jsonOutput struct {
	Width     float64          `json:"width"`
	Height    float64          `json:"height"`
	MarginX   float64          `json:"margin_x"`
	MarginY   float64          `json:"margin_y"`
	Style     string           `json:"style,omitempty"`
	Seed      uint64           `json:"seed,omitempty"`
	Randomize bool             `json:"randomize,omitempty"`
	Merged    bool             `json:"merged,omitempty"`
	Rows      map[int][]string `json:"rows,omitempty"`
	Blocks    []jsonBlock      `json:"blocks"`
	Edges     []jsonEdge       `json:"edges,omitempty"`
	Nebraska  []jsonNebraska   `json:"nebraska,omitempty"`
}

type jsonBlock struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	X         float64   `json:"x"`
	Y         float64   `json:"y"`
	Width     float64   `json:"width"`
	Height    float64   `json:"height"`
	URL       string    `json:"url,omitempty"`
	Brittle   bool      `json:"brittle,omitempty"`
	Auxiliary bool      `json:"auxiliary,omitempty"`
	Synthetic bool      `json:"synthetic,omitempty"`
	Meta      *jsonMeta `json:"meta,omitempty"`
}

type jsonMeta struct {
	Description string `json:"description,omitempty"`
	Stars       int    `json:"stars,omitempty"`
	LastCommit  string `json:"last_commit,omitempty"`
	LastRelease string `json:"last_release,omitempty"`
	Archived    bool   `json:"archived,omitempty"`
}

type jsonEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type jsonNebraska struct {
	Maintainer string           `json:"maintainer"`
	Score      float64          `json:"score"`
	Packages   []jsonNebPackage `json:"packages"`
}

type jsonNebPackage struct {
	Package string `json:"package"`
	Role    string `json:"role"` // "owner", "lead", or "maintainer"
	URL     string `json:"url,omitempty"`
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
func RenderJSON(l layout.Layout, opts ...JSONOption) ([]byte, error) {
	r := jsonRenderer{}
	for _, opt := range opts {
		opt(&r)
	}

	out := jsonOutput{
		Width:     l.FrameWidth,
		Height:    l.FrameHeight,
		MarginX:   l.MarginX,
		MarginY:   l.MarginY,
		Style:     r.style,
		Seed:      r.seed,
		Randomize: r.randomize,
		Merged:    r.merged,
		Rows:      l.RowOrders,
		Blocks:    buildJSONBlocks(l, r.graph),
	}

	if r.graph != nil {
		out.Edges = buildJSONEdges(l, r.graph, r.merged)
	}

	if len(r.nebraska) > 0 {
		out.Nebraska = buildJSONNebraska(r.nebraska)
	}

	return json.MarshalIndent(out, "", "  ")
}

func buildJSONBlocks(l layout.Layout, g *dag.DAG) []jsonBlock {
	blocks := make([]jsonBlock, 0, len(l.Blocks))
	for id, b := range l.Blocks {
		jb := jsonBlock{
			ID:     id,
			Label:  b.NodeID,
			X:      b.Left,
			Y:      b.Bottom,
			Width:  b.Width(),
			Height: b.Height(),
		}
		if g != nil {
			if n, ok := g.Node(id); ok {
				jb.Auxiliary = n.IsAuxiliary()
				jb.Synthetic = n.IsSynthetic()
				if n.Meta != nil {
					jb.URL, _ = n.Meta["repo_url"].(string)
					jb.Brittle = feature.IsBrittle(n)
					jb.Meta = extractJSONMeta(n)
				}
			}
		}
		blocks = append(blocks, jb)
	}
	return blocks
}

func extractJSONMeta(n *dag.Node) *jsonMeta {
	if n.Meta == nil {
		return nil
	}
	m := &jsonMeta{
		Stars: feature.AsInt(n.Meta["repo_stars"]),
	}
	m.LastCommit, _ = n.Meta["repo_last_commit"].(string)
	m.LastRelease, _ = n.Meta["repo_last_release"].(string)
	m.Archived, _ = n.Meta["repo_archived"].(bool)

	if desc, ok := n.Meta["description"].(string); ok && desc != "" {
		m.Description = desc
	} else if summary, ok := n.Meta["summary"].(string); ok && summary != "" {
		m.Description = summary
	}

	if m.Description == "" && m.Stars == 0 && m.LastCommit == "" && m.LastRelease == "" && !m.Archived {
		return nil
	}
	return m
}

func buildJSONEdges(l layout.Layout, g *dag.DAG, merged bool) []jsonEdge {
	if merged {
		return buildJSONMergedEdges(l, g)
	}
	edges := make([]jsonEdge, 0)
	for _, e := range g.Edges() {
		if _, ok := l.Blocks[e.From]; !ok {
			continue
		}
		if _, ok := l.Blocks[e.To]; !ok {
			continue
		}
		edges = append(edges, jsonEdge{From: e.From, To: e.To})
	}
	return edges
}

func buildJSONMergedEdges(l layout.Layout, g *dag.DAG) []jsonEdge {
	masterOf := func(id string) string {
		if n, ok := g.Node(id); ok && n.MasterID != "" {
			return n.MasterID
		}
		return id
	}

	type edgeKey struct{ from, to string }
	seen := make(map[edgeKey]struct{})
	var edges []jsonEdge

	for _, e := range g.Edges() {
		fromMaster, toMaster := masterOf(e.From), masterOf(e.To)
		if fromMaster == toMaster {
			continue
		}
		key := edgeKey{fromMaster, toMaster}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		edges = append(edges, jsonEdge{From: fromMaster, To: toMaster})
	}
	return edges
}

func buildJSONNebraska(rankings []feature.NebraskaRanking) []jsonNebraska {
	result := make([]jsonNebraska, len(rankings))
	for i, r := range rankings {
		pkgs := make([]jsonNebPackage, len(r.Packages))
		for j, p := range r.Packages {
			pkgs[j] = jsonNebPackage{Package: p.Package, Role: string(p.Role), URL: p.URL}
		}
		result[i] = jsonNebraska{
			Maintainer: r.Maintainer,
			Score:      r.Score,
			Packages:   pkgs,
		}
	}
	return result
}
