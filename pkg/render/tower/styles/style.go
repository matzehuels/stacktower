package styles

import "bytes"

type Style interface {
	RenderDefs(buf *bytes.Buffer)
	RenderBlock(buf *bytes.Buffer, b Block)
	RenderEdge(buf *bytes.Buffer, e Edge)
	RenderText(buf *bytes.Buffer, b Block)
	RenderPopup(buf *bytes.Buffer, b Block)
}

type Block struct {
	ID         string
	Label      string
	X, Y, W, H float64
	CX, CY     float64
	URL        string
	Popup      *PopupData
	Brittle    bool
}

type PopupData struct {
	Description string
	Stars       int
	LastCommit  string
	LastRelease string
	Maintainers int
	Archived    bool
	Brittle     bool
}

type Edge struct {
	FromID, ToID   string
	X1, Y1, X2, Y2 float64
}
