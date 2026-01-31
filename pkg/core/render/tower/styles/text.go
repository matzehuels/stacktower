package styles

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

const (
	fontHeightRatio  = 0.6
	fontWidthRatio   = 0.85
	fontCharWidth    = 0.55
	fontSizeMin      = 8.0
	fontSizeMax      = 24.0
	rotateSizeDampen = 0.75
)

func FontSize(b Block) float64        { return fontSizeFor(b.W, b.H, len(b.ID)) }
func FontSizeRotated(b Block) float64 { return fontSizeFor(b.H*rotateSizeDampen, b.W, len(b.ID)) }

func fontSizeFor(availWidth, availHeight float64, textLen int) float64 {
	n := max(1, textLen)
	byHeight := availHeight * fontHeightRatio
	byWidth := (availWidth * fontWidthRatio) / (float64(n) * fontCharWidth)
	return max(fontSizeMin, min(fontSizeMax, min(byHeight, byWidth)))
}

func ShouldRotate(b Block) bool {
	horizSize := fontSizeFor(b.W, b.H, len(b.ID))
	rotSize := fontSizeFor(b.H, b.W, len(b.ID))
	if len(b.ID) > 10 {
		return rotSize*1.1 >= horizSize
	}
	return rotSize > horizSize
}

func TruncateLabel(b Block, rotated bool) string {
	label := b.Label
	availW := b.W * fontWidthRatio
	if rotated {
		availW = b.H * fontWidthRatio
	}

	fontSize := FontSize(b)
	if rotated {
		fontSize = FontSizeRotated(b)
	}

	charWidth := fontSize * fontCharWidth
	maxChars := int(availW / charWidth)
	if maxChars < 3 {
		maxChars = 3
	}

	if len(label) <= maxChars {
		return label
	}
	return label[:maxChars-2] + ".."
}

func EscapeXML(s string) string {
	var buf bytes.Buffer
	xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

func WrapURL(buf *bytes.Buffer, url string, fn func()) {
	if url != "" {
		fmt.Fprintf(buf, `  <a href="%s" target="_blank">`, EscapeXML(url))
	}
	fn()
	if url != "" {
		buf.WriteString("</a>")
	}
}
