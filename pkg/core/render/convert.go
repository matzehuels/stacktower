package render

import (
	"bytes"
	"fmt"
	"os/exec"
)

// ToPDF converts SVG bytes to PDF using rsvg-convert.
// Requires librsvg: brew install librsvg (macOS), apt install librsvg2-bin (Linux).
func ToPDF(svg []byte) ([]byte, error) {
	return rsvgConvert(svg, "pdf")
}

// ToPNG converts SVG bytes to PNG using rsvg-convert with the given scale factor.
// Scale of 2.0 produces a 2x resolution image.
// Requires librsvg: brew install librsvg (macOS), apt install librsvg2-bin (Linux).
func ToPNG(svg []byte, scale float64) ([]byte, error) {
	return rsvgConvert(svg, "png", "-z", fmt.Sprintf("%.2f", scale))
}

// rsvgConvert shells out to rsvg-convert for format conversion.
func rsvgConvert(svg []byte, format string, extraArgs ...string) ([]byte, error) {
	if _, err := exec.LookPath("rsvg-convert"); err != nil {
		return nil, fmt.Errorf("%s export requires librsvg. Install with:\n  macOS:  brew install librsvg\n  Linux:  apt install librsvg2-bin", format)
	}

	args := append([]string{"-f", format}, extraArgs...)
	cmd := exec.Command("rsvg-convert", args...)
	cmd.Stdin = bytes.NewReader(svg)

	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("rsvg-convert: %v: %s", err, errBuf.String())
	}
	return out.Bytes(), nil
}
