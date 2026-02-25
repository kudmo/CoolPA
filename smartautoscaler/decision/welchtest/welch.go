package welchtest

import (
	"errors"
	"math"
)

type WelchTestResult struct {
	TStatistic       float64
	DegreesOfFreedom float64
	Mean1            float64
	Mean2            float64
	Var1             float64
	Var2             float64
	N1               float64
	N2               float64
}

func calculateMean(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range samples {
		sum += v
	}
	return sum / float64(len(samples))
}

func calculateVariance(samples []float64, mean float64) float64 {
	if len(samples) < 2 {
		return 0
	}
	sumSquares := 0.0
	for _, v := range samples {
		diff := v - mean
		sumSquares += diff * diff
	}
	return sumSquares / float64(len(samples)-1)
}

// TwoSampleWelch performs Welch's t-test on two independent samples
// Returns t-statistic and approximate degrees of freedom
func TwoSampleWelch(a, b []float64) (WelchTestResult, error) {
	n1 := float64(len(a))
	n2 := float64(len(b))
	if n1 < 2 || n2 < 2 {
		return WelchTestResult{}, errors.New("samples too small for t-test")
	}

	mean1 := calculateMean(a)
	mean2 := calculateMean(b)
	var1 := calculateVariance(a, mean1)
	var2 := calculateVariance(b, mean2)

	// t-statistic
	den := math.Sqrt(var1/n1 + var2/n2)
	if den == 0 {
		return WelchTestResult{}, errors.New("zero variance in samples")
	}
	tStat := (mean1 - mean2) / den

	// degrees of freedom (Welch–Satterthwaite)
	num := math.Pow(var1/n1+var2/n2, 2)
	denom := math.Pow(var1/n1, 2)/(n1-1) + math.Pow(var2/n2, 2)/(n2-1)
	df := num / denom

	return WelchTestResult{
		TStatistic:       tStat,
		DegreesOfFreedom: df,
		Mean1:            mean1,
		Mean2:            mean2,
		Var1:             var1,
		Var2:             var2,
		N1:               n1,
		N2:               n2,
	}, nil
}

// Optionally: calculate a one-sided p-value from the t-distribution
// Example: mean(current) < mean(baseline)
func (r *WelchTestResult) PValueOneSided() float64 {
	// We'll approximate via CDF of Student t
	// For simplicity, we can precompute sorted t values
	// For production, use a library or approximator
	return studentTCDF(r.TStatistic, r.DegreesOfFreedom)
}

// studentTCDF returns the cumulative distribution function for Student's t
// Use a numeric approximation (could be replaced with an external lib)
func studentTCDF(t, df float64) float64 {
	// Simple approximation via symmetry
	// For positive t, return 0.5 + CDF
	// Accurate modules/libraries are recommended for real use
	return 0.5 * (1 + math.Erf(t/math.Sqrt(2*(df/(df-2)))))
}

// TwoSampleWelchTestAlpha returns true if t-statistic is significant
func TwoSampleWelchTestAlpha(a, b []float64, alpha float64) (bool, WelchTestResult, error) {
	res, err := TwoSampleWelch(a, b)
	if err != nil {
		return false, res, err
	}

	// one-sided test: current sample mean < baseline mean
	significant := res.TStatistic < -studentTCrit(res.DegreesOfFreedom, alpha)
	return significant, res, nil
}

func studentTCrit(df, alpha float64) float64 {
	// Basic numeric approximation for critical value
	// For production, use distribution tables or libraries
	return math.Sqrt(df) * (1.0 - alpha) // coarse approximation
}
