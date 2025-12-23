package layout

// Block represents a single rectangular element in the tower layout.
// All coordinates are in user units (typically pixels in SVG).
type Block struct {
	NodeID      string
	Left, Right float64
	Bottom, Top float64
}

// Width returns the horizontal span of the block.
func (b Block) Width() float64 { return b.Right - b.Left }

// Height returns the vertical span of the block.
func (b Block) Height() float64 { return b.Top - b.Bottom }

// CenterX returns the horizontal center point of the block.
func (b Block) CenterX() float64 { return (b.Left + b.Right) / 2 }

// CenterY returns the vertical center point of the block.
func (b Block) CenterY() float64 { return (b.Bottom + b.Top) / 2 }
