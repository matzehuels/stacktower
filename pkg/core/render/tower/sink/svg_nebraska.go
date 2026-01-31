package sink

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/matzehuels/stacktower/pkg/core/render/tower/feature"
	"github.com/matzehuels/stacktower/pkg/core/render/tower/styles"
	"github.com/matzehuels/stacktower/pkg/fonts"
)

const (
	nebraskaPanelLandscape = 280.0
	nebraskaPanelPortrait  = 540.0
	nebraskaPanelPadding   = 24.0
	nebraskaTitleY         = 40.0
	nebraskaUnderlineY     = 16.0
	nebraskaEntryStartY    = 80.0
	nebraskaEntryHeight    = 155.0
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

func renderNebraskaPanel(buf *bytes.Buffer, frameWidth, frameHeight float64, rankings []feature.NebraskaRanking) {
	numEntries := min(len(rankings), 6)
	padding := 30.0
	isLandscape := frameWidth > frameHeight

	if isLandscape {
		// Panel on the right side: 2 columns, 3 rows
		panelX := frameWidth + nebraskaPanelPadding
		panelWidth := nebraskaPanelLandscape - 2*nebraskaPanelPadding
		centerX := panelX + panelWidth/2

		// Title at top of panel
		titleY := watermarkMargin + nebraskaTitleY
		fmt.Fprintf(buf, `  <text x="%.1f" y="%.1f" text-anchor="middle" font-family="%s" font-size="24" fill="#333" font-weight="bold">Nebraska Guy</text>`+"\n",
			centerX, titleY, fonts.FallbackFontFamily)
		fmt.Fprintf(buf, `  <text x="%.1f" y="%.1f" text-anchor="middle" font-family="%s" font-size="24" fill="#333" font-weight="bold">Ranking</text>`+"\n",
			centerX, titleY+28, fonts.FallbackFontFamily)
		fmt.Fprintf(buf, `  <path d="M %.1f %.1f q 30 2 60 -1 t 65 2" fill="none" stroke="#333" stroke-width="2.5" stroke-linecap="round"/>`+"\n",
			centerX-63, titleY+28+nebraskaUnderlineY)

		// 2 columns, 3 rows grid with margins
		cols := 2
		colMargin := 12.0
		rowMargin := 5.0
		entryWidth := (panelWidth - colMargin) / float64(cols)
		entryHeight := 145.0 + rowMargin
		startY := titleY + 70

		for i := 0; i < numEntries; i++ {
			row := i / cols
			col := i % cols
			entryX := panelX + float64(col)*(entryWidth+colMargin)
			entryY := startY + float64(row)*entryHeight
			renderNebraskaEntry(buf, rankings[i], i, entryX, entryY, entryWidth)
		}
	} else {
		// Panel below tower (portrait mode): 3 columns, 2 rows
		panelY := frameHeight + watermarkMargin + nebraskaPanelPadding
		centerX := frameWidth / 2

		fmt.Fprintf(buf, `  <text x="%.1f" y="%.1f" text-anchor="middle" font-family="%s" font-size="30" fill="#333" font-weight="bold">Nebraska Guy Ranking</text>`+"\n",
			centerX, panelY+nebraskaTitleY, fonts.FallbackFontFamily)
		fmt.Fprintf(buf, `  <path d="M %.1f %.1f q 60 4 120 -1 t 135 3" fill="none" stroke="#333" stroke-width="2.5" stroke-linecap="round"/>`+"\n",
			centerX-128, panelY+nebraskaTitleY+nebraskaUnderlineY)

		// 3 columns, 2 rows grid with margins
		cols := 3
		colMargin := 16.0
		rowMargin := 8.0
		availableWidth := frameWidth - 2*padding
		entryWidth := (availableWidth - float64(cols-1)*colMargin) / float64(cols)

		for i := 0; i < numEntries; i++ {
			row := i / cols
			col := i % cols
			entryX := padding + float64(col)*(entryWidth+colMargin)
			entryY := panelY + nebraskaEntryStartY + float64(row)*(nebraskaEntryHeight+rowMargin)
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
	fmt.Fprintf(buf, `    <text x="%.1f" y="%.1f" text-anchor="middle" font-family="%s" font-size="18" fill="#333" font-weight="bold">#%d @%s</text>`+"\n",
		centerX, y+20, fonts.FallbackFontFamily, idx+1, styles.EscapeXML(r.Maintainer))
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
		fmt.Fprintf(buf, `  <text class="package-entry" data-package="%s" x="%.1f" y="%.1f" text-anchor="middle" font-family="%s" font-size="14" fill="#666" style="cursor:pointer">%s</text>`+"\n",
			styles.EscapeXML(pkg), centerX, lineY, fonts.FallbackFontFamily, styles.EscapeXML(displayPkg))
		lineY += 20
	}
	if extra := len(r.Packages) - displayed; extra > 0 {
		fmt.Fprintf(buf, `  <text x="%.1f" y="%.1f" text-anchor="middle" font-family="%s" font-size="14" fill="#aaa">+%d more</text>`+"\n",
			centerX, lineY, fonts.FallbackFontFamily, extra)
	}
}

func renderNebraskaScript(buf *bytes.Buffer) {
	fmt.Fprintf(buf, "  <style>%s\n  </style>\n", nebraskaCSS)
	fmt.Fprintf(buf, "  <script type=\"text/javascript\"><![CDATA[%s\n  ]]></script>\n", nebraskaJS)
}
