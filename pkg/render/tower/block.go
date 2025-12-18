package tower

type Block struct {
	NodeID string  `json:"id"`
	Left   float64 `json:"x"`
	Right  float64 `json:"-"`
	Bottom float64 `json:"y"`
	Top    float64 `json:"-"`
}

func (b Block) Width() float64   { return b.Right - b.Left }
func (b Block) Height() float64  { return b.Top - b.Bottom }
func (b Block) CenterX() float64 { return (b.Left + b.Right) / 2 }
func (b Block) CenterY() float64 { return (b.Bottom + b.Top) / 2 }

type jsonBlock struct {
	ID     string  `json:"id"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func (b Block) MarshalJSON() ([]byte, error) {
	return jsonMarshal(jsonBlock{
		ID:     b.NodeID,
		X:      b.Left,
		Y:      b.Bottom,
		Width:  b.Width(),
		Height: b.Height(),
	})
}
