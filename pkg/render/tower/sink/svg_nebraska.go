package sink

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/matzehuels/stacktower/pkg/render/tower/feature"
	"github.com/matzehuels/stacktower/pkg/render/tower/styles"
)

const (
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
    .maintainer-link text { cursor: pointer; }
    .maintainer-link:hover text { text-decoration: underline; }`

const nebraskaJS = `
    document.querySelectorAll('.maintainer-link').forEach(el => {
      el.addEventListener('mouseenter', () => highlight(el.dataset.packages.split(',')));
      el.addEventListener('mouseleave', clearHighlight);
    });
    document.querySelectorAll('.package-entry').forEach(el => {
      el.addEventListener('mouseenter', () => highlight([el.dataset.package]));
      el.addEventListener('mouseleave', clearHighlight);
    });`

func calcNebraskaPanelHeight(w, h float64) float64 {
	if h > w {
		return nebraskaPanelPortrait
	}
	return nebraskaPanelLandscape
}

func renderNebraskaPanel(buf *bytes.Buffer, frameWidth, frameHeight float64, rankings []feature.NebraskaRanking) {
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

func renderNebraskaEntry(buf *bytes.Buffer, r feature.NebraskaRanking, idx int, x, y, width float64) {
	allPkgIDs := make([]string, len(r.Packages))
	for j, p := range r.Packages {
		allPkgIDs[j] = p.Package
	}

	displayed := min(len(r.Packages), maxDisplayedPackages)
	centerX := x + width/2

	// Maintainer name with link
	fmt.Fprintf(buf, `  <a href="https://github.com/%s" target="_blank" class="maintainer-link" data-packages="%s">`+"\n",
		r.Maintainer, styles.EscapeXML(strings.Join(allPkgIDs, ",")))
	fmt.Fprintf(buf, `    <text x="%.1f" y="%.1f" text-anchor="middle" font-family="%s" font-size="16" fill="#0366d6" font-weight="bold">#%d @%s</text>`+"\n",
		centerX, y+20, fontFamily, idx+1, styles.EscapeXML(r.Maintainer))
	buf.WriteString("  </a>\n")

	// Package names
	lineY := y + 45
	for j := 0; j < displayed; j++ {
		pkg := r.Packages[j].Package
		displayPkg := pkg
		// Truncate long package names for display
		if len(displayPkg) > 25 {
			displayPkg = displayPkg[:22] + "..."
		}
		fmt.Fprintf(buf, `  <text class="package-entry" data-package="%s" x="%.1f" y="%.1f" text-anchor="middle" font-family="%s" font-size="12" fill="#666" style="cursor:pointer">%s</text>`+"\n",
			styles.EscapeXML(pkg), centerX, lineY, fontFamily, styles.EscapeXML(displayPkg))
		lineY += 18
	}
	if extra := len(r.Packages) - displayed; extra > 0 {
		fmt.Fprintf(buf, `  <text x="%.1f" y="%.1f" text-anchor="middle" font-family="%s" font-size="12" fill="#aaa">+%d more</text>`+"\n",
			centerX, lineY, fontFamily, extra)
	}
}

func renderNebraskaScript(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "  <style>%s\n  </style>\n", nebraskaCSS)
	fmt.Fprintf(buf, "  <script type=\"text/javascript\"><![CDATA[%s\n  ]]></script>\n", nebraskaJS)
}
