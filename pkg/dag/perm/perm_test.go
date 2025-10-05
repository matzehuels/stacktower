package perm

import (
	"slices"
	"testing"
)

func TestSeq(t *testing.T) {
	tests := []struct {
		n    int
		want []int
	}{
		{0, []int{}},
		{1, []int{0}},
		{3, []int{0, 1, 2}},
		{5, []int{0, 1, 2, 3, 4}},
	}

	for _, tt := range tests {
		got := Seq(tt.n)
		if !slices.Equal(got, tt.want) {
			t.Errorf("Seq(%d) = %v, want %v", tt.n, got, tt.want)
		}
	}
}

func TestFactorial(t *testing.T) {
	tests := []struct {
		n    int
		want int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 6},
		{4, 24},
		{5, 120},
		{6, 720},
	}

	for _, tt := range tests {
		got := Factorial(tt.n)
		if got != tt.want {
			t.Errorf("Factorial(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestGenerate(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		perms := Generate(0, -1)
		if len(perms) != 1 || len(perms[0]) != 0 {
			t.Error("Generate(0) should return one empty slice")
		}
	})

	t.Run("single", func(t *testing.T) {
		perms := Generate(1, -1)
		if len(perms) != 1 || !slices.Equal(perms[0], []int{0}) {
			t.Errorf("Generate(1) = %v, want [[0]]", perms)
		}
	})

	t.Run("generates all", func(t *testing.T) {
		perms := Generate(4, -1)
		if len(perms) != 24 {
			t.Errorf("Generate(4) generated %d, want 24", len(perms))
		}

		seen := make(map[string]bool)
		for _, p := range perms {
			key := ""
			for _, v := range p {
				key += string(rune('0' + v))
			}
			if seen[key] {
				t.Errorf("duplicate permutation: %v", p)
			}
			seen[key] = true
		}
	})

	t.Run("respects limit", func(t *testing.T) {
		perms := Generate(5, 10)
		if len(perms) != 10 {
			t.Errorf("Generate(5, 10) generated %d, want 10", len(perms))
		}
	})

	t.Run("limit larger than total", func(t *testing.T) {
		perms := Generate(3, 100)
		if len(perms) != 6 {
			t.Errorf("Generate(3, 100) generated %d, want 6", len(perms))
		}
	})
}
