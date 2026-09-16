package engine

import (
	"sort"
)

// TopNIndices returns the indices of the top N values in scores (descending order).
func TopNIndices(scores []float64, n int) []int {
	if n <= 0 || len(scores) == 0 {
		return nil
	}
	type iv struct {
		i int
		v float64
	}
	arr := make([]iv, 0, len(scores))
	for i, v := range scores {
		arr = append(arr, iv{i: i, v: v})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].v > arr[j].v })
	if n > len(arr) {
		n = len(arr)
	}
	out := make([]int, n)
	for i := 0; i < n; i++ {
		out[i] = arr[i].i
	}
	return out
}
