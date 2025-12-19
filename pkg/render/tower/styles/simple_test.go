package styles

import (
	"bytes"
	"strings"
	"testing"
)

func TestSimpleRenderDefs(t *testing.T) {
	s := Simple{}
	var buf bytes.Buffer
	s.RenderDefs(&buf)

	// Simple style has no defs
	if buf.Len() != 0 {
		t.Errorf("RenderDefs() wrote %d bytes, want 0", buf.Len())
	}
}

func TestSimpleRenderBlock(t *testing.T) {
	s := Simple{}

	tests := []struct {
		name     string
		block    Block
		contains []string
	}{
		{
			name: "basic block",
			block: Block{
				ID: "test-pkg",
				X:  10, Y: 20, W: 100, H: 50,
			},
			contains: []string{
				`<rect`,
				`id="block-test-pkg"`,
				`class="block"`,
				`x="10.00"`,
				`y="20.00"`,
				`width="100.00"`,
				`height="50.00"`,
				`fill="white"`,
				`stroke="#333"`,
			},
		},
		{
			name: "block with URL",
			block: Block{
				ID:  "linked-pkg",
				URL: "https://example.com",
				X:   0, Y: 0, W: 80, H: 40,
			},
			contains: []string{
				`<a href="https://example.com"`,
				`target="_blank"`,
				`</a>`,
			},
		},
		{
			name: "block with special chars in ID",
			block: Block{
				ID: "pkg<script>",
				X:  0, Y: 0, W: 50, H: 50,
			},
			contains: []string{
				`id="block-pkg&lt;script&gt;"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			s.RenderBlock(&buf, tt.block)
			output := buf.String()

			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("RenderBlock() output missing %q\nGot: %s", want, output)
				}
			}
		})
	}
}

func TestSimpleRenderBlockCornerRadius(t *testing.T) {
	s := Simple{}

	// Small block should have smaller radius
	smallBlock := Block{ID: "small", X: 0, Y: 0, W: 30, H: 30}
	var buf bytes.Buffer
	s.RenderBlock(&buf, smallBlock)
	output := buf.String()

	// rx and ry should be present
	if !strings.Contains(output, "rx=") || !strings.Contains(output, "ry=") {
		t.Error("RenderBlock() should include corner radius")
	}
}

func TestSimpleRenderEdge(t *testing.T) {
	s := Simple{}

	edge := Edge{
		FromID: "a",
		ToID:   "b",
		X1:     10, Y1: 20,
		X2: 100, Y2: 200,
	}

	var buf bytes.Buffer
	s.RenderEdge(&buf, edge)
	output := buf.String()

	expected := []string{
		`<line`,
		`x1="10.00"`,
		`y1="20.00"`,
		`x2="100.00"`,
		`y2="200.00"`,
		`stroke="#333"`,
		`stroke-width="1.5"`,
		`stroke-dasharray="6,4"`,
	}

	for _, want := range expected {
		if !strings.Contains(output, want) {
			t.Errorf("RenderEdge() output missing %q\nGot: %s", want, output)
		}
	}
}

func TestSimpleRenderText(t *testing.T) {
	s := Simple{}

	tests := []struct {
		name     string
		block    Block
		contains []string
	}{
		{
			name: "horizontal text",
			block: Block{
				ID:    "pkg",
				Label: "pkg",
				X:     0, Y: 0, W: 100, H: 30,
				CX: 50, CY: 15,
			},
			contains: []string{
				`<g class="block-text"`,
				`data-block="pkg"`,
				`<text`,
				`text-anchor="middle"`,
				`font-family="Times,serif"`,
				`>pkg</text>`,
			},
		},
		{
			name: "text with URL",
			block: Block{
				ID:    "linked",
				Label: "linked",
				URL:   "https://example.com",
				X:     0, Y: 0, W: 100, H: 30,
				CX: 50, CY: 15,
			},
			contains: []string{
				`<a href="https://example.com"`,
				`target="_blank"`,
			},
		},
		{
			name: "rotated text (tall narrow block)",
			block: Block{
				ID:    "tall-package",
				Label: "tall-package",
				X:     0, Y: 0, W: 30, H: 100,
				CX: 15, CY: 50,
			},
			contains: []string{
				`transform="rotate(-90`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			s.RenderText(&buf, tt.block)
			output := buf.String()

			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("RenderText() output missing %q\nGot: %s", want, output)
				}
			}
		})
	}
}

func TestSimpleRenderTextEscapesXML(t *testing.T) {
	s := Simple{}

	block := Block{
		ID:    "<script>",
		Label: "A & B",
		X:     0, Y: 0, W: 100, H: 30,
		CX: 50, CY: 15,
	}

	var buf bytes.Buffer
	s.RenderText(&buf, block)
	output := buf.String()

	if strings.Contains(output, "<script>") {
		t.Error("RenderText() should escape < in ID")
	}
	if strings.Contains(output, "A & B") && !strings.Contains(output, "A &amp; B") {
		t.Error("RenderText() should escape & in label")
	}
}

func TestSimpleRenderPopup(t *testing.T) {
	s := Simple{}

	block := Block{
		ID: "test",
		Popup: &PopupData{
			Description: "Test description",
			Stars:       100,
		},
	}

	var buf bytes.Buffer
	s.RenderPopup(&buf, block)

	// Simple style has no popup implementation
	if buf.Len() != 0 {
		t.Errorf("RenderPopup() wrote %d bytes, want 0 (Simple has no popups)", buf.Len())
	}
}

func TestSimpleImplementsStyle(t *testing.T) {
	// Compile-time check that Simple implements Style
	var _ Style = Simple{}
}
