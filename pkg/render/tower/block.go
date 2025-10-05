package tower

type Block struct {
	NodeID      string
	Left, Right float64
	Bottom, Top float64
}

func (b Block) Width() float64  { return b.Right - b.Left }
func (b Block) Height() float64 { return b.Top - b.Bottom }

func (b Block) CenterX() float64 { return (b.Left + b.Right) / 2 }
func (b Block) CenterY() float64 { return (b.Bottom + b.Top) / 2 }
