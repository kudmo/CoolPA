// Package analyzer implements the core analysis logic for the autoscaler.
// It detects anomalous services that violate their SLO (scale up) or
// underutilized services (scale down) using statistical tests on
// request metrics and latency histograms.
package analyzer

import (
	"context"
	"slices"

	"github.com/kudmo/CoolPA/internal/metrics"
	"github.com/kudmo/CoolPA/internal/statistics"

	"github.com/kudmo/CoolPA/utils/welchtest"
)

// Analyzer performs periodic analysis of service metrics to decide
// whether scaling actions are needed. It uses a metrics provider to
// fetch current and historical data, a histogram store for latency
// distributions, and maintains previous request statistics for
// comparison over time.
type Analyzer struct {
	metricsProvider metrics.MetricsRepository
	histStore       *statistics.HistStore
	config          AnalyzerConfig

	// previousStatistics stores the request rate statistics from the
	// previous analysis cycle for each service, used as a baseline
	// for detecting changes.
	previousStatistics map[string]*welchtest.Stats
}

// NewAnalyzer creates a new Analyzer with the given configuration,
// metrics provider, and histogram store.
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

// Analyze runs a single analysis cycle and returns the result.
// It first checks for services that violate their SLO (bottlenecks),
// which triggers a scale-up recommendation. If no bottlenecks are
// found, it checks for underutilized services, which triggers a
// scale-down recommendation. The result contains the list of affected
// services and the scaling direction (1, 0, or -1).
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

// updateStatistics refreshes the internal histograms for latency and
// the previous request statistics for services that were part of the
// analysis result. If a scale-up is recommended, it ensures every
// service has a histogram registered and observes its current 95th
// percentile latency. For every service in the result, it updates
// the previous request count statistics using the request rate over
// the configured time window.
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
