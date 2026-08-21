package analyzer

import (
	"context"
	"time"

	"github.com/kudmo/CoolPA/internal/analyzer/interfaces"

	"github.com/kudmo/CoolPA/utils/welchtest"
)

type Analyzer struct {
	metricsProvider interfaces.MetricsProvider
	config          AnalyzerConfig

	previousStatistics map[string]*welchtest.Stats
}

func NewAnalyzer(config AnalyzerConfig, metricsProvider interfaces.MetricsProvider) *Analyzer {
	return &Analyzer{
		metricsProvider:    metricsProvider,
		config:             config,
		previousStatistics: make(map[string]*welchtest.Stats),
	}
}

// Analyze runs analysis to detect anomalous or underutilized services and
// returns an AnalysisResult describing affected services and scaling action.
func (a *Analyzer) Analyze(ctx context.Context) AnalysisResult {
	result := AnalysisResult{
		Services: []string{},
		Scale:    0,
	}
	bottlenecks := a.analyzeWithSLOViolation(ctx)

	if len(bottlenecks) > 0 {
		result.Services = bottlenecks
		result.Scale = 1
	} else {
		underutilized := a.analyzeUnderutilization(ctx)
		if len(underutilized) > 0 {
			result.Services = underutilized
			result.Scale = -1
		}
	}

	time_now := time.Now()
	time_now_begin := time_now.Add(-a.config.WelchNowIntervalBegin)

	for _, s := range result.Services {
		new, _ := a.metricsProvider.GetServiceRequestsCountRange(ctx, s, time_now_begin, time_now)
		newStats := welchtest.NewOnlineStats()
		newStats.N = len(new)
		for _, i := range new {
			newStats.Mean += i
		}
		newStats.Mean = newStats.Mean / float64(newStats.N)

		for _, i := range new {
			newStats.M2 += (i - newStats.Mean) * (i - newStats.Mean)
		}
		a.previousStatistics[s] = newStats
	}

	return result
}
