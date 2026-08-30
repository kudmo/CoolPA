package analyzer

import (
	"context"
	"slices"

	"github.com/kudmo/CoolPA/internal/metrics"
	"github.com/kudmo/CoolPA/internal/statistics"

	"github.com/kudmo/CoolPA/utils/welchtest"
)

type Analyzer struct {
	metricsProvider metrics.MetricsRepository
	histStore       *statistics.HistStore
	config          AnalyzerConfig

	previousStatistics map[string]*welchtest.Stats
}

func NewAnalyzer(
	config AnalyzerConfig,
	metricsProvider metrics.MetricsRepository,
	histStore *statistics.HistStore,
) *Analyzer {
	return &Analyzer{
		metricsProvider:    metricsProvider,
		config:             config,
		histStore:          histStore,
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
	a.updateStatistics(ctx, result)

	return result
}

func (a *Analyzer) updateStatistics(ctx context.Context, analysisResult AnalysisResult) {
	if analysisResult.Scale == 1 {
		services, _ := a.metricsProvider.ListServices(ctx)
		for _, svc := range services {
			if a.histStore.GetHistogram(svc) == nil {
				bounds := statistics.LogBounds(float64(a.config.SLO))
				a.histStore.Register(svc, bounds)
			}
			lat_95, _ := a.metricsProvider.GetServiceAverageLatency95Value(ctx, svc)
			a.histStore.GetHistogram(svc).Observe(lat_95, slices.Contains(analysisResult.Services, svc))
		}
	}

	for _, s := range analysisResult.Services {
		new, _ := a.metricsProvider.GetServiceRequestsCountRange(ctx, s, a.config.Window)
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
}
