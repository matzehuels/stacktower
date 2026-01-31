// Package perm provides permutation generation algorithms and the PQ-tree
// data structure for constrained ordering problems.
//
// # Overview
//
// Finding the optimal row ordering in a layered graph is NP-hard—the number
// of possible orderings grows factorially with row size. This package provides
// tools to make this tractable:
//
//   - [PQTree]: A data structure that compactly represents families of valid
//     permutations, allowing massive pruning of the search space
//   - [Generate]: Efficient permutation generation with optional limits
//   - [Factorial]: Helper for combinatorial calculations
//
// # The PQ-Tree
//
// A PQ-tree represents a set of permutations that satisfy "consecutive ones"
// constraints. If elements {A, B, C} must appear consecutively (in any order),
// the PQ-tree encodes only those permutations where they do.
//
// The tree has two types of internal nodes:
//
//   - P-nodes (Permutable): Children can be arranged in any order (n! orderings)
//   - Q-nodes (seQuence): Children have a fixed order, but can be reversed (2 orderings)
//
// # How It Works
//
// Consider 5 nodes with a constraint that {1, 2, 3} must be consecutive:
//
//	Without PQ-tree: 5! = 120 permutations to check
//	With PQ-tree:    3! × 3! × 2 = 72 valid permutations
//
// As more constraints are applied, the valid count shrinks dramatically. For
// dependency graphs with strong structure (chains, hierarchies), PQ-trees
// often reduce millions of permutations to just a few hundred.
//
// # Basic Usage
//
// Create a tree, apply constraints via [PQTree.Reduce], then enumerate valid
// orderings:
//
//	tree := perm.NewPQTree(5)  // 5 elements: 0, 1, 2, 3, 4
//
//	// Elements 1, 2, 3 must be consecutive
//	tree.Reduce([]int{1, 2, 3})
//
//	// Elements 0, 1 must be consecutive
//	tree.Reduce([]int{0, 1})
//
//	// Get all valid orderings (or first N with a limit)
//	orderings := tree.Enumerate(100)
//
// If a constraint is impossible to satisfy, [PQTree.Reduce] returns false,
// allowing early termination of search branches.
//
// # Permutation Generation
//
// For small sets or when PQ-tree constraints don't apply, use [Generate] for
// efficient permutation enumeration:
//
//	// All 24 permutations of 4 elements
//	all := perm.Generate(4, -1)
//
//	// First 100 permutations of 10 elements (for sampling)
//	sample := perm.Generate(10, 100)
//
// # In the Ordering Pipeline
//
// The ordering algorithms in [render/tower/ordering] use PQ-trees to encode:
//
//   - Chain constraints: Subdivider nodes sharing a MasterID must stay together
//   - Parent constraints: Children of the same parent should be adjacent
//   - Child constraints: Parents of the same child should be adjacent
//
// This dramatically reduces the search space for the branch-and-bound
// algorithm that finds optimal or near-optimal orderings.
//
// [render/tower/ordering]: github.com/matzehuels/stacktower/pkg/core/render/tower/ordering
package perm
