package styles

import (
	"bytes"
	"fmt"
)

const (
	maxCornerRadius    = 18.0
	cornerRatioDivisor = 3.0
	textWidthRatio     = 0.6
	textHeightRatio    = 1.2
)

type Simple struct{}

func (Simple) RenderDefs(*bytes.Buffer) {}

func (Simple) RenderBlock(buf *bytes.Buffer, b Block) {
	radius := min(maxCornerRadius, b.W/cornerRatioDivisor, b.H/cornerRatioDivisor)
	WrapURL(buf, b.URL, func() {
		fmt.Fprintf(buf, `<rect id="block-%s" class="block" x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="%.1f" ry="%.1f" fill="white" stroke="#333" stroke-width="1"/>`,
			EscapeXML(b.ID), b.X, b.Y, b.W, b.H, radius, radius)
	})
	buf.WriteByte('\n')
}

func (Simple) RenderEdge(buf *bytes.Buffer, e Edge) {
	fmt.Fprintf(buf, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#333" stroke-width="1.5" stroke-dasharray="6,4"/>`+"\n",
		e.X1, e.Y1, e.X2, e.Y2)
}

func (Simple) RenderText(buf *bytes.Buffer, b Block) {
	rotate := ShouldRotate(b)
	size := FontSize(b)
	if rotate {
		size = FontSizeRotated(b)
	}
	label := TruncateLabel(b, rotate)

	textW, textH := float64(len(label))*size*textWidthRatio, size*textHeightRatio
	if rotate {
		textW, textH = textH, textW
	}

	fmt.Fprintf(buf, `  <g class="block-text" data-block="%s">`+"\n", EscapeXML(b.ID))
	WrapURL(buf, b.URL, func() {
		fmt.Fprintf(buf, `    <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="white"/>`+"\n",
			b.CX-textW/2, b.CY-textH/2, textW, textH)

		if rotate {
			fmt.Fprintf(buf, `    <text x="%.2f" y="%.2f" text-anchor="middle" dominant-baseline="middle" font-family="Times,serif" font-size="%.1f" fill="#333" transform="rotate(-90 %.2f %.2f)">%s</text>`+"\n",
				b.CX, b.CY, size, b.CX, b.CY, EscapeXML(label))
		} else {
			fmt.Fprintf(buf, `    <text x="%.2f" y="%.2f" text-anchor="middle" dominant-baseline="middle" font-family="Times,serif" font-size="%.1f" fill="#333">%s</text>`+"\n",
				b.CX, b.CY, size, EscapeXML(label))
		}
	})
	buf.WriteString("  </g>\n")
}

func (Simple) RenderPopup(*bytes.Buffer, Block) {}
