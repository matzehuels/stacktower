package sink

import "github.com/matzehuels/stacktower/pkg/render/tower/layout"

type PDFOption func(*pdfRenderer)

type pdfRenderer struct {
	svgOpts []SVGOption
}

func WithPDFSVGOptions(opts ...SVGOption) PDFOption {
	return func(r *pdfRenderer) { r.svgOpts = opts }
}

func RenderPDF(l layout.Layout, opts ...PDFOption) ([]byte, error) {
	r := pdfRenderer{}
	for _, opt := range opts {
		opt(&r)
	}
	svg := RenderSVG(l, r.svgOpts...)
	return rsvgConvert(svg, "pdf")
}
