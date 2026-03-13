package fitness

import (
	"math"

	"github.com/kudmo/CoolPA/decision/ga/genome"
	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/CoolPA/storage/metrics"
)

// FeatureBuilder converts genomes into per-service feature vectors suitable for predictors.
type FeatureBuilder interface {
	Build(genome *genome.ReactionGenome) [][]float64
}

type DefaultFeatureBuilder struct {
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
func (b *DefaultFeatureBuilder) Build(g *genome.ReactionGenome) [][]float64 {
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

		container_cpu_quota, _, _ := b.Store.ResourceMetrics.GetPodMetricHeadValue(svc, b.Store.ServicePods[svc][0], metrics.CPUQuota)
		container_memory_limit, _, _ := b.Store.ResourceMetrics.GetPodMetricHeadValue(svc, b.Store.ServicePods[svc][0], metrics.MemoryLimit)
		replicas_count := float64(len(b.Store.ServicePods[svc]))
		istio_requests_total := node.RequestCount.Avg()
		slo_threshold := float64(200)

		switch sg.ReactionType {
		case genome.HPA:
			replicas_count = math.Max(0, math.Round(float64(replicas_count)*(1.0+sg.DeltaReplicas)))
		case genome.VPA_CPU:
			container_cpu_quota = container_cpu_quota * (1.0 + sg.DeltaCPU)
			// container_memory_limit = container_memory_limit * (1.0 + sg.DeltaMem)
		}

		rps_per_replica := istio_requests_total / replicas_count
		cpu_per_rps := container_cpu_quota / istio_requests_total
		memory_per_rps := container_memory_limit / istio_requests_total
		total_cpu_limit := replicas_count * container_cpu_quota
		total_memory_limit := replicas_count * container_memory_limit

		out = append(out, []float64{
			container_cpu_quota,
			container_memory_limit,
			replicas_count,
			istio_requests_total,
			slo_threshold,
			rps_per_replica,
			cpu_per_rps,
			memory_per_rps,
			total_cpu_limit,
			total_memory_limit,
		})
	}
	return out
}
