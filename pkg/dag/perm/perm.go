package perm

import "slices"

func Seq(n int) []int {
	result := make([]int, n)
	for i := range result {
		result[i] = i
	}
	return result
}

func Factorial(n int) int {
	result := 1
	for i := 2; i <= n; i++ {
		result *= i
	}
	return result
}

func Generate(n, limit int) [][]int {
	if n == 0 {
		return [][]int{{}}
	}
	if n == 1 {
		return [][]int{{0}}
	}

	perm := Seq(n)
	state := make([]int, n)

	capacity := limit
	if capacity <= 0 || n <= 12 {
		capacity = Factorial(min(n, 12))
	}
	result := make([][]int, 0, capacity)
	result = append(result, slices.Clone(perm))

	for i := 0; i < n && (limit <= 0 || len(result) < limit); {
		if state[i] < i {
			if i&1 == 0 {
				perm[0], perm[i] = perm[i], perm[0]
			} else {
				perm[state[i]], perm[i] = perm[i], perm[state[i]]
			}
			result = append(result, slices.Clone(perm))
			state[i]++
			i = 0
		} else {
			state[i] = 0
			i++
		}
	}
	return result
}
