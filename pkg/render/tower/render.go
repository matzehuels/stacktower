package tower

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/matzehuels/stacktower/pkg/dag"
	"github.com/matzehuels/stacktower/pkg/render/tower/styles"
)

type RenderOption func(*renderer)

type renderer struct {
	graph     *dag.DAG
	style     styles.Style
	showEdges bool
	merged    bool
	nebraska  []NebraskaRanking
	popups    bool
}

func WithGraph(g *dag.DAG) RenderOption     { return func(r *renderer) { r.graph = g } }
func WithEdges() RenderOption               { return func(r *renderer) { r.showEdges = true } }
func WithStyle(s styles.Style) RenderOption { return func(r *renderer) { r.style = s } }
func WithMerged() RenderOption              { return func(r *renderer) { r.merged = true } }
func WithNebraska(rankings []NebraskaRanking) RenderOption {
	return func(r *renderer) { r.nebraska = rankings }
}
func WithPopups() RenderOption { return func(r *renderer) { r.popups = true } }

const (
	nebraskaPanelHeightLandscape = 260.0
	nebraskaPanelHeightPortrait  = 480.0
)

func calcNebraskaPanelHeight(frameWidth, frameHeight float64) float64 {
	if frameHeight > frameWidth {
		return nebraskaPanelHeightPortrait
	}
	return nebraskaPanelHeightLandscape
}

func RenderSVG(layout Layout, opts ...RenderOption) []byte {
	r := renderer{style: styles.Simple{}}
	for _, opt := range opts {
		opt(&r)
	}

	blocks := buildBlocks(layout, r.graph, r.popups)
	slices.SortFunc(blocks, func(a, b styles.Block) int {
		return cmp.Compare(a.ID, b.ID)
	})

	var edges []styles.Edge
	if r.showEdges {
		edges = buildEdges(layout, r.graph, r.merged)
	}

	totalHeight := layout.FrameHeight
	if len(r.nebraska) > 0 {
		totalHeight += calcNebraskaPanelHeight(layout.FrameWidth, layout.FrameHeight)
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.1f %.1f" width="%.0f" height="%.0f">`+"\n",
		layout.FrameWidth, totalHeight, layout.FrameWidth, totalHeight)

	r.style.RenderDefs(&buf)

	for _, b := range blocks {
		r.style.RenderBlock(&buf, b)
	}
	for _, e := range edges {
		r.style.RenderEdge(&buf, e)
	}
	for _, b := range blocks {
		if r.graph != nil {
			if n, ok := r.graph.Node(b.ID); ok && n.IsAuxiliary() {
				continue
			}
		}
		r.style.RenderText(&buf, b)
	}

	if len(r.nebraska) > 0 {
		renderNebraskaPanel(&buf, layout.FrameWidth, layout.FrameHeight, r.nebraska)
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

const (
	nebraskaPanelPadding = 24.0
	nebraskaTitleY       = 40.0
	nebraskaUnderlineY   = 16.0
	nebraskaEntryStartY  = 80.0
	nebraskaEntryHeight  = 120.0
)

func renderNebraskaPanel(buf *bytes.Buffer, frameWidth, frameHeight float64, rankings []NebraskaRanking) {
	panelY := frameHeight + nebraskaPanelPadding
	centerX := frameWidth / 2

	fmt.Fprintf(buf, `  <text x="%.1f" y="%.1f" text-anchor="middle" font-family="'Patrick Hand', cursive" font-size="30" fill="#333" font-weight="bold">Nebraska Guy Ranking</text>`+"\n",
		centerX, panelY+nebraskaTitleY)
	fmt.Fprintf(buf, `  <path d="M %.1f %.1f q 60 4 120 -1 t 135 3" fill="none" stroke="#333" stroke-width="2.5" stroke-linecap="round"/>`+"\n",
		centerX-128, panelY+nebraskaTitleY+nebraskaUnderlineY)

	numEntries := min(len(rankings), 5)
	padding := 30.0
	isPortrait := frameHeight > frameWidth

	if isPortrait {
		cols := 2
		availableWidth := frameWidth - 2*padding
		entryWidth := availableWidth / float64(cols)

		for i := 0; i < numEntries; i++ {
			row, col := i/cols, i%cols
			var entryX float64
			if row == 2 && numEntries == 5 {
				entryX = (frameWidth - entryWidth) / 2
			} else {
				entryX = padding + float64(col)*entryWidth
			}
			entryY := panelY + nebraskaEntryStartY + float64(row)*nebraskaEntryHeight
			renderNebraskaEntry(buf, rankings[i], i, entryX, entryY, entryWidth)
		}
	} else {
		availableWidth := frameWidth - 2*padding
		entryWidth := availableWidth / float64(numEntries)
		entryY := panelY + nebraskaEntryStartY

		for i := 0; i < numEntries; i++ {
			entryX := padding + float64(i)*entryWidth
			renderNebraskaEntry(buf, rankings[i], i, entryX, entryY, entryWidth)
		}
	}
}

func renderNebraskaEntry(buf *bytes.Buffer, r NebraskaRanking, idx int, x, y, width float64) {
	pkgIDs := make([]string, len(r.Packages))
	for j, p := range r.Packages {
		pkgIDs[j] = p.Package
	}

	fmt.Fprintf(buf, `  <foreignObject x="%.1f" y="%.1f" width="%.1f" height="%.1f">`+"\n",
		x, y, width, nebraskaEntryHeight)
	fmt.Fprintf(buf, `    <div xmlns="http://www.w3.org/1999/xhtml" class="nebraska-entry">`+"\n")
	fmt.Fprintf(buf, `      <a href="https://github.com/%s" target="_blank" class="maintainer-name" data-packages="%s">#%d @%s</a>`+"\n",
		r.Maintainer, styles.EscapeXML(strings.Join(pkgIDs, ",")), idx+1, styles.EscapeXML(r.Maintainer))
	buf.WriteString(`      <div class="packages">` + "\n")
	for j, p := range r.Packages {
		if j >= 3 {
			break
		}
		fmt.Fprintf(buf, `        <span>%s</span>`+"\n", styles.EscapeXML(p.Package))
	}
	buf.WriteString("      </div>\n    </div>\n  </foreignObject>\n")
}

const nebraskaCSS = `
    .block { transition: stroke-width 0.2s ease; }
    .block.highlight { stroke-width: 4; }
    .block-text { transition: transform 0.2s ease; transform-origin: center; transform-box: fill-box; }
    .block-text.highlight { transform: scale(1.08); font-weight: bold; }
    a, .package-entry { cursor: pointer; }
    .nebraska-entry {
      text-align: center;
      font-family: 'Patrick Hand', cursive;
      overflow: hidden;
      height: 100%;
    }
    .nebraska-entry .maintainer-name {
      display: block;
      font-size: 24px;
      color: #333;
      text-decoration: none;
      word-wrap: break-word;
      overflow-wrap: break-word;
      margin-bottom: 8px;
    }
    .nebraska-entry .maintainer-name:hover { text-decoration: underline; }
    .nebraska-entry .packages {
      font-size: 16px;
      color: #888;
      line-height: 1.4;
    }
    .nebraska-entry .packages span {
      display: block;
      word-wrap: break-word;
      overflow-wrap: break-word;
    }`

const nebraskaJS = `
    function highlight(pkgs) {
      document.querySelectorAll('.block').forEach(b => b.classList.toggle('highlight', pkgs.includes(b.id.replace('block-', ''))));
      document.querySelectorAll('.block-text').forEach(t => t.classList.toggle('highlight', pkgs.includes(t.dataset.block)));
    }
    function clearHighlight() {
      document.querySelectorAll('.block, .block-text').forEach(el => el.classList.remove('highlight'));
    }
    document.querySelectorAll('.maintainer-name').forEach(el => {
      el.addEventListener('mouseenter', () => highlight(el.dataset.packages.split(',')));
      el.addEventListener('mouseleave', clearHighlight);
    });
    document.querySelectorAll('.package-entry').forEach(el => {
      el.addEventListener('mouseenter', () => highlight([el.dataset.package]));
      el.addEventListener('mouseleave', clearHighlight);
    });
    document.querySelectorAll('.block').forEach(el => {
      el.addEventListener('mouseenter', () => highlight([el.id.replace('block-', '')]));
      el.addEventListener('mouseleave', clearHighlight);
    });`

func renderNebraskaScript(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "  <style>%s\n  </style>\n", nebraskaCSS)
	fmt.Fprintf(buf, "  <script type=\"text/javascript\"><![CDATA[%s\n  ]]></script>\n", nebraskaJS)
}

const popupCSS = `
    .popup { pointer-events: none; transition: opacity 0.15s ease, transform 0.1s ease; }
    .popup[visibility="hidden"] { opacity: 0; }
    .popup[visibility="visible"] { opacity: 1; }`

const popupJS = `
    const svg = document.querySelector('svg');
    const vb = svg.viewBox.baseVal;
    document.querySelectorAll('.block-text').forEach(el => {
      const id = el.dataset.block;
      const popup = document.querySelector('.popup[data-for="' + id + '"]');
      if (!popup) return;
      el.style.cursor = 'pointer';
      el.addEventListener('mouseenter', () => {
        const textBox = el.getBBox();
        const popupBox = popup.getBBox();
        let x = textBox.x + textBox.width/2 - popupBox.width/2;
        let y = textBox.y + textBox.height + 12;
        if (y + popupBox.height > vb.y + vb.height - 10) y = textBox.y - popupBox.height - 8;
        if (y < vb.y + 10) y = vb.y + 10;
        x = Math.max(vb.x + 10, Math.min(x, vb.x + vb.width - popupBox.width - 10));
        popup.setAttribute('transform', 'translate(' + x.toFixed(1) + ',' + y.toFixed(1) + ')');
        popup.setAttribute('visibility', 'visible');
      });
      el.addEventListener('mouseleave', () => popup.setAttribute('visibility', 'hidden'));
    });`

func renderPopupScript(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "  <style>%s\n  </style>\n", popupCSS)
	fmt.Fprintf(buf, "  <script type=\"text/javascript\"><![CDATA[%s\n  ]]></script>\n", popupJS)
}

func buildBlocks(l Layout, g *dag.DAG, withPopups bool) []styles.Block {
	blocks := make([]styles.Block, 0, len(l.Blocks))
	for id, b := range l.Blocks {
		blk := styles.Block{
			ID: id,
			X:  b.Left, Y: b.Bottom,
			W: b.Width(), H: b.Height(),
			CX: b.CenterX(), CY: b.CenterY(),
		}
		if g != nil {
			if n, ok := g.Node(id); ok && n.Meta != nil {
				blk.URL, _ = n.Meta["repo_url"].(string)
				blk.Brittle = IsBrittle(n)
				if withPopups {
					blk.Popup = extractPopupData(n)
				}
			}
		}
		blocks = append(blocks, blk)
	}
	return blocks
}

func extractPopupData(n *dag.Node) *styles.PopupData {
	if n.Meta == nil {
		return nil
	}
	p := &styles.PopupData{
		Stars:       asInt(n.Meta["repo_stars"]),
		Maintainers: countMaintainers(n.Meta["repo_maintainers"]),
		Brittle:     IsBrittle(n),
	}
	p.LastCommit, _ = n.Meta["repo_last_commit"].(string)
	p.LastRelease, _ = n.Meta["repo_last_release"].(string)
	p.Archived, _ = n.Meta["repo_archived"].(bool)

	if desc, ok := n.Meta["description"].(string); ok && desc != "" {
		p.Description = desc
	} else if summary, ok := n.Meta["summary"].(string); ok && summary != "" {
		p.Description = summary
	}
	return p
}

func asInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	default:
		return 0
	}
}

func buildEdges(l Layout, g *dag.DAG, merged bool) []styles.Edge {
	if g == nil {
		return nil
	}
	if merged {
		return buildMergedEdges(l, g)
	}
	return buildSimpleEdges(l, g)
}

func buildSimpleEdges(l Layout, g *dag.DAG) []styles.Edge {
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

func buildMergedEdges(l Layout, g *dag.DAG) []styles.Edge {
	masterOf := func(id string) string {
		if n, ok := g.Node(id); ok && n.MasterID != "" {
			return n.MasterID
		}
		return id
	}

	blockFor := func(id string) (Block, bool) {
		if b, ok := l.Blocks[id]; ok {
			return b, true
		}
		if master := masterOf(id); master != id {
			if b, ok := l.Blocks[master]; ok {
				return b, true
			}
		}
		return Block{}, false
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
