package sink

import (
	"github.com/matzehuels/stacktower/pkg/render"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
)

// PNGOption configures PNG rendering.
type PNGOption func(*pngRenderer)

type pngRenderer struct {
	svgOpts []SVGOption
	scale   float64
}

// WithPNGSVGOptions passes options through to the underlying SVG renderer.
func WithPNGSVGOptions(opts ...SVGOption) PNGOption {
	return func(r *pngRenderer) { r.svgOpts = opts }
}

// WithScale sets the PNG scale factor (default 2.0 for 2x resolution).
func WithScale(s float64) PNGOption {
	return func(r *pngRenderer) { r.scale = s }
}

// RenderPNG renders the layout as PNG via SVG conversion.
// Requires librsvg: brew install librsvg (macOS), apt install librsvg2-bin (Linux).
func RenderPNG(l layout.Layout, opts ...PNGOption) ([]byte, error) {
	r := pngRenderer{scale: 2.0}
	for _, opt := range opts {
		opt(&r)
	}
	svg := RenderSVG(l, r.svgOpts...)
	return render.ToPNG(svg, r.scale)
}
