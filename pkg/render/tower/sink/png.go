package sink

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
)

type PNGOption func(*pngRenderer)

type pngRenderer struct {
	svgOpts []SVGOption
	scale   float64
}

func WithPNGSVGOptions(opts ...SVGOption) PNGOption {
	return func(r *pngRenderer) { r.svgOpts = opts }
}

func WithScale(s float64) PNGOption {
	return func(r *pngRenderer) { r.scale = s }
}

func RenderPNG(l layout.Layout, opts ...PNGOption) ([]byte, error) {
	r := pngRenderer{scale: 2.0}
	for _, opt := range opts {
		opt(&r)
	}
	svg := RenderSVG(l, r.svgOpts...)
	return rsvgConvert(svg, "png", "-z", fmt.Sprintf("%.2f", r.scale))
}

// Requires librsvg: brew install librsvg (macOS), apt install librsvg2-bin (Linux)
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
