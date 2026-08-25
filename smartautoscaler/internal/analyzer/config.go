package analyzer

import "time"

const BETA = 0.8

type AnalyzerConfig struct {
	SLO        float64
	Confidence float64

	WelchOldIntervalBegin time.Duration
	WelchNowIntervalBegin time.Duration
	AnomalyServicesCount  int

	abnormalParams AbnormalParams
}

type AbnormalParams struct {
	Window time.Duration
	SLO    float64
	Alpha  float64
}
