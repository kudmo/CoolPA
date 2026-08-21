package analyzer

import (
	"context"
	"sort"
	"time"

	"github.com/kudmo/CoolPA/logger"
	"github.com/kudmo/CoolPA/utils"
	"github.com/kudmo/CoolPA/utils/welchtest"
)

func (a *Analyzer) analyzeRPSlowing(ctx context.Context) []underutilizationAnalyzeResult {
	anomalys := make([]underutilizationAnalyzeResult, 0)
	services, _ := a.metricsProvider.ListServices(ctx)
	for _, service := range services {
		time_now := time.Now()
		time_now_begin := time_now.Add(-a.config.WelchNowIntervalBegin)

		new, _ := a.metricsProvider.GetServiceRequestsCountRange(ctx, service.Name, time_now_begin, time_now)

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
		if _, ok := a.previousStatistics[service.Name]; !ok {
			continue
		}
		oldStats.N = a.previousStatistics[service.Name].N
		oldStats.Mean = a.previousStatistics[service.Name].Mean * BETA
		oldStats.M2 = a.previousStatistics[service.Name].M2 * BETA * BETA

		welch_result := welchtest.WelchTTest(newStats, oldStats)
		if welch_result.TStatistic < 0 && welch_result.PValue <= a.config.Confidence {
			service_cpu_limit, _ := a.metricsProvider.GetServiceCpuQuotaValue(ctx, service.Name)
			service_mem_limit, _ := a.metricsProvider.GetServiceMemoryQuotaValue(ctx, service.Name)
			replicas, _ := a.metricsProvider.GetServiceReplicasCountValue(ctx, service.Name)

			// Already minimal configuration
			if /*a.store.Limits.ServiceLimits[quotas.ServiceMinCpu] >= int64(service_cpu_limit) &&
			a.store.Limits.ServiceLimits[quotas.ServiceMinMem] >= int64(service_mem_limit) &&*/
			1 >= replicas {
				continue
			}

			service_cpu_utilization_range, _ := a.metricsProvider.GetServiceCpuUsageRange(ctx, service.Name, time_now_begin, time_now)
			service_cpu_utilization := utils.Avg(service_cpu_utilization_range)
			service_cpu_underutilization_total := (service_cpu_limit - service_cpu_utilization) * replicas
			total_cpu_limit, _ := a.metricsProvider.GetGlobalTotalCpuLimit(ctx)
			service_cpu_underutilization_total_percent := service_cpu_underutilization_total / total_cpu_limit

			service_mem_utilization_range, _ := a.metricsProvider.GetServiceMemoryUsageRange(ctx, service.Name, time_now_begin, time_now)
			service_mem_utilization := utils.Avg(service_mem_utilization_range)
			service_mem_underutilization_total := (service_mem_limit - service_mem_utilization) * replicas
			total_mem_limit, _ := a.metricsProvider.GetGlobalTotalMemoryLimit(ctx)
			service_mem_underutilization_total_percent := service_mem_underutilization_total / total_mem_limit

			// Choose services with maximum underutilization percent
			anomalys = append(anomalys, underutilizationAnalyzeResult{service.Name, max(service_cpu_underutilization_total_percent, service_mem_underutilization_total_percent)})
		}
	}

	return anomalys
}

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
