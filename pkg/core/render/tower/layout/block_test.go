package layout

import "testing"

func TestBlockWidth(t *testing.T) {
	tests := []struct {
		name  string
		block Block
		want  float64
	}{
		{
			name:  "positive width",
			block: Block{Left: 10, Right: 50},
			want:  40,
		},
		{
			name:  "zero width",
			block: Block{Left: 10, Right: 10},
			want:  0,
		},
		{
			name:  "from origin",
			block: Block{Left: 0, Right: 100},
			want:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.block.Width(); got != tt.want {
				t.Errorf("Width() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockHeight(t *testing.T) {
	tests := []struct {
		name  string
		block Block
		want  float64
	}{
		{
			name:  "positive height",
			block: Block{Bottom: 20, Top: 80},
			want:  60,
		},
		{
			name:  "zero height",
			block: Block{Bottom: 50, Top: 50},
			want:  0,
		},
		{
			name:  "from origin",
			block: Block{Bottom: 0, Top: 100},
			want:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.block.Height(); got != tt.want {
				t.Errorf("Height() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockCenterX(t *testing.T) {
	tests := []struct {
		name  string
		block Block
		want  float64
	}{
		{
			name:  "symmetric",
			block: Block{Left: 0, Right: 100},
			want:  50,
		},
		{
			name:  "offset",
			block: Block{Left: 20, Right: 80},
			want:  50,
		},
		{
			name:  "zero width",
			block: Block{Left: 50, Right: 50},
			want:  50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.block.CenterX(); got != tt.want {
				t.Errorf("CenterX() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockCenterY(t *testing.T) {
	tests := []struct {
		name  string
		block Block
		want  float64
	}{
		{
			name:  "symmetric",
			block: Block{Bottom: 0, Top: 100},
			want:  50,
		},
		{
			name:  "offset",
			block: Block{Bottom: 30, Top: 70},
			want:  50,
		},
		{
			name:  "zero height",
			block: Block{Bottom: 50, Top: 50},
			want:  50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.block.CenterY(); got != tt.want {
				t.Errorf("CenterY() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBlockNodeID(t *testing.T) {
	block := Block{NodeID: "test-node"}
	if block.NodeID != "test-node" {
		t.Errorf("NodeID = %q, want %q", block.NodeID, "test-node")
	}
}

func TestBlockDimensions(t *testing.T) {
	block := Block{
		NodeID: "test",
		Left:   10,
		Right:  60,
		Bottom: 20,
		Top:    70,
	}

	if block.Width() != 50 {
		t.Errorf("Width() = %v, want 50", block.Width())
	}
	if block.Height() != 50 {
		t.Errorf("Height() = %v, want 50", block.Height())
	}
	if block.CenterX() != 35 {
		t.Errorf("CenterX() = %v, want 35", block.CenterX())
	}
	if block.CenterY() != 45 {
		t.Errorf("CenterY() = %v, want 45", block.CenterY())
	}
}
