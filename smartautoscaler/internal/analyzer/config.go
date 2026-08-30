package analyzer

import "time"

// BETA is a decay factor applied to previous request statistics
// when comparing current and past RPS distributions. Values less
// than 1 account for expected gradual decrease in traffic over time.
const BETA = 0.8

// AnalyzerConfig holds parameters that control the behavior of the
// Analyzer, including thresholds for anomaly detection and the
// maximum number of services to recommend for scaling.
type AnalyzerConfig struct {
	// SLO is the target 95th percentile latency in milliseconds.
	// Calls exceeding this threshold are considered SLO violations.
	SLO float64

	// Confidence is the statistical significance level (e.g., 0.05)
	// used in Welch's t-test to decide whether an RPS decrease is
	// significant.
	Confidence float64

	// Window is the time duration used to query metric ranges for
	// analysis.
	Window time.Duration

	// AnomalyServicesCount limits the number of anomalous services
	// returned in an AnalysisResult.
	AnomalyServicesCount int

	// Alpha is a tolerance factor (0..1) that relaxes the SLO
	// threshold. A call is considered abnormal if its latency
	// exceeds SLO * (1 - Alpha).
	Alpha float64
}
