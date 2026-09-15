package analyzer

import (
	"context"
	"sort"

	"github.com/kudmo/CoolPA/logger"
	"github.com/kudmo/CoolPA/utils"
	"github.com/kudmo/CoolPA/utils/welchtest"
)

// analyzeRPSlowing detects services whose request rate (RPS) has
// significantly decreased compared to the previous analysis window,
// using Welch's t-test. For each underutilized service, it calculates
// the percentage of unused CPU or memory relative to the cluster total
// and returns a list of candidates sorted by that percentage.
func (a *Analyzer) analyzeRPSlowing(ctx context.Context) []underutilizationAnalyzeResult {
	anomalys := make([]underutilizationAnalyzeResult, 0)
	services, _ := a.metricsProvider.ListServices(ctx)
	for _, service := range services {
		// Build current statistics from the recent RPS range.
		new, _ := a.metricsProvider.GetServiceRequestsCountRange(ctx, service, a.config.Window)

		newStats := welchtest.NewOnlineStats()
		newStats.N = len(new)
		for _, i := range new {
			newStats.Mean += i
		}
		newStats.Mean = newStats.Mean / float64(newStats.N)

		for _, i := range new {
			newStats.M2 += (i - newStats.Mean) * (i - newStats.Mean)
		}

		// Skip services without a previous statistics baseline.
		oldStats := welchtest.NewOnlineStats()
		if _, ok := a.previousStatistics[service]; !ok {
			continue
		}
		// Apply BETA factor to previous statistics to model expected decay.
		oldStats.N = a.previousStatistics[service].N
		oldStats.Mean = a.previousStatistics[service].Mean * BETA
		oldStats.M2 = a.previousStatistics[service].M2 * BETA * BETA

		welch_result := welchtest.WelchTTest(newStats, oldStats)
		if welch_result.TStatistic < 0 && welch_result.PValue <= a.config.Confidence {
			service_cpu_limit, _ := a.metricsProvider.GetServiceCpuQuota(ctx, service)
			service_mem_limit, _ := a.metricsProvider.GetServiceMemoryQuota(ctx, service)
			replicas, _ := a.metricsProvider.GetServiceReplicasCountValue(ctx, service)

			// Skip services already at minimum replicas (cannot scale down further).
			if 1 >= replicas {
				continue
			}

			// Calculate underutilization as percentage of total cluster resources.
			service_cpu_utilization_range, _ := a.metricsProvider.GetServiceCpuUsageRange(ctx, service, a.config.Window)
			service_cpu_utilization := utils.Avg(service_cpu_utilization_range)
			service_cpu_underutilization_total := (service_cpu_limit - service_cpu_utilization) * replicas
			total_cpu_limit, _ := a.metricsProvider.GetGlobalTotalCpuLimit(ctx)
			service_cpu_underutilization_total_percent := service_cpu_underutilization_total / total_cpu_limit

			service_mem_utilization_range, _ := a.metricsProvider.GetServiceMemoryUsageRange(ctx, service, a.config.Window)
			service_mem_utilization := utils.Avg(service_mem_utilization_range)
			service_mem_underutilization_total := (service_mem_limit - service_mem_utilization) * replicas
			total_mem_limit, _ := a.metricsProvider.GetGlobalTotalMemoryLimit(ctx)
			service_mem_underutilization_total_percent := service_mem_underutilization_total / total_mem_limit

			// Use the larger underutilization percentage as the rate.
			anomalys = append(anomalys, underutilizationAnalyzeResult{
				Service: service,
				Rate:    max(service_cpu_underutilization_total_percent, service_mem_underutilization_total_percent),
			})
		}
	}

	return anomalys
}

// analyzeUnderutilization orchestrates the detection of underutilized
// services. It calls analyzeRPSlowing, sorts the results by descending
// underutilization rate, and returns up to AnomalyServicesCount service
// names recommended for scaling down.
func (a *Analyzer) analyzeUnderutilization(ctx context.Context) []string {
	result := make([]string, 0)

	anomalys := a.analyzeRPSlowing(ctx)

	sort.Slice(anomalys, func(i, j int) bool {
		return anomalys[i].Rate > anomalys[j].Rate
	})

	for i := 0; i < len(anomalys) && len(result) < a.config.AnomalyServicesCount; i++ {
		result = append(result, anomalys[i].Service)
	}

	logger.Info("analyzer", "calculated underutilization", "services", result)
	return result
}
