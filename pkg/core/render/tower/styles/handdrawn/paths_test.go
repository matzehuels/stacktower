package handdrawn

import (
	"strings"
	"testing"
)

func TestWobbledRect(t *testing.T) {
	// Basic test
	path := wobbledRect(10, 20, 100, 50, 42, "test-block")

	// Should start with M (moveto)
	if !strings.HasPrefix(path, "M") {
		t.Errorf("wobbledRect() should start with M, got: %s", path)
	}

	// Should end with Z (close path)
	if !strings.HasSuffix(path, "Z") {
		t.Errorf("wobbledRect() should end with Z, got: %s", path)
	}

	// Should contain Q (quadratic bezier) for curves
	if !strings.Contains(path, "Q") {
		t.Errorf("wobbledRect() should contain Q commands, got: %s", path)
	}

	// Deterministic - same inputs produce same output
	path2 := wobbledRect(10, 20, 100, 50, 42, "test-block")
	if path != path2 {
		t.Errorf("wobbledRect() should be deterministic")
	}

	// Different IDs produce different paths
	path3 := wobbledRect(10, 20, 100, 50, 42, "other-block")
	if path == path3 {
		t.Errorf("wobbledRect() should produce different paths for different IDs")
	}
}

func TestWobbledRect_SmallRect(t *testing.T) {
	// Very small rectangle should still work
	path := wobbledRect(0, 0, 5, 5, 42, "tiny")
	if path == "" {
		t.Error("wobbledRect() should produce output for small rectangles")
	}
	if !strings.HasPrefix(path, "M") {
		t.Errorf("small wobbledRect() should start with M, got: %s", path)
	}
}

func TestCurvedEdge(t *testing.T) {
	tests := []struct {
		name         string
		x1, y1       float64
		x2, y2       float64
		wantCurve    bool
		wantContains string
	}{
		{
			name: "short edge (straight line)",
			x1:   0, y1: 0,
			x2: 20, y2: 20,
			wantCurve:    false,
			wantContains: "L",
		},
		{
			name: "long edge (curved)",
			x1:   0, y1: 0,
			x2: 100, y2: 100,
			wantCurve:    true,
			wantContains: "C",
		},
		{
			name: "horizontal long edge",
			x1:   0, y1: 50,
			x2: 200, y2: 50,
			wantCurve:    true,
			wantContains: "C",
		},
		{
			name: "vertical long edge",
			x1:   50, y1: 0,
			x2: 50, y2: 200,
			wantCurve:    true,
			wantContains: "C",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := curvedEdge(tt.x1, tt.y1, tt.x2, tt.y2)

			if !strings.HasPrefix(path, "M") {
				t.Errorf("curvedEdge() should start with M, got: %s", path)
			}

			if !strings.Contains(path, tt.wantContains) {
				t.Errorf("curvedEdge() should contain %q, got: %s", tt.wantContains, path)
			}
		})
	}
}

func TestRotationFor(t *testing.T) {
	// Same ID should produce same rotation
	rot1 := rotationFor("test", 100, 50)
	rot2 := rotationFor("test", 100, 50)
	if rot1 != rot2 {
		t.Errorf("rotationFor() should be deterministic")
	}

	// Different IDs should produce different rotations
	rot3 := rotationFor("other", 100, 50)
	if rot1 == rot3 {
		t.Errorf("rotationFor() should produce different values for different IDs")
	}

	// Rotation should be small (within reasonable bounds)
	for _, id := range []string{"a", "b", "c", "test", "package"} {
		rot := rotationFor(id, 100, 50)
		if rot < -10 || rot > 10 {
			t.Errorf("rotationFor(%q) = %f, expected within [-10, 10]", id, rot)
		}
	}
}

func TestRNG(t *testing.T) {
	rng := newRNG(42)
	if rng == nil {
		t.Fatal("newRNG() returned nil")
	}

	// Test that it produces values in [0, 1)
	for i := 0; i < 100; i++ {
		v := rng.next()
		if v < 0 || v >= 1 {
			t.Errorf("rng.next() = %f, should be in [0, 1)", v)
		}
	}

	// Test determinism
	rng1 := newRNG(42)
	rng2 := newRNG(42)
	for i := 0; i < 10; i++ {
		v1, v2 := rng1.next(), rng2.next()
		if v1 != v2 {
			t.Errorf("rng should be deterministic: %f != %f", v1, v2)
		}
	}

	// Test different seeds produce different sequences
	rng3 := newRNG(43)
	rng1 = newRNG(42)
	different := false
	for i := 0; i < 10; i++ {
		if rng1.next() != rng3.next() {
			different = true
			break
		}
	}
	if !different {
		t.Error("different seeds should produce different sequences")
	}
}
