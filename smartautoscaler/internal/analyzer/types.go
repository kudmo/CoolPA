package analyzer

// AnalysisResult represents the outcome of an analysis cycle.
// It indicates which services were detected as anomalous and
// the recommended scaling action.
type AnalysisResult struct {
	// Services is the list of service names that were found to be
	// anomalous during the analysis.
	Services []string

	// Scale indicates the type of scaling action to perform:
	//   -1: scale down
	//    0: no change
	//    1: scale up
	Scale int
}

// underutilizationAnalyzeResult holds the result of an internal
// analysis step for a single service, indicating how underutilized
// the service is (Rate) to support scaling down decisions.
type underutilizationAnalyzeResult struct {
	// Service is the name of the service that was analyzed.
	Service string

	// Rate represents the degree of underutilization as a fraction
	// of current allocated resources (e.g., CPU or memory).
	// Values closer to 1 mean severe underutilization.
	Rate float64
}
