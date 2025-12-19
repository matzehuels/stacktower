package perm_test

import (
	"fmt"

	"github.com/matzehuels/stacktower/pkg/dag/perm"
)

func ExampleGenerate() {
	// Generate all permutations of 3 elements
	perms := perm.Generate(3, -1)
	fmt.Println("All permutations of [0,1,2]:")
	for _, p := range perms {
		fmt.Println(p)
	}
	// Output:
	// All permutations of [0,1,2]:
	// [0 1 2]
	// [1 0 2]
	// [2 0 1]
	// [0 2 1]
	// [1 2 0]
	// [2 1 0]
}

func ExampleGenerate_limited() {
	// Generate only the first 5 permutations of 10 elements
	perms := perm.Generate(10, 5)
	fmt.Println("Count:", len(perms))
	// Output:
	// Count: 5
}

func ExampleFactorial() {
	fmt.Println("4! =", perm.Factorial(4))
	fmt.Println("5! =", perm.Factorial(5))
	// Output:
	// 4! = 24
	// 5! = 120
}

func ExamplePQTree_basic() {
	// Create a PQ-tree for 5 elements
	tree := perm.NewPQTree(5)

	// Without constraints: all 5! = 120 permutations are valid
	fmt.Println("Before constraints:", tree.ValidCount())

	// Require {1, 2, 3} to be consecutive
	tree.Reduce([]int{1, 2, 3})
	fmt.Println("After {1,2,3} consecutive:", tree.ValidCount())
	// Output:
	// Before constraints: 120
	// After {1,2,3} consecutive: 36
}

func ExamplePQTree_Reduce() {
	// PQ-trees encode "consecutive ones" constraints
	tree := perm.NewPQTree(5)

	// Elements 0, 1 must be consecutive
	ok := tree.Reduce([]int{0, 1})
	fmt.Println("Constraint possible:", ok)

	// Elements 2, 3 must also be consecutive
	ok = tree.Reduce([]int{2, 3})
	fmt.Println("Constraint possible:", ok)

	fmt.Println("Valid orderings:", tree.ValidCount())
	// Output:
	// Constraint possible: true
	// Constraint possible: true
	// Valid orderings: 24
}

func ExamplePQTree_impossible() {
	// Some constraint combinations are impossible
	tree := perm.NewPQTree(4)

	// {0, 1} consecutive and {2, 3} consecutive
	tree.Reduce([]int{0, 1})
	tree.Reduce([]int{2, 3})

	// Now try: {1, 2} consecutive (impossible - would require 0,1,2,3 all consecutive)
	// Actually this IS possible: [0 1 2 3] or [3 2 1 0], etc.
	// Let's try a truly impossible case:
	tree2 := perm.NewPQTree(4)
	tree2.Reduce([]int{0, 2}) // 0 and 2 must be adjacent
	tree2.Reduce([]int{1, 3}) // 1 and 3 must be adjacent

	// Now require 0 and 1 adjacent - impossible since 0 is paired with 2, and 1 with 3
	ok := tree2.Reduce([]int{0, 1})
	fmt.Println("Contradictory constraint:", !ok)
	// Output:
	// Contradictory constraint: true
}

func ExamplePQTree_Enumerate() {
	tree := perm.NewPQTree(4)

	// Require {0, 1, 2} consecutive
	tree.Reduce([]int{0, 1, 2})

	// Get first 5 valid orderings
	orderings := tree.Enumerate(5)
	fmt.Println("Sample orderings (limit 5):")
	for _, o := range orderings {
		fmt.Println(o)
	}
	// Output:
	// Sample orderings (limit 5):
	// [0 1 2 3]
	// [1 0 2 3]
	// [2 0 1 3]
	// [0 2 1 3]
	// [1 2 0 3]
}

func ExamplePQTree_String() {
	tree := perm.NewPQTree(5)

	// P-nodes (permutable) shown with {}, Q-nodes (sequence) with []
	fmt.Println("Initial:", tree.String())

	// Add constraint - groups elements into a P-node (they can be in any order)
	tree.Reduce([]int{1, 2, 3})
	fmt.Println("After constraint:", tree.String())
	// Output:
	// Initial: {0 1 2 3 4}
	// After constraint: {0 {1 2 3} 4}
}

func ExamplePQTree_StringWithLabels() {
	// Use meaningful labels instead of indices
	tree := perm.NewPQTree(4)
	labels := []string{"app", "auth", "cache", "db"}

	tree.Reduce([]int{1, 2}) // auth and cache consecutive

	fmt.Println(tree.StringWithLabels(labels))
	// Output:
	// {app {auth cache} db}
}

func ExampleSeq() {
	// Create a sequence [0, 1, 2, ..., n-1]
	seq := perm.Seq(5)
	fmt.Println(seq)
	// Output:
	// [0 1 2 3 4]
}
