package analyzer

import (
	"log/slog"
	"sort"
	"time"

	sloviolation "github.com/kudmo/CoolPA/analyzer/slo_violation"
	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/CoolPA/storage/metrics"
	"github.com/kudmo/CoolPA/storage/quotas"
	"github.com/kudmo/CoolPA/utils/welchtest"
	"github.com/kudmo/toporank/api"
	"github.com/kudmo/toporank/types"
)

// TODO: возможно в конфиги
const BETA = 0.8
const ANOMALY_SERVICES_COUNT = 3
const UNDERUTILIZATION_SERVICES_COUNT = 3

type AnalyzerConfig struct {
	SLO        float64
	Confidence float64

	WelchOldIntervalBegin time.Duration
	WelchNowIntervalBegin time.Duration
}

type Analyzer struct {
	store  *storage.Storage
	config AnalyzerConfig

	previousStatistics map[string]*welchtest.Stats
}

func NewAnalyzer(config AnalyzerConfig, store *storage.Storage) *Analyzer {
	return &Analyzer{
		store:              store,
		config:             config,
		previousStatistics: make(map[string]*welchtest.Stats),
	}
}

type AnalysisResult struct {
	Services []string // List of anomalous services
	Scale    int      // Scale can be -1 (scale down), 0 (no change), or 1 (scale up)s
}

func (a *Analyzer) analyzeWithSLOViolation() []string {
	params := sloviolation.AbnormalParams{
		Window: 1 * time.Minute,
		SLO:    a.config.SLO,
		Alpha:  0.2,
	}
	result := make([]string, 0)

	graph, err := sloviolation.BuildAbnormalCorrelationGraph(time.Now(), params, a.store)
	if err != nil {
		slog.Error("Failed to build abnormal correlation graph", "error", err)
	} else {
		if graph == nil {
			return result
		}
	}
	anomalys := api.RunTopoRank(graph, types.DefaultConfig())
	sort.Slice(anomalys, func(i, j int) bool {
		return anomalys[i].Rank > anomalys[j].Rank
	})

	// КОСТЫЛЬ, ЧТОБЫ НЕ БЫЛО LOADGEN
	for i := 0; i < len(anomalys) && len(result) < ANOMALY_SERVICES_COUNT; i++ {
		if anomalys[i].ID != "istio-ingressgateway" {
			result = append(result, anomalys[i].ID)
		}
	}

	for _, service := range anomalys {
		slog.Info("Calculated anomaly", "service", service.ID, "value", service.Rank)
	}
	return result
}

type underutilizationAnalyzeResult struct {
	Service string
	Rate    float64
}

func (a *Analyzer) analyzeRPSlowing() []underutilizationAnalyzeResult {
	anomalys := make([]underutilizationAnalyzeResult, 0)

	for _, service := range a.store.ResourceMetrics.GetServices() {
		n, _ := a.store.Graph.GetNode(service)
		if n == nil {
			slog.Error("Unexisting service", "service name", service)
			continue
		}
		time_now := time.Now()
		time_now_begin := time_now.Add(-a.config.WelchNowIntervalBegin)

		new := n.RequestCount.SeriesRange(time_now_begin, time_now)

		newStats := welchtest.NewOnlineStats()
		newStats.N = len(new)
		for _, i := range new {
			newStats.Mean += i
		}
		newStats.Mean = newStats.Mean / float64(newStats.N)

		for _, i := range new {
			newStats.M2 += (i - newStats.Mean) * (i - newStats.Mean)
		}
		oldStats := welchtest.NewOnlineStats()
		if _, ok := a.previousStatistics[service]; !ok {
			continue
		}
		oldStats.N = a.previousStatistics[service].N
		oldStats.Mean = a.previousStatistics[service].Mean * BETA
		oldStats.M2 = a.previousStatistics[service].M2 * BETA * BETA

		welch_result := welchtest.WelchTTest(newStats, oldStats)
		if welch_result.TStatistic < 0 && welch_result.PValue <= a.config.Confidence {
			service_cpu_limit, _, _ := a.store.ResourceMetrics.GetServiceMetricAvgHead(service, metrics.CPUQuota)
			service_mem_limit, _, _ := a.store.ResourceMetrics.GetServiceMetricAvgHead(service, metrics.MemoryLimit)
			replicas := float64(len(a.store.ResourceMetrics.GetServicePods(service)))

			// Already minimal configuration
			if /*a.store.Limits.ServiceLimits[quotas.ServiceMinCpu] >= int64(service_cpu_limit) &&
			a.store.Limits.ServiceLimits[quotas.ServiceMinMem] >= int64(service_mem_limit) &&*/
			1 >= replicas {
				continue
			}

			service_cpu_utilization, _, _ := a.store.ResourceMetrics.GetServiceMetricAvgRange(service, metrics.CPUUsage, time_now_begin, time_now)
			service_cpu_underutilization_total := (service_cpu_limit - service_cpu_utilization) * replicas
			service_cpu_underutilization_total_percent := service_cpu_underutilization_total / float64(a.store.Limits.NamespaceLimits[quotas.NamespaceMaxCpu])

			service_mem_utilization, _, _ := a.store.ResourceMetrics.GetServiceMetricAvgRange(service, metrics.MemoryUsage, time_now_begin, time_now)
			service_mem_underutilization_total := (service_mem_limit - service_mem_utilization) * replicas
			service_mem_underutilization_total_percent := service_mem_underutilization_total / float64(a.store.Limits.NamespaceLimits[quotas.NamespaceMaxMem])

			// Choose services with maximum underutilization percent
			anomalys = append(anomalys, underutilizationAnalyzeResult{service, max(service_cpu_underutilization_total_percent, service_mem_underutilization_total_percent)})
		}
	}

	return anomalys
}

func (a *Analyzer) analyzeUnderutilization() []string {
	result := make([]string, 0)

	anomalys := a.analyzeRPSlowing()

	sort.Slice(anomalys, func(i, j int) bool {
		return anomalys[i].Rate > anomalys[j].Rate
	})

	for i := 0; i < len(anomalys) && len(result) < UNDERUTILIZATION_SERVICES_COUNT; i++ {
		result = append(result, anomalys[i].Service)
	}

	slog.Info("Calculated underutilization", "services", result)
	return result
}

func (a *Analyzer) Analyze() AnalysisResult {
	result := AnalysisResult{
		Services: []string{},
		Scale:    0,
	}
	bottlenecks := a.analyzeWithSLOViolation()

	if len(bottlenecks) > 0 {
		result.Services = bottlenecks
		result.Scale = 1
	} else {
		underutilized := a.analyzeUnderutilization()
		if len(underutilized) > 0 {
			result.Services = underutilized
			result.Scale = -1
		}
	}

	time_now := time.Now()
	time_now_begin := time_now.Add(-a.config.WelchNowIntervalBegin)

	for _, s := range result.Services {
		n, _ := a.store.Graph.GetNode(s)
		new := n.RequestCount.SeriesRange(time_now_begin, time_now)
		newStats := welchtest.NewOnlineStats()
		newStats.N = len(new)
		for _, i := range new {
			newStats.Mean += i
		}
		newStats.Mean = newStats.Mean / float64(newStats.N)

		for _, i := range new {
			newStats.M2 += (i - newStats.Mean) * (i - newStats.Mean)
		}
		if a.previousStatistics[s] != nil {
			slog.Warn("stats UPDATE", "service", s, "old", a.previousStatistics[s].Mean, "new", newStats.Mean)
		} else {
			slog.Warn("stats Create", "service", s, "new", newStats.Mean)

		}
		a.previousStatistics[s] = newStats
	}

	return result
}
