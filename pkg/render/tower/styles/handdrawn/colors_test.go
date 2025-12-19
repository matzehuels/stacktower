package handdrawn

import (
	"fmt"
	"regexp"
	"testing"
)

func TestGreyForID(t *testing.T) {
	// Test that different IDs produce different colors
	id1 := "package-a"
	id2 := "package-b"

	grey1 := greyForID(id1)
	grey2 := greyForID(id2)

	if grey1 == grey2 {
		t.Errorf("greyForID() should produce different colors for different IDs")
	}

	// Test that same ID produces same color (deterministic)
	if greyForID(id1) != greyForID(id1) {
		t.Errorf("greyForID() should be deterministic")
	}

	// Test that output is a valid hex color
	hexColorRegex := regexp.MustCompile(`^#[0-9a-f]{6}$`)
	if !hexColorRegex.MatchString(grey1) {
		t.Errorf("greyForID() should produce valid hex color, got %q", grey1)
	}
}

func TestGreyForID_Range(t *testing.T) {
	// Test many IDs to ensure colors are within expected range
	for i := 0; i < 100; i++ {
		id := string(rune('a' + i%26))
		grey := greyForID(id)

		// Parse the hex color
		var r, g, b int
		_, err := fmt.Sscanf(grey, "#%02x%02x%02x", &r, &g, &b)
		if err != nil {
			t.Errorf("failed to parse color %q: %v", grey, err)
			continue
		}

		// Check that it's a grey (r == g == b)
		if r != g || g != b {
			t.Errorf("greyForID(%q) = %q is not a grey color", id, grey)
		}

		// Check range
		if r < greyMin || r > greyMax {
			t.Errorf("greyForID(%q) = %q value %d outside range [%d, %d]",
				id, grey, r, greyMin, greyMax)
		}
	}
}

func TestHash(t *testing.T) {
	// Same input, same seed should produce same hash
	h1 := hash("test", 42)
	h2 := hash("test", 42)
	if h1 != h2 {
		t.Errorf("hash() should be deterministic")
	}

	// Same input, different seed should produce different hash
	h3 := hash("test", 43)
	if h1 == h3 {
		t.Errorf("hash() with different seed should produce different hash")
	}

	// Different input, same seed should produce different hash
	h4 := hash("other", 42)
	if h1 == h4 {
		t.Errorf("hash() with different input should produce different hash")
	}

	// Zero seed still works
	h5 := hash("test", 0)
	h6 := hash("test", 0)
	if h5 != h6 {
		t.Errorf("hash() with zero seed should be deterministic")
	}
}
