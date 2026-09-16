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
