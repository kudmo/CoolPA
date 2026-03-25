package slopredictor

import (
	"math"

	"github.com/kudmo/CoolPA/decision/ga/genome"
	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/CoolPA/storage/metrics"
	"github.com/kudmo/CoolPA/utils"
)

type SLOPredictorFeatureBuilder struct {
	Store *storage.Storage
}

// container_cpu_quota
// container_memory_limit
// replicas_count
// istio_requests_total
// slo_threshold
// rps_per_replica = istio_requests_total / replicas_count
// cpu_per_rps = container_cpu_quota / istio_requests_total
// memory_per_rps = container_memory_limit / istio_requests_total
// total_cpu_limit = replicas_count * container_cpu_quota
// total_memory_limit = replicas_count * container_memory_limit
func (b *SLOPredictorFeatureBuilder) Build(g *genome.ReactionGenome) [][]float64 {
	if g == nil {
		return nil
	}
	out := make([][]float64, 0, len(g.Genes))
	for _, sg := range g.Genes {
		if sg == nil {
			continue
		}
		svc := sg.ServiceName
		node, _ := b.Store.Graph.GetNode(svc)
		pods := b.Store.ResourceMetrics.GetServicePods(svc)

		containerCPUQuota, _, _ := b.Store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.CPUQuota)
		containerMemoryLimit, _, _ := b.Store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.MemoryLimit)
		replicasCount := float64(len(pods))
		istioRequestsTotal := node.RequestCount.Avg()

		switch sg.ReactionType {
		case genome.HPA:
			replicasCount = math.Max(0, math.Round(float64(replicasCount)+sg.DeltaReplicas))
		case genome.VPA:
			containerCPUQuota = math.Max(0, containerCPUQuota+sg.DeltaCPU)
			containerMemoryLimit = math.Max(0, containerMemoryLimit+sg.DeltaMemory)
		}

		eps := 1e-6
		if istioRequestsTotal < eps {
			istioRequestsTotal = eps
		}

		totalCPULimit := replicasCount * containerCPUQuota
		totalMemoryLimit := replicasCount * containerMemoryLimit

		rpsPerReplica := istioRequestsTotal / replicasCount
		cpuPerRps := containerCPUQuota / istioRequestsTotal
		memoryPerRps := containerMemoryLimit / istioRequestsTotal

		cpuUsageWindow, _, _ := b.Store.ResourceMetrics.GetPodMetricWindow(svc, pods[0], metrics.CPUUsage)
		memUsageWindow, _, _ := b.Store.ResourceMetrics.GetPodMetricWindow(svc, pods[0], metrics.MemoryUsage)
		avgCPUUsage := cpuUsageWindow.Avg()
		avgMemUsage := memUsageWindow.Avg()

		cpuUtilization := avgCPUUsage / (totalCPULimit + eps)
		memUtilization := avgMemUsage / (totalMemoryLimit + eps)

		rpsCV := node.RequestCount.StdDev() / (node.RequestCount.Avg() + eps)
		cpuCV := cpuUsageWindow.StdDev() / (cpuUsageWindow.Avg() + eps)

		rpsValues := node.RequestCount.Values()
		cpuValues := cpuUsageWindow.Values()
		memValues := memUsageWindow.Values()

		cpuRpsCorr := 0.0
		if len(rpsValues) > 1 && len(cpuValues) > 1 && len(rpsValues) == len(cpuValues) {
			cpuRpsCorr = utils.Pearson(rpsValues, cpuValues)
		}

		memRpsCorr := 0.0
		if len(rpsValues) > 1 && len(memValues) > 1 && len(rpsValues) == len(memValues) {
			memRpsCorr = utils.Pearson(rpsValues, memValues)
		}

		out = append(out, []float64{
			totalCPULimit,
			totalMemoryLimit,
			rpsPerReplica,
			cpuPerRps,
			memoryPerRps,
			cpuUtilization,
			memUtilization,
			rpsCV,
			cpuCV,
			cpuRpsCorr,
			memRpsCorr,
		})
	}
	return out
}
