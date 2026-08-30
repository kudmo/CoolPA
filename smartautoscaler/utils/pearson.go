package utils

import "math"

// Pearson calculates the Pearson correlation coefficient between
// two slices of equal length. Returns 0 if slices are empty,
// have different lengths, or have zero variance.
func Pearson(x, y []float64) float64 {
	n := len(x)
	if n == 0 || len(y) != n {
		return 0
	}

	var sumX, sumY float64
	for i := range n {
		sumX += x[i]
		sumY += y[i]
	}

	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	var num, denX, denY float64
	for i := range n {
		dx := x[i] - meanX
		dy := y[i] - meanY
		num += dx * dy
		denX += dx * dx
		denY += dy * dy
	}

	if denX == 0 || denY == 0 {
		return 0
	}

	return num / math.Sqrt(denX*denY)
}
