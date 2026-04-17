package slopredictor

import (
	"log/slog"

	"github.com/kudmo/CoolPA/decision/ga/genome"
	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/CoolPA/storage/metrics"
	"github.com/kudmo/CoolPA/storage/quotas"
)

// ResourceOptimizerFitness composes a FeatureBuilder and Predictor.
type ResourceOptimizerFitness struct {
	Builder   SLOPredictorFeatureBuilder
	Predictor *SLOPredictor
	Store     *storage.Storage
}

func NewResourceOptimizerFitness(store *storage.Storage) *ResourceOptimizerFitness {
	predictor, err := NewSLOPredictor("random_forest_model.onnx", 11)
	if err != nil {
		slog.Error("Error init predictor", "error", err)
	}
	return &ResourceOptimizerFitness{
		Builder:   SLOPredictorFeatureBuilder{Store: store},
		Predictor: predictor,
		Store:     store,
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
	risks := f.Predictor.PredictBatch(X)
	prod := 1.0
	for _, r := range risks {
		prod *= (1.0 - r)
	}
	return prod
}

func (f *ResourceOptimizerFitness) EvaluateBatch(pop []*genome.ReactionGenome) []float64 {
	LAMBDA := 0.5

	scores := make([]float64, len(pop))
	for i, g := range pop {
		R1 := f.calculateR1(g)
		R2 := f.calculateR2(g)
		scores[i] = LAMBDA*R1 + (1-LAMBDA)*R2
	}
	return scores
}
