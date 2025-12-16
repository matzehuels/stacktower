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

type Role string

const (
	RoleOwner      Role = "owner"
	RoleLead       Role = "lead"
	RoleMaintainer Role = "maintainer"
)

type PackageRole struct {
	Package string
	Role    Role
	URL     string
	Depth   int
}

type NebraskaRanking struct {
	Maintainer string
	Score      float64
	Packages   []PackageRole
}

const (
	ownerWeight      = 3.0
	leadWeight       = 1.5
	maintainerWeight = 1.0

	nebraskaPanelLandscape = 280.0
	nebraskaPanelPortrait  = 540.0
	nebraskaPanelPadding   = 24.0
	nebraskaTitleY         = 40.0
	nebraskaUnderlineY     = 16.0
	nebraskaEntryStartY    = 80.0
	nebraskaEntryHeight    = 140.0

	fontFamily = `'Patrick Hand', 'Comic Sans MS', 'Bradley Hand', 'Segoe Script', sans-serif`
)

const nebraskaCSS = `
    .nebraska-entry {
      text-align: center;
      font-family: 'Patrick Hand', 'Comic Sans MS', 'Bradley Hand', 'Segoe Script', sans-serif;
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
    document.querySelectorAll('.maintainer-name').forEach(el => {
      el.addEventListener('mouseenter', () => highlight(el.dataset.packages.split(',')));
      el.addEventListener('mouseleave', clearHighlight);
    });
    document.querySelectorAll('.package-entry').forEach(el => {
      el.addEventListener('mouseenter', () => highlight([el.dataset.package]));
      el.addEventListener('mouseleave', clearHighlight);
    });`

func RankNebraska(g *dag.DAG, topN int) []NebraskaRanking {
	scores := make(map[string]float64)
	packages := make(map[string][]PackageRole)
	bestRole := make(map[string]Role)
	minRow := findMinRow(g)

	for _, n := range g.Nodes() {
		if n.IsSynthetic() || g.InDegree(n.ID) == 0 {
			continue
		}

		roles := getMaintainerRoles(n)
		if len(roles) == 0 {
			continue
		}

		depth := n.Row - minRow
		share := float64(depth) / float64(len(roles))

		for maintainer, role := range roles {
			scores[maintainer] += share * roleWeight(role)

			if !hasPackage(packages[maintainer], n.ID) {
				url, _ := n.Meta["repo_url"].(string)
				packages[maintainer] = append(packages[maintainer], PackageRole{
					Package: n.ID,
					Role:    role,
					URL:     url,
					Depth:   depth,
				})
			}

			if roleRank(role) < roleRank(bestRole[maintainer]) {
				bestRole[maintainer] = role
			}
		}
	}

	rankings := make([]NebraskaRanking, 0, len(scores))
	for m, score := range scores {
		pkgs := packages[m]
		slices.SortFunc(pkgs, func(a, b PackageRole) int {
			// Sort by role first (owner > lead > maintainer)
			if c := cmp.Compare(roleRank(a.Role), roleRank(b.Role)); c != 0 {
				return c
			}
			// Then by depth descending (deeper = more foundational)
			if c := cmp.Compare(b.Depth, a.Depth); c != 0 {
				return c
			}
			// Then alphabetically for stability
			return cmp.Compare(a.Package, b.Package)
		})
		rankings = append(rankings, NebraskaRanking{
			Maintainer: m,
			Score:      score,
			Packages:   pkgs,
		})
	}

	slices.SortFunc(rankings, func(a, b NebraskaRanking) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		if c := cmp.Compare(roleRank(bestRole[a.Maintainer]), roleRank(bestRole[b.Maintainer])); c != 0 {
			return c
		}
		return cmp.Compare(a.Maintainer, b.Maintainer)
	})

	if len(rankings) > topN {
		return rankings[:topN]
	}
	return rankings
}

func roleRank(r Role) int {
	switch r {
	case RoleOwner:
		return 0
	case RoleLead:
		return 1
	case RoleMaintainer:
		return 2
	default:
		return 3
	}
}

func roleWeight(r Role) float64 {
	switch r {
	case RoleOwner:
		return ownerWeight
	case RoleLead:
		return leadWeight
	default:
		return maintainerWeight
	}
}

func findMinRow(g *dag.DAG) int {
	minRow := -1
	for _, n := range g.Nodes() {
		if !n.IsSynthetic() && (minRow < 0 || n.Row < minRow) {
			minRow = n.Row
		}
	}
	return max(0, minRow)
}

func getMaintainerRoles(n *dag.Node) map[string]Role {
	if n.Meta == nil {
		return nil
	}

	owner, _ := n.Meta["repo_owner"].(string)
	maintainers := getStringSlice(n.Meta["repo_maintainers"])

	if len(maintainers) == 0 && owner != "" {
		return map[string]Role{owner: RoleOwner}
	}

	roles := make(map[string]Role, len(maintainers))
	leadAssigned := false

	for _, m := range maintainers {
		switch {
		case m == owner:
			roles[m] = RoleOwner
		case !leadAssigned:
			roles[m] = RoleLead
			leadAssigned = true
		default:
			roles[m] = RoleMaintainer
		}
	}
	return roles
}

func getStringSlice(v any) []string {
	switch v := v.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func hasPackage(pkgs []PackageRole, id string) bool {
	return slices.ContainsFunc(pkgs, func(p PackageRole) bool { return p.Package == id })
}

func CalcNebraskaPanelHeight(w, h float64) float64 {
	if h > w {
		return nebraskaPanelPortrait
	}
	return nebraskaPanelLandscape
}

func RenderNebraskaPanel(buf *bytes.Buffer, frameWidth, frameHeight float64, rankings []NebraskaRanking) {
	panelY := frameHeight + nebraskaPanelPadding
	centerX := frameWidth / 2

	fmt.Fprintf(buf, `  <text x="%.1f" y="%.1f" text-anchor="middle" font-family="%s" font-size="30" fill="#333" font-weight="bold">Nebraska Guy Ranking</text>`+"\n",
		centerX, panelY+nebraskaTitleY, fontFamily)
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

const maxDisplayedPackages = 3

func renderNebraskaEntry(buf *bytes.Buffer, r NebraskaRanking, idx int, x, y, width float64) {
	// All packages for hover highlighting
	allPkgIDs := make([]string, len(r.Packages))
	for j, p := range r.Packages {
		allPkgIDs[j] = p.Package
	}

	displayed := min(len(r.Packages), maxDisplayedPackages)

	fmt.Fprintf(buf, `  <foreignObject x="%.1f" y="%.1f" width="%.1f" height="%.1f">`+"\n",
		x, y, width, nebraskaEntryHeight)
	fmt.Fprintf(buf, `    <div xmlns="http://www.w3.org/1999/xhtml" class="nebraska-entry">`+"\n")
	fmt.Fprintf(buf, `      <a href="https://github.com/%s" target="_blank" class="maintainer-name" data-packages="%s">#%d @%s</a>`+"\n",
		r.Maintainer, styles.EscapeXML(strings.Join(allPkgIDs, ",")), idx+1, styles.EscapeXML(r.Maintainer))
	buf.WriteString(`      <div class="packages">` + "\n")
	for j := 0; j < displayed; j++ {
		fmt.Fprintf(buf, `        <span>%s</span>`+"\n", styles.EscapeXML(r.Packages[j].Package))
	}
	if extra := len(r.Packages) - displayed; extra > 0 {
		fmt.Fprintf(buf, `        <span style="color:#aaa">+%d more</span>`+"\n", extra)
	}
	buf.WriteString("      </div>\n    </div>\n  </foreignObject>\n")
}

func RenderNebraskaScript(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "  <style>%s\n  </style>\n", nebraskaCSS)
	fmt.Fprintf(buf, "  <script type=\"text/javascript\"><![CDATA[%s\n  ]]></script>\n", nebraskaJS)
}
