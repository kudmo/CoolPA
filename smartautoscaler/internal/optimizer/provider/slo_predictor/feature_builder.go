package slopredictor

import (
	"context"
	"math"

	"github.com/kudmo/CoolPA/internal/metrics"
	"github.com/kudmo/CoolPA/internal/optimizer/provider/ga/genome"
	"github.com/kudmo/CoolPA/utils"
)

type LatencyDeltaPredictorFeatureBuilder struct {
	config          FitnessConfig
	metricsProvider metrics.MetricsRepository
}

func (b *LatencyDeltaPredictorFeatureBuilder) Build(ctx context.Context, g *genome.ReactionGenome) [][]float64 {
	if g == nil {
		return nil
	}
	out := make([][]float64, 0, len(g.Genes))
	eps := 1e-6

	for _, sg := range g.Genes {
		if sg == nil {
			continue
		}

		svcID := metrics.ServiceID{Namespace: g.Namespace, Name: sg.ServiceName}
		replicas_count, _ := b.metricsProvider.GetServiceReplicasCountValue(ctx, svcID)

		if replicas_count == 0 {
			continue
		}

		containerCpuQuota, _ := b.metricsProvider.GetServiceCpuQuota(ctx, svcID)
		containerMemoryQuota, _ := b.metricsProvider.GetServiceMemoryQuota(ctx, svcID)

		cpuUsageRange, _ := b.metricsProvider.GetServiceCpuUsageRange(ctx, svcID, b.config.Window)
		memUsageRange, _ := b.metricsProvider.GetServiceMemoryUsageRange(ctx, svcID, b.config.Window)

		cpu := utils.Avg(cpuUsageRange)
		memory := utils.Avg(memUsageRange)

		cpuLimit := containerCpuQuota
		memLimit := containerMemoryQuota

		switch sg.ReactionType {
		case genome.HPA:
			replicas_count = math.Max(0, replicas_count+sg.DeltaReplicas)
		case genome.VPA:
			cpuLimit = math.Max(0, cpuLimit+sg.DeltaCPU)
			memLimit = math.Max(0, memLimit+sg.DeltaMemory)
		}

		totalCPULimit := replicas_count * cpuLimit
		totalMemLimit := replicas_count * memLimit

		replicasDelta := sg.DeltaReplicas
		totalCPULimitDelta := (replicas_count+sg.DeltaReplicas)*(cpuLimit+sg.DeltaCPU) - totalCPULimit
		totalMemLimitDelta := (replicas_count+sg.DeltaReplicas)*(memLimit+sg.DeltaMemory) - totalMemLimit

		netRecv, _ := b.metricsProvider.GetServiceNetworkReceiveRange(ctx, svcID, b.config.Window)
		netRecvAvg := utils.Avg(netRecv)
		netTrans, _ := b.metricsProvider.GetServiceNetworkTransmitRange(ctx, svcID, b.config.Window)
		netTransAvg := utils.Avg(netTrans)

		features := make([]float64, 11)

		// 1. cpu_utilization
		features[0] = cpu / (replicas_count*cpuLimit + eps)
		// 2. mem_utilization
		features[1] = memory / (replicas_count*memLimit + eps)
		// 3. cpu_cv
		features[2] = utils.StdDev(cpuUsageRange) / (cpu + eps)
		// 4. memory_cv
		features[3] = utils.StdDev(memUsageRange) / (memory + eps)
		// 5. net_recv_trans_ratio
		features[4] = netRecvAvg / (netTransAvg + eps)
		// 6. replicas
		features[5] = replicas_count
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
