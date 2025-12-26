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

func TestPQTree_Clone(t *testing.T) {
	// Test that clone creates independent copy
	original := NewPQTree(5)
	original.Reduce([]int{0, 1, 2})

	originalCountBefore := original.ValidCount()
	originalStringBefore := original.String()

	clone := original.Clone()

	// Verify initial state matches
	if clone.ValidCount() != originalCountBefore {
		t.Errorf("Clone ValidCount %d != original %d", clone.ValidCount(), originalCountBefore)
	}

	if clone.String() != originalStringBefore {
		t.Errorf("Clone structure doesn't match:\nClone:    %s\nOriginal: %s", clone.String(), originalStringBefore)
	}

	// Modify clone - should not affect original
	ok := clone.Reduce([]int{3, 4})
	if !ok {
		t.Fatal("Clone reduce failed")
	}

	cloneCountAfter := clone.ValidCount()
	originalCountAfter := original.ValidCount()
	originalStringAfter := original.String()

	t.Logf("Original before: count=%d, structure=%s", originalCountBefore, originalStringBefore)
	t.Logf("Original after:  count=%d, structure=%s", originalCountAfter, originalStringAfter)
	t.Logf("Clone after:     count=%d, structure=%s", cloneCountAfter, clone.String())

	// Verify original unchanged
	if originalCountAfter != originalCountBefore {
		t.Errorf("Original ValidCount changed from %d to %d", originalCountBefore, originalCountAfter)
	}

	if originalStringAfter != originalStringBefore {
		t.Errorf("Original structure changed from %s to %s", originalStringBefore, originalStringAfter)
	}

	// Verify clone changed
	if cloneCountAfter >= originalCountBefore {
		t.Errorf("Clone ValidCount should be less than %d after constraint, got %d", originalCountBefore, cloneCountAfter)
	}
}

func TestPQTree_CloneEmpty(t *testing.T) {
	original := NewPQTree(0)
	clone := original.Clone()

	if clone.ValidCount() != 1 {
		t.Errorf("Empty clone ValidCount = %d, want 1", clone.ValidCount())
	}
}

func TestPQTree_EnumerateFunc(t *testing.T) {
	tree := NewPQTree(3)

	// Collect via EnumerateFunc
	var collected [][]int
	count := tree.EnumerateFunc(func(perm []int) bool {
		collected = append(collected, slices.Clone(perm))
		return true
	})

	if count != 6 {
		t.Errorf("EnumerateFunc returned count %d, want 6", count)
	}

	if len(collected) != 6 {
		t.Errorf("Collected %d permutations, want 6", len(collected))
	}

	// Verify all unique
	seen := make(map[string]bool)
	for _, p := range collected {
		key := ""
		for _, v := range p {
			key += string(rune('0' + v))
		}
		if seen[key] {
			t.Errorf("EnumerateFunc generated duplicate: %v", p)
		}
		seen[key] = true
	}
}

func TestPQTree_EnumerateFuncEarlyStop(t *testing.T) {
	tree := NewPQTree(4)

	// Stop after 3 permutations
	stopAfter := 3
	collected := 0
	count := tree.EnumerateFunc(func(perm []int) bool {
		collected++
		return collected < stopAfter
	})

	if count != stopAfter {
		t.Errorf("EnumerateFunc count = %d, want %d", count, stopAfter)
	}

	if collected != stopAfter {
		t.Errorf("Collected %d permutations, want %d", collected, stopAfter)
	}
}

func TestPQTree_EnumerateFuncEmptyTree(t *testing.T) {
	tree := NewPQTree(0)

	called := false
	count := tree.EnumerateFunc(func(perm []int) bool {
		called = true
		if len(perm) != 0 {
			t.Errorf("Expected empty permutation, got %v", perm)
		}
		return true
	})

	if !called {
		t.Error("EnumerateFunc should call function for empty tree")
	}

	if count != 1 {
		t.Errorf("EnumerateFunc count = %d, want 1", count)
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
