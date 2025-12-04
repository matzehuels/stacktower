package handdrawn

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/matzehuels/stacktower/pkg/render/tower/styles"
)

const (
	popupWidth      = 300.0
	popupLineHeight = 26.0
	charsPerLine    = 55
	popupPadding    = 20.0
	popupTextX      = 10.0
	popupTextStartY = 22.0
	popupTextSize   = 14.0
	popupStarSize   = 22.0
	popupStarShift  = 14.0
	dateLineSpacing = 0.9
	warnSymbolShift = 8.0
	textWidthRatio  = 0.45
	textHeightRatio = 1.0
)

type HandDrawn struct{ seed uint64 }

func New(seed uint64) *HandDrawn { return &HandDrawn{seed: seed} }

func (h *HandDrawn) RenderDefs(buf *bytes.Buffer) {
	buf.WriteString(`  <defs>
    <style>
      @import url('https://fonts.googleapis.com/css2?family=Patrick+Hand&amp;display=swap');
    </style>
    <pattern id="brittleTexture" patternUnits="userSpaceOnUse" width="200" height="200">
      <image href="`)
	buf.WriteString(getBrittleTextureDataURI())
	buf.WriteString(`" x="0" y="0" width="200" height="200" preserveAspectRatio="xMidYMid slice" opacity="0.6"/>
    </pattern>
  </defs>
`)
}

func (h *HandDrawn) RenderBlock(buf *bytes.Buffer, b styles.Block) {
	grey := greyForID(b.ID)
	rot := rotationFor(b.ID, b.W, b.H)
	path := wobbledRect(b.X, b.Y, b.W, b.H, h.seed, b.ID)

	styles.WrapURL(buf, b.URL, func() {
		class := "block"
		if b.Brittle {
			class = "block brittle"
		}
		fmt.Fprintf(buf, `<path id="block-%s" class="%s" d="%s" fill="%s" stroke="#333" stroke-width="2" stroke-linejoin="round" transform="rotate(%.3f %.2f %.2f)"/>`,
			styles.EscapeXML(b.ID), class, path, grey, rot, b.CX, b.CY)
	})
	buf.WriteByte('\n')

	if b.Brittle {
		fmt.Fprintf(buf, `  <path class="block-texture" d="%s" fill="url(#brittleTexture)" style="pointer-events: none;" transform="rotate(%.3f %.2f %.2f)"/>`+"\n",
			path, rot, b.CX, b.CY)
	}
}

func (h *HandDrawn) RenderEdge(buf *bytes.Buffer, e styles.Edge) {
	path := curvedEdge(e.X1, e.Y1, e.X2, e.Y2)
	fmt.Fprintf(buf, `  <path class="edge" d="%s" fill="none" stroke="#333" stroke-width="2.5" stroke-dasharray="8,5" stroke-linecap="round"/>`+"\n", path)
}

func (h *HandDrawn) RenderText(buf *bytes.Buffer, b styles.Block) {
	size := styles.FontSize(b)
	rotate := styles.ShouldRotate(b, size)
	if rotate {
		size = styles.FontSizeRotated(b)
	}
	grey := greyForID(b.ID)

	textW, textH := float64(len(b.ID))*size*textWidthRatio, size*textHeightRatio
	if rotate {
		textW, textH = textH, textW
	}

	fmt.Fprintf(buf, `  <g class="block-text" data-block="%s">`+"\n", styles.EscapeXML(b.ID))
	styles.WrapURL(buf, b.URL, func() {
		fmt.Fprintf(buf, `    <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`+"\n",
			b.CX-textW/2, b.CY-textH/2, textW, textH, grey)

		if rotate {
			fmt.Fprintf(buf, `    <text x="%.2f" y="%.2f" text-anchor="middle" dominant-baseline="middle" font-family="'Patrick Hand', cursive" font-size="%.1f" fill="#333" transform="rotate(-90 %.2f %.2f)">%s</text>`+"\n",
				b.CX, b.CY, size, b.CX, b.CY, styles.EscapeXML(b.ID))
		} else {
			fmt.Fprintf(buf, `    <text x="%.2f" y="%.2f" text-anchor="middle" dominant-baseline="middle" font-family="'Patrick Hand', cursive" font-size="%.1f" fill="#333">%s</text>`+"\n",
				b.CX, b.CY, size, styles.EscapeXML(b.ID))
		}
	})
	buf.WriteString("  </g>\n")
}

func (h *HandDrawn) RenderPopup(buf *bytes.Buffer, b styles.Block) {
	p := b.Popup
	if p == nil {
		return
	}

	descLines := wrapText(p.Description, charsPerLine)
	numDescLines := max(1, len(descLines))

	hasStats := p.Stars > 0 || p.LastCommit != "" || p.LastRelease != ""
	hasWarning := p.Archived || p.Brittle

	statsRows := 0
	if hasStats {
		statsRows = 1
		if p.LastCommit != "" && p.LastRelease != "" && p.LastRelease != "0001-01-01" {
			statsRows = 2
		}
	}

	height := float64(numDescLines+statsRows)*popupLineHeight + popupPadding
	path := wobbledRect(0, 0, popupWidth, height, h.seed, b.ID+"_popup")

	fmt.Fprintf(buf, `  <g class="popup" data-for="%s" visibility="hidden">`+"\n", styles.EscapeXML(b.ID))
	fmt.Fprintf(buf, `    <path d="%s" fill="white" stroke="#333" stroke-width="1.5" stroke-linejoin="round"/>`+"\n", path)

	textY := popupTextStartY
	for _, line := range descLines {
		fmt.Fprintf(buf, `    <text x="%.1f" y="%.1f" font-family="'Patrick Hand', cursive" font-size="%.0f" fill="#444">%s</text>`+"\n",
			popupTextX, textY, popupTextSize, styles.EscapeXML(line))
		textY += popupLineHeight
	}

	if hasStats {
		statsStartY := textY
		rightY := statsStartY
		leftCenterX := popupWidth / 4
		rightCenterX := popupWidth * 3 / 4

		warnPrefix := ""
		dateX := rightCenterX
		if hasWarning {
			warnPrefix = "⚠ "
			dateX -= warnSymbolShift
		}

		if p.LastCommit != "" {
			fmt.Fprintf(buf, `    <text x="%.1f" y="%.1f" text-anchor="middle" font-family="'Patrick Hand', cursive" font-size="%.0f" fill="#444">%slast commit: %s</text>`+"\n",
				dateX, rightY, popupTextSize, warnPrefix, p.LastCommit)
			rightY += popupLineHeight * dateLineSpacing
		}
		if p.LastRelease != "" && p.LastRelease != "0001-01-01" {
			fmt.Fprintf(buf, `    <text x="%.1f" y="%.1f" text-anchor="middle" font-family="'Patrick Hand', cursive" font-size="%.0f" fill="#444">%slast release: %s</text>`+"\n",
				dateX, rightY, popupTextSize, warnPrefix, p.LastRelease)
		}

		if p.Stars > 0 {
			starsCenterY := statsStartY + (popupLineHeight*float64(statsRows))/2 - popupStarShift
			fmt.Fprintf(buf, `    <text x="%.1f" y="%.1f" text-anchor="middle" dominant-baseline="middle" font-family="'Patrick Hand', cursive" font-size="%.0f" fill="#222" font-weight="bold">★ %s</text>`+"\n",
				leftCenterX, starsCenterY, popupStarSize, formatNumber(p.Stars))
		}
	}

	buf.WriteString("  </g>\n")
}

func formatNumber(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func wrapText(s string, maxChars int) []string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= maxChars {
		return []string{s}
	}

	var lines []string
	var line strings.Builder

	for _, word := range strings.Fields(s) {
		if line.Len() == 0 {
			line.WriteString(word)
		} else if line.Len()+1+len(word) <= maxChars {
			line.WriteByte(' ')
			line.WriteString(word)
		} else {
			lines = append(lines, line.String())
			line.Reset()
			line.WriteString(word)
		}
	}
	if line.Len() > 0 {
		lines = append(lines, line.String())
	}
	return lines
}
