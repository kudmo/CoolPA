package analyzer

type AnalysisResult struct {
	Services []string // List of anomalous services
	Scale    int      // Scale can be -1 (scale down), 0 (no change), or 1 (scale up)s
}

type underutilizationAnalyzeResult struct {
	Service string
	Rate    float64
}
