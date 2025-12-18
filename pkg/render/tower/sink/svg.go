package sink

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/feature"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
	"github.com/matzehuels/stacktower/pkg/render/tower/styles"
)

const blockInteractionCSS = `
    .block { transition: stroke-width 0.2s ease; }
    .block.highlight { stroke-width: 4; }
    .block-text { transition: transform 0.2s ease; transform-origin: center; transform-box: fill-box; }
    .block-text.highlight { transform: scale(1.08); font-weight: bold; }
    a { cursor: pointer; }`

const blockInteractionJS = `
    function highlight(pkgs) {
      document.querySelectorAll('.block').forEach(b => b.classList.toggle('highlight', pkgs.includes(b.id.replace('block-', ''))));
      document.querySelectorAll('.block-text').forEach(t => t.classList.toggle('highlight', pkgs.includes(t.dataset.block)));
    }
    function clearHighlight() {
      document.querySelectorAll('.block, .block-text').forEach(el => el.classList.remove('highlight'));
    }
    document.querySelectorAll('.block').forEach(el => {
      el.addEventListener('mouseenter', () => highlight([el.id.replace('block-', '')]));
      el.addEventListener('mouseleave', clearHighlight);
    });`

type SVGOption func(*svgRenderer)

type svgRenderer struct {
	graph     *dag.DAG
	style     styles.Style
	showEdges bool
	merged    bool
	nebraska  []feature.NebraskaRanking
	popups    bool
}

func WithGraph(g *dag.DAG) SVGOption     { return func(r *svgRenderer) { r.graph = g } }
func WithEdges() SVGOption               { return func(r *svgRenderer) { r.showEdges = true } }
func WithStyle(s styles.Style) SVGOption { return func(r *svgRenderer) { r.style = s } }
func WithMerged() SVGOption              { return func(r *svgRenderer) { r.merged = true } }
func WithNebraska(rankings []feature.NebraskaRanking) SVGOption {
	return func(r *svgRenderer) { r.nebraska = rankings }
}
func WithPopups() SVGOption { return func(r *svgRenderer) { r.popups = true } }

func RenderSVG(l layout.Layout, opts ...SVGOption) []byte {
	r := newSVGRenderer(opts...)

	blocks := buildBlocks(l, r.graph, r.popups)
	slices.SortFunc(blocks, func(a, b styles.Block) int {
		return cmp.Compare(a.ID, b.ID)
	})

	var edges []styles.Edge
	if r.showEdges {
		edges = buildEdges(l, r.graph, r.merged)
	}

	totalHeight := calculateHeight(l, r.nebraska)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.1f %.1f" width="%.0f" height="%.0f">`+"\n",
		l.FrameWidth, totalHeight, l.FrameWidth, totalHeight)

	r.style.RenderDefs(&buf)
	renderContent(&buf, &r, blocks, edges)
	renderBlockInteraction(&buf)

	if len(r.nebraska) > 0 {
		renderNebraskaPanel(&buf, l.FrameWidth, l.FrameHeight, r.nebraska)
		renderNebraskaScript(&buf)
	}

	if r.popups {
		for _, b := range blocks {
			r.style.RenderPopup(&buf, b)
		}
		renderPopupScript(&buf)
	}

	buf.WriteString("</svg>\n")
	return buf.Bytes()
}

func newSVGRenderer(opts ...SVGOption) svgRenderer {
	r := svgRenderer{style: styles.Simple{}}
	for _, opt := range opts {
		opt(&r)
	}
	return r
}

func calculateHeight(l layout.Layout, nebraska []feature.NebraskaRanking) float64 {
	h := l.FrameHeight
	if len(nebraska) > 0 {
		h += calcNebraskaPanelHeight(l.FrameWidth, l.FrameHeight)
	}
	return h
}

func renderContent(buf *bytes.Buffer, r *svgRenderer, blocks []styles.Block, edges []styles.Edge) {
	for _, b := range blocks {
		r.style.RenderBlock(buf, b)
	}
	for _, e := range edges {
		r.style.RenderEdge(buf, e)
	}
	for _, b := range blocks {
		if shouldSkipText(r.graph, b.ID) {
			continue
		}
		r.style.RenderText(buf, b)
	}
}

func shouldSkipText(g *dag.DAG, id string) bool {
	if g == nil {
		return false
	}
	n, ok := g.Node(id)
	return ok && n.IsAuxiliary()
}

func renderBlockInteraction(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "  <style>%s\n  </style>\n", blockInteractionCSS)
	fmt.Fprintf(buf, "  <script type=\"text/javascript\"><![CDATA[%s\n  ]]></script>\n", blockInteractionJS)
}

func buildBlocks(l layout.Layout, g *dag.DAG, withPopups bool) []styles.Block {
	blocks := make([]styles.Block, 0, len(l.Blocks))
	for id, b := range l.Blocks {
		blk := styles.Block{
			ID:    id,
			Label: b.NodeID,
			X:     b.Left, Y: b.Bottom,
			W: b.Width(), H: b.Height(),
			CX: b.CenterX(), CY: b.CenterY(),
		}
		if g != nil {
			if n, ok := g.Node(id); ok && n.Meta != nil {
				blk.URL, _ = n.Meta["repo_url"].(string)
				blk.Brittle = feature.IsBrittle(n)
				if withPopups {
					blk.Popup = extractPopupData(n)
				}
			}
		}
		blocks = append(blocks, blk)
	}
	return blocks
}

func buildEdges(l layout.Layout, g *dag.DAG, merged bool) []styles.Edge {
	if g == nil {
		return nil
	}
	if merged {
		return buildMergedEdges(l, g)
	}
	return buildSimpleEdges(l, g)
}

func buildSimpleEdges(l layout.Layout, g *dag.DAG) []styles.Edge {
	edges := make([]styles.Edge, 0, len(g.Edges()))
	for _, e := range g.Edges() {
		src, okS := l.Blocks[e.From]
		dst, okD := l.Blocks[e.To]
		if !okS || !okD {
			continue
		}
		edges = append(edges, styles.Edge{
			FromID: e.From, ToID: e.To,
			X1: src.CenterX(), Y1: src.CenterY(),
			X2: dst.CenterX(), Y2: dst.CenterY(),
		})
	}
	return edges
}

func buildMergedEdges(l layout.Layout, g *dag.DAG) []styles.Edge {
	masterOf := func(id string) string {
		if n, ok := g.Node(id); ok && n.MasterID != "" {
			return n.MasterID
		}
		return id
	}

	blockFor := func(id string) (layout.Block, bool) {
		if b, ok := l.Blocks[id]; ok {
			return b, true
		}
		if master := masterOf(id); master != id {
			if b, ok := l.Blocks[master]; ok {
				return b, true
			}
		}
		return layout.Block{}, false
	}

	type edgeKey struct{ from, to string }
	seen := make(map[edgeKey]struct{})
	var edges []styles.Edge

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

		src, okS := blockFor(e.From)
		dst, okD := blockFor(e.To)
		if !okS || !okD {
			continue
		}

		edges = append(edges, styles.Edge{
			FromID: fromMaster, ToID: toMaster,
			X1: src.CenterX(), Y1: src.CenterY(),
			X2: dst.CenterX(), Y2: dst.CenterY(),
		})
	}
	return edges
}
