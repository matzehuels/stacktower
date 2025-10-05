package perm

import (
	"slices"
	"testing"
)

func TestNewPQTree_UniversalTree(t *testing.T) {
	tree := NewPQTree(4)

	if count := tree.ValidCount(); count != 24 {
		t.Errorf("expected 24 permutations, got %d", count)
	}

	if perms := tree.Enumerate(0); len(perms) != 24 {
		t.Errorf("expected 24 permutations, got %d", len(perms))
	}
}

func TestPQTree_SingleConstraint(t *testing.T) {
	tree := NewPQTree(4)

	if !tree.Reduce([]int{0, 1, 2}) {
		t.Fatal("reduction should succeed")
	}

	perms := tree.Enumerate(0)

	for _, perm := range perms {
		if !areConsecutive(perm, []int{0, 1, 2}) {
			t.Errorf("constraint violated in permutation %v", perm)
		}
	}

	if len(perms) >= 24 {
		t.Errorf("expected fewer permutations after constraint, got %d", len(perms))
	}

	t.Logf("Permutations after constraint [0,1,2]: %d", len(perms))
	t.Logf("Tree: %s", tree.String())
}

func TestPQTree_TwoConstraints(t *testing.T) {
	tree := NewPQTree(4)

	if !tree.Reduce([]int{0, 1}) {
		t.Fatal("first reduction should succeed")
	}

	if !tree.Reduce([]int{2, 3}) {
		t.Fatal("second reduction should succeed")
	}

	perms := tree.Enumerate(0)

	for _, perm := range perms {
		if !areConsecutive(perm, []int{0, 1}) {
			t.Errorf("constraint [0,1] violated in permutation %v", perm)
		}
		if !areConsecutive(perm, []int{2, 3}) {
			t.Errorf("constraint [2,3] violated in permutation %v", perm)
		}
	}

	if len(perms) != 8 {
		t.Errorf("expected 8 permutations, got %d", len(perms))
	}

	t.Logf("Tree: %s", tree.String())
}

func TestPQTree_OverlappingConstraints(t *testing.T) {
	tree := NewPQTree(4)

	if !tree.Reduce([]int{0, 1}) {
		t.Fatal("first reduction should succeed")
	}

	if !tree.Reduce([]int{1, 2}) {
		t.Fatal("second reduction should succeed")
	}

	perms := tree.Enumerate(0)

	for _, perm := range perms {
		if !areConsecutive(perm, []int{0, 1}) {
			t.Errorf("constraint [0,1] violated in permutation %v", perm)
		}
		if !areConsecutive(perm, []int{1, 2}) {
			t.Errorf("constraint [1,2] violated in permutation %v", perm)
		}
	}

	t.Logf("Permutations after overlapping constraints: %d", len(perms))
	t.Logf("Tree: %s", tree.String())
}

func TestPQTree_EmptyAndTrivial(t *testing.T) {
	tree := NewPQTree(0)
	perms := tree.Enumerate(0)
	if len(perms) != 1 || len(perms[0]) != 0 {
		t.Error("empty tree should have one empty permutation")
	}

	tree = NewPQTree(1)
	perms = tree.Enumerate(0)
	if len(perms) != 1 || !slices.Equal(perms[0], []int{0}) {
		t.Error("single element tree should have one permutation [0]")
	}

	tree = NewPQTree(3)
	if !tree.Reduce([]int{1}) {
		t.Fatal("trivial constraint should succeed")
	}
	if tree.ValidCount() != 6 {
		t.Errorf("trivial constraint should not change count, got %d", tree.ValidCount())
	}
}

func TestPQTree_EnumerateLimit(t *testing.T) {
	tree := NewPQTree(5)

	perms := tree.Enumerate(10)
	if len(perms) != 10 {
		t.Errorf("expected 10 permutations with limit, got %d", len(perms))
	}
}

func TestPQTree_ValidCount(t *testing.T) {
	tests := []struct {
		n           int
		constraints [][]int
		want        int
	}{
		{3, nil, 6},
		{4, nil, 24},
		{4, [][]int{{0, 1}}, 12},
		{4, [][]int{{0, 1}, {2, 3}}, 8},
	}

	for _, tt := range tests {
		tree := NewPQTree(tt.n)
		for _, c := range tt.constraints {
			tree.Reduce(c)
		}
		if got := tree.ValidCount(); got != tt.want {
			t.Errorf("n=%d constraints=%v: got %d, want %d", tt.n, tt.constraints, got, tt.want)
		}
	}
}

func areConsecutive(perm, subset []int) bool {
	if len(subset) <= 1 {
		return true
	}

	subsetSet := make(map[int]bool, len(subset))
	for _, e := range subset {
		subsetSet[e] = true
	}

	positions := make([]int, 0, len(subset))
	for i, e := range perm {
		if subsetSet[e] {
			positions = append(positions, i)
		}
	}

	if len(positions) != len(subset) {
		return false
	}

	slices.Sort(positions)
	for i := 1; i < len(positions); i++ {
		if positions[i] != positions[i-1]+1 {
			return false
		}
	}
	return true
}
