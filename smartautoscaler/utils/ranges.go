package utils

import "math"

// Avg returns the arithmetic mean of data. Returns 0 for empty slice.
func Avg(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	var sum float64
	for _, d := range data {
		sum += d
	}
	return sum / float64(len(data))
}

// StdDev returns the population standard deviation of data.
// Returns 0 for slices with fewer than 2 elements.
func StdDev(data []float64) float64 {
	if len(data) < 2 {
		return 0
	}

	sum := 0.0
	for _, v := range data {
		sum += v
	}
	mean := sum / float64(len(data))

	variance := 0.0
	for _, v := range data {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(data))

	return math.Sqrt(variance)
}
