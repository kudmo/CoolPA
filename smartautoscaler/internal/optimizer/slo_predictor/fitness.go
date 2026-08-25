package slopredictor

import (
	"context"
	"math"
	"time"

	"github.com/kudmo/CoolPA/internal/optimizer/ga/genome"
	"github.com/kudmo/CoolPA/internal/optimizer/interfaces"
	"github.com/kudmo/CoolPA/internal/statistics"
	"github.com/kudmo/CoolPA/logger"
	"github.com/kudmo/CoolPA/utils"
)

// ResourceOptimizerFitness composes a FeatureBuilder and Predictor.
type ResourceOptimizerFitness struct {
	metricsProvider interfaces.MetricsRepository
	histStore       *statistics.HistStore

	Builder   LatencyDeltaPredictorFeatureBuilder
	Predictor *LatencyDeltaPredictor
	config    FitnessConfig
}

func NewResourceOptimizerFitness(metricsProvider interfaces.MetricsRepository, histStore *statistics.HistStore, config FitnessConfig) *ResourceOptimizerFitness {
	predictor, err := NewSLOPredictor("latency_model.onnx", 11)
	if err != nil {
		logger.Error("slo_predictor", "error init predictor", "error", err)
	}

	return &ResourceOptimizerFitness{
		Builder:   LatencyDeltaPredictorFeatureBuilder{metricsProvider: metricsProvider, config: config},
		Predictor: predictor,
		config:    config,
		histStore: histStore,
	}
}

func (f *ResourceOptimizerFitness) Close() {
	f.Predictor.Close()
}

func (f *ResourceOptimizerFitness) calculateR2(ctx context.Context, now time.Time, gen *genome.ReactionGenome) float64 {
	current := make([]genome.ServiceState, 0, len(gen.Genes))
	services, _ := f.metricsProvider.ListServices(ctx)
	for _, svc := range services {
		replicas_count, _ := f.metricsProvider.GetServiceReplicasCountValue(ctx, svc)
		cpu_app_quota, _ := f.metricsProvider.GetServiceCpuQuota(ctx, svc)
		cpu_pod_quota, _ := f.metricsProvider.GetServicePodCpuQuota(ctx, svc)
		mem_app_quota, _ := f.metricsProvider.GetServiceMemoryQuota(ctx, svc)
		mem_pod_quota, _ := f.metricsProvider.GetServicePodMemoryQuota(ctx, svc)
		current = append(current, genome.ServiceState{Name: svc, CurrentReplicas: int(replicas_count), CurrentAppCPU: cpu_app_quota, CurrentPodCPU: cpu_pod_quota, CurrentAppMemory: mem_app_quota, CurrentPodMemory: mem_pod_quota})
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
	total_cpu_limit, _ := f.metricsProvider.GetGlobalTotalCpuLimit(ctx)
	total_mem_limit, _ := f.metricsProvider.GetGlobalTotalMemoryLimit(ctx)
	total_replicas_limit, _ := f.metricsProvider.GetGlobalTotalPodsLimit(ctx)

	cpu_percent := cpu_sum / total_cpu_limit
	mem_percent := mem_sum / total_mem_limit
	replics_percent := replics_sum / total_replicas_limit

	return 1 - max(cpu_percent, replics_percent, mem_percent)
}

func (f *ResourceOptimizerFitness) calculateR1(ctx context.Context, now time.Time, g *genome.ReactionGenome) float64 {
	X := f.Builder.Build(ctx, now, g)
	deltas := f.Predictor.PredictBatch(X)
	prod := 1.0

	fromBegin := now.Add(-f.config.Window)

	for i, g := range g.Genes {
		lat_95_curr, _ := f.metricsProvider.GetServiceAverageLatency95Range(ctx, g.ServiceName, fromBegin, now)
		lat_95_curr_avg := utils.Avg(lat_95_curr)
		lat_95_new := lat_95_curr_avg * math.Pow(math.E, deltas[i])

		// TODO: Перенести это в отдельное хранилище, используемое
		risk := f.histStore.GetHistogram(g.ServiceName).Risk(lat_95_new)
		prod *= (1.0 - risk)
	}
	return prod
}

func (f *ResourceOptimizerFitness) EvaluateBatch(ctx context.Context, now time.Time, pop []*genome.ReactionGenome) []float64 {
	scores := make([]float64, len(pop))
	for i, g := range pop {
		R1 := f.calculateR1(ctx, now, g)
		R2 := f.calculateR2(ctx, now, g)
		scores[i] = f.config.Lambda*R1 + (1-f.config.Lambda)*R2
	}
	return scores
}
