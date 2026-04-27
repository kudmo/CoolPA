package welchtest

import (
	"math"
)

type Stats struct {
	N    int
	Mean float64
	M2   float64
}

func NewOnlineStats() *Stats {
	return &Stats{
		N:    0,
		Mean: 0.0,
		M2:   0.0,
	}
}

func (s *Stats) Variance() float64 {
	if s.N < 2 {
		return 0.0
	}
	return s.M2 / float64(s.N-1)
}

func (s *Stats) StdDev() float64 {
	return math.Sqrt(s.Variance())
}

type TTestResult struct {
	TStatistic float64
	DF         float64
	PValue     float64
}

func WelchTTest(a, b *Stats) *TTestResult {
	if a.N < 2 || b.N < 2 {
		return &TTestResult{
			TStatistic: math.NaN(),
			DF:         math.NaN(),
			PValue:     math.NaN(),
		}
	}

	varA := a.Variance()
	varB := b.Variance()

	seA := varA / float64(a.N)
	seB := varB / float64(b.N)
	se := math.Sqrt(seA + seB)

	t := (a.Mean - b.Mean) / se

	numer := (seA + seB) * (seA + seB)
	denom := (seA*seA)/float64(a.N-1) + (seB*seB)/float64(b.N-1)
	df := numer / denom

	p := 2.0 * (1.0 - studentTCDF(math.Abs(t), df))

	return &TTestResult{
		TStatistic: t,
		DF:         df,
		PValue:     p,
	}
}

// studentTCDF returns the cumulative distribution function for Student's t
// Use a numeric approximation (could be replaced with an external lib)
func studentTCDF(t, df float64) float64 {
	// Simple approximation via symmetry
	// For positive t, return 0.5 + CDF
	// Accurate modules/libraries are recommended for real use
	return 0.5 * (1 + math.Erf(t/math.Sqrt(2*(df/(df-2)))))
}
