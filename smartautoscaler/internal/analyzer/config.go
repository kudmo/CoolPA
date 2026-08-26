package analyzer

import "time"

const BETA = 0.8

type AnalyzerConfig struct {
	SLO        float64
	Confidence float64

	Window               time.Duration
	AnomalyServicesCount int

	Alpha float64
}
