package sink

import (
	"github.com/matzehuels/stacktower/pkg/render"
	"github.com/matzehuels/stacktower/pkg/render/tower/layout"
)

// PDFOption configures PDF rendering.
type PDFOption func(*pdfRenderer)

type pdfRenderer struct {
	svgOpts []SVGOption
}

// WithPDFSVGOptions passes options through to the underlying SVG renderer.
func WithPDFSVGOptions(opts ...SVGOption) PDFOption {
	return func(r *pdfRenderer) { r.svgOpts = opts }
}

// RenderPDF renders the layout as PDF via SVG conversion.
// Requires librsvg: brew install librsvg (macOS), apt install librsvg2-bin (Linux).
func RenderPDF(l layout.Layout, opts ...PDFOption) ([]byte, error) {
	r := pdfRenderer{}
	for _, opt := range opts {
		opt(&r)
	}
	svg := RenderSVG(l, r.svgOpts...)
	return render.ToPDF(svg)
}
