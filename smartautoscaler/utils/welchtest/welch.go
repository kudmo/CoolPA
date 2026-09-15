// Package welchtest provides statistical
// utilities for performing Welch's t-test.
package welchtest

import (
	"math"
)

// Stats represents running statistics for a sample population.
// It uses Welford's online algorithm to maintain mean and
// variance without storing all data points.
//
// The struct is safe for sequential use but not for concurrent
// access. External synchronization is required if used from
// multiple goroutines.
type Stats struct {
	// N is the number of observations in the sample
	N int

	// Mean is the current arithmetic mean of all observations
	Mean float64

	// M2 is the sum of squared differences from the mean,
	// used for calculating variance
	M2 float64
}

// NewOnlineStats creates a new Stats instance initialized
// with zero observations and zero mean.
//
// The returned Stats is ready for immediate use with the
// Add method (not included in this example) or direct
// manipulation of fields.
func NewOnlineStats() *Stats {
	return &Stats{
		N:    0,
		Mean: 0.0,
		M2:   0.0,
	}
}

// Variance returns the sample variance (using N-1 degrees
// of freedom) of the observations.
//
// Returns 0.0 if fewer than 2 observations have been recorded,
// as variance is undefined for samples with less than 2 points.
func (s *Stats) Variance() float64 {
	if s.N < 2 {
		return 0.0
	}
	return s.M2 / float64(s.N-1)
}

// StdDev returns the sample standard deviation, which is
// the square root of the variance.
//
// Returns 0.0 if fewer than 2 observations have been recorded.
func (s *Stats) StdDev() float64 {
	return math.Sqrt(s.Variance())
}

// TTestResult contains the results of a Welch's t-test
// comparing two independent samples.
type TTestResult struct {
	// TStatistic is the calculated t-value. A larger absolute
	// value indicates a greater difference between the means
	// relative to the variability in the data.
	TStatistic float64

	// DF represents the degrees of freedom calculated using
	// the Welch–Satterthwaite equation, which accounts for
	// unequal variances between the two samples.
	DF float64

	// PValue is the two-tailed probability of observing a
	// t-statistic at least as extreme as the one calculated,
	// assuming the null hypothesis (no difference in means).
	// Values below the significance level (typically 0.05)
	// indicate a statistically significant difference.
	PValue float64
}

// WelchTTest performs Welch's t-test to compare the means of
// two independent samples with potentially unequal variances
// and sample sizes.
//
// The test is appropriate when:
//   - The two samples are independent
//   - The data is approximately normally distributed
//   - The samples may have different variances
//   - The samples may have different sizes
//
// Parameters:
//   - a: Statistics for the first sample
//   - b: Statistics for the second sample
//
// Returns:
//   - A pointer to TTestResult containing the test statistics,
//     or NaN values if either sample has fewer than 2 observations
//
// Example:
//
//	stats1 := NewOnlineStats()
//	stats2 := NewOnlineStats()
//	// ... add data to stats1 and stats2 ...
//	result := WelchTTest(stats1, stats2)
//	if result.PValue < 0.05 {
//	    fmt.Println("Significant difference detected")
//	}
func WelchTTest(a, b *Stats) *TTestResult {
	// Check if both samples have sufficient data for the test
	if a.N < 2 || b.N < 2 {
		return &TTestResult{
			TStatistic: math.NaN(),
			DF:         math.NaN(),
			PValue:     math.NaN(),
		}
	}

	// Calculate variances for both samples
	varA := a.Variance()
	varB := b.Variance()

	// Calculate standard errors for both samples
	seA := varA / float64(a.N)
	seB := varB / float64(b.N)

	// Calculate the standard error of the difference
	se := math.Sqrt(seA + seB)

	// Calculate the t-statistic
	t := (a.Mean - b.Mean) / se

	// Calculate degrees of freedom using Welch–Satterthwaite equation
	numer := (seA + seB) * (seA + seB)
	denom := (seA*seA)/float64(a.N-1) + (seB*seB)/float64(b.N-1)
	df := numer / denom

	// Calculate two-tailed p-value
	p := 2.0 * (1.0 - studentTCDF(math.Abs(t), df))

	return &TTestResult{
		TStatistic: t,
		DF:         df,
		PValue:     p,
	}
}

// studentTCDF returns the cumulative distribution function (CDF)
// for Student's t-distribution.
//
// Note: This implementation uses a simplified approximation
// based on the error function and is not accurate for all
// degrees of freedom. For production use, it is recommended
// to replace this with a more accurate implementation from
// a statistical library.
//
// Parameters:
//   - t: The t-value (should be non-negative)
//   - df: Degrees of freedom (must be > 2 for this approximation)
//
// Returns:
//   - The probability that a random variable following the
//     t-distribution with df degrees of freedom is less than
//     or equal to t
func studentTCDF(t, df float64) float64 {
	// Simple approximation via symmetry
	// For positive t, return 0.5 + CDF
	// Accurate modules/libraries are recommended for real use
	return 0.5 * (1 + math.Erf(t/math.Sqrt(2*(df/(df-2)))))
}
