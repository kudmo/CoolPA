package slopredictor

import (
	"math"

	"github.com/kudmo/CoolPA/decision/ga/genome"
	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/CoolPA/storage/metrics"
)

type LatencyDeltaPredictorFeatureBuilder struct {
	Store *storage.Storage
}

func (b *LatencyDeltaPredictorFeatureBuilder) Build(g *genome.ReactionGenome) [][]float64 {
	if g == nil {
		return nil
	}
	out := make([][]float64, 0, len(g.Genes))
	eps := 1e-6

	for _, sg := range g.Genes {
		if sg == nil {
			continue
		}

		svc := sg.ServiceName
		pods := b.Store.ResourceMetrics.GetServicePods(svc)

		if len(pods) == 0 {
			continue
		}

		containerCPUQuota, _, _ := b.Store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.CPUQuota)
		containerMemoryLimit, _, _ := b.Store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.MemoryLimit)
		replicas := float64(len(pods))

		cpuUsageWindow, _, _ := b.Store.ResourceMetrics.GetPodMetricWindow(svc, pods[0], metrics.CPUUsage)
		memUsageWindow, _, _ := b.Store.ResourceMetrics.GetPodMetricWindow(svc, pods[0], metrics.MemoryUsage)

		cpu := cpuUsageWindow.Avg()
		memory := memUsageWindow.Avg()

		cpuLimit := containerCPUQuota
		memLimit := containerMemoryLimit

		switch sg.ReactionType {
		case genome.HPA:
			replicas = math.Max(0, replicas+sg.DeltaReplicas)
		case genome.VPA:
			cpuLimit = math.Max(0, cpuLimit+sg.DeltaCPU)
			memLimit = math.Max(0, memLimit+sg.DeltaMemory)
		}

		totalCPULimit := replicas * cpuLimit
		totalMemLimit := replicas * memLimit

		replicasDelta := sg.DeltaReplicas
		totalCPULimitDelta := (replicas+sg.DeltaReplicas)*(cpuLimit+sg.DeltaCPU) - totalCPULimit
		totalMemLimitDelta := (replicas+sg.DeltaReplicas)*(memLimit+sg.DeltaMemory) - totalMemLimit

		netRecv, _, _ := b.Store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.NetworkReceive)
		netTrans, _, _ := b.Store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.NetworkTransmit)

		features := make([]float64, 11)

		// 1. cpu_utilization
		features[0] = cpu / (replicas*cpuLimit + eps)
		// 2. mem_utilization
		features[1] = memory / (replicas*memLimit + eps)
		// 3. cpu_cv
		features[2] = cpuUsageWindow.StdDev() / (cpu + eps)
		// 4. memory_cv
		features[3] = memUsageWindow.StdDev() / (memory + eps)
		// 5. net_recv_trans_ratio
		features[4] = netRecv / (netTrans + eps)
		// 6. replicas
		features[5] = replicas
		// 7. total_cpu_limit
		features[6] = totalCPULimit
		// 8. total_mem_limit
		features[7] = totalMemLimit
		// 9. replicas_delta
		features[8] = replicasDelta
		// 10. total_cpu_limit_delta
		features[9] = totalCPULimitDelta
		// 11. total_mem_limit_delta
		features[10] = totalMemLimitDelta

		out = append(out, features)
	}

	return out
}
