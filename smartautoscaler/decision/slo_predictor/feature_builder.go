package slopredictor

import (
	"math"

	"github.com/kudmo/CoolPA/decision/ga/genome"
	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/CoolPA/storage/metrics"
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

		container_cpu_quota, _, _ := b.Store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.CPUQuota)
		container_memory_limit, _, _ := b.Store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.MemoryLimit)
		replicas_count := float64(len(pods))
		istio_requests_total := node.RequestCount.Avg()
		// slo_threshold := float64(200)

		switch sg.ReactionType {
		case genome.HPA:
			replicas_count = math.Max(0, math.Round(float64(replicas_count)+sg.DeltaReplicas))
		case genome.VPA_CPU:
			container_cpu_quota = container_cpu_quota + sg.DeltaCPU
			// container_memory_limit = container_memory_limit * (1.0 + sg.DeltaMem)
		}

		eps := 1e-6
		if istio_requests_total < eps {
			istio_requests_total = eps
		}
		total_cpu_limit := replicas_count * container_cpu_quota
		total_memory_limit := replicas_count * container_memory_limit

		rps_per_replica := istio_requests_total / replicas_count
		cpu_per_rps := container_cpu_quota / istio_requests_total
		memory_per_rps := container_memory_limit / istio_requests_total

		cpuUsageWindow, _, _ := b.Store.ResourceMetrics.GetPodMetricWindow(svc, pods[0], metrics.CPUUsage)
		memUsageWindow, _, _ := b.Store.ResourceMetrics.GetPodMetricWindow(svc, pods[0], metrics.MemoryUsage)
		avg_cpu_usage := cpuUsageWindow.Avg()
		avg_mem_usage := memUsageWindow.Avg()

		cpu_utilization := avg_cpu_usage / (total_cpu_limit + eps)
		mem_utilization := avg_mem_usage / (total_memory_limit + eps)

		rps_cv := node.RequestCount.StdDev() / (node.RequestCount.Avg() + eps)
		cpu_cv := cpuUsageWindow.StdDev() / (cpuUsageWindow.Avg() + eps)
		rps_values := node.RequestCount.Values()
		cpu_values := cpuUsageWindow.Values()
		mem_values := memUsageWindow.Values()

		cpu_rps_corr := 0.0
		if len(rps_values) > 1 && len(cpu_values) > 1 && len(rps_values) == len(cpu_values) {
			cpu_rps_corr = pearsonCorrelation(rps_values, cpu_values)
		}

		mem_rps_corr := 0.0
		if len(rps_values) > 1 && len(mem_values) > 1 && len(rps_values) == len(mem_values) {
			mem_rps_corr = pearsonCorrelation(rps_values, mem_values)
		}

		out = append(out, []float64{
			total_cpu_limit,
			total_memory_limit,
			rps_per_replica,
			cpu_per_rps,
			memory_per_rps,
			cpu_utilization,
			mem_utilization,
			rps_cv,
			cpu_cv,
			cpu_rps_corr,
			mem_rps_corr,
		})
	}
	return out
}

func pearsonCorrelation(x, y []float64) float64 {
	n := len(x)
	if n == 0 || len(y) != n {
		return 0
	}

	var sumX, sumY float64
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
	}

	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	var num, denX, denY float64
	for i := 0; i < n; i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		num += dx * dy
		denX += dx * dx
		denY += dy * dy
	}

	if denX == 0 || denY == 0 {
		return 0
	}

	return num / math.Sqrt(denX*denY)
}
