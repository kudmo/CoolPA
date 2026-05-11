package slopredictor

import (
	"math"
	"time"

	"github.com/kudmo/CoolPA/decision/ga/genome"
	"github.com/kudmo/CoolPA/logger"
	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/CoolPA/storage/metrics"
	"github.com/kudmo/CoolPA/storage/quotas"
)

// ResourceOptimizerFitness composes a FeatureBuilder and Predictor.
type ResourceOptimizerFitness struct {
	Builder               LatencyDeltaPredictorFeatureBuilder
	Predictor             *LatencyDeltaPredictor
	Store                 *storage.Storage
	cachedP95LatencyState map[string]float64
	Lambda                float64
}

func NewResourceOptimizerFitness(store *storage.Storage, lambda float64) *ResourceOptimizerFitness {
	predictor, err := NewSLOPredictor("latency_model.onnx", 11)
	if err != nil {
		logger.Error("slo_predictor", "error init predictor", "error", err)
	}

	now := time.Now()
	fromTime := now.Add(-1 * time.Minute)
	cachedP95LatencyState := map[string]float64{}
	for _, s := range store.ResourceMetrics.GetServices() {
		cachedP95LatencyState[s] = store.Graph.AverageLatencyByInboundRange(s, fromTime, now, true)
	}
	return &ResourceOptimizerFitness{
		Builder:               LatencyDeltaPredictorFeatureBuilder{Store: store},
		Predictor:             predictor,
		Store:                 store,
		cachedP95LatencyState: cachedP95LatencyState,
		Lambda:                lambda,
	}
}

func (f *ResourceOptimizerFitness) Close() {
	f.Predictor.Close()
}

func (f *ResourceOptimizerFitness) calculateR2(gen *genome.ReactionGenome) float64 {
	current := make([]genome.ServiceState, 0, len(gen.Genes))
	for _, svc := range f.Store.ResourceMetrics.GetServices() {
		pods := f.Store.ResourceMetrics.GetServicePods(svc)
		cpu_app_quota, _, _ := f.Store.ResourceMetrics.GetServiceMetricAvgHead(svc, metrics.CPUQuota)
		cpu_pod_quota, _, _ := f.Store.ResourceMetrics.GetServiceMetricAvgHead(svc, metrics.PodCPUQuota)
		mem_app_quota, _, _ := f.Store.ResourceMetrics.GetServiceMetricAvgHead(svc, metrics.MemoryLimit)
		mem_pod_quota, _, _ := f.Store.ResourceMetrics.GetServiceMetricAvgHead(svc, metrics.PodMemoryLimit)
		current = append(current, genome.ServiceState{Name: svc, CurrentReplicas: len(pods), CurrentAppCPU: cpu_app_quota, CurrentPodCPU: cpu_pod_quota, CurrentAppMemory: mem_app_quota, CurrentPodMemory: mem_pod_quota})
	}
	res := gen.DecodeAll(current)

	cpu_sum := 0.0
	mem_sum := 0.0
	replics_sum := 0.0

	for _, svc := range res.Services {
		cpu_sum += float64(svc.Replicas) * svc.PodCPU
		mem_sum += float64(svc.Replicas) * svc.PodMemory
		replics_sum += float64(svc.Replicas)
	}
	cpu_percent := cpu_sum / float64(f.Store.Limits.NamespaceLimits[quotas.NamespaceMaxCpu])
	mem_percent := mem_sum / float64(f.Store.Limits.NamespaceLimits[quotas.NamespaceMaxMem])
	replics_percent := replics_sum / float64(f.Store.Limits.NamespaceLimits[quotas.NamespaceMaxPods])

	return 1 - max(cpu_percent, replics_percent, mem_percent)
}

func (f *ResourceOptimizerFitness) calculateR1(g *genome.ReactionGenome) float64 {
	X := f.Builder.Build(g)
	deltas := f.Predictor.PredictBatch(X)
	prod := 1.0

	for i, g := range g.Genes {
		lat_95_curr_avg := f.cachedP95LatencyState[g.ServiceName]
		lat_95_new := lat_95_curr_avg * math.Pow(math.E, deltas[i])
		risk := f.Store.Hist.GetHistogram(g.ServiceName).Risk(lat_95_new)
		prod *= (1.0 - risk)
	}
	return prod
}

func (f *ResourceOptimizerFitness) EvaluateBatch(pop []*genome.ReactionGenome) []float64 {
	scores := make([]float64, len(pop))
	for i, g := range pop {
		R1 := f.calculateR1(g)
		R2 := f.calculateR2(g)
		scores[i] = f.Lambda*R1 + (1-f.Lambda)*R2
	}
	return scores
}
