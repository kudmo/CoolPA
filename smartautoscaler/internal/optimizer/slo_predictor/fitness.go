package slopredictor

import (
	"context"
	"math"
	"sync"

	"github.com/kudmo/CoolPA/internal/metrics"
	"github.com/kudmo/CoolPA/internal/optimizer/ga/genome"
	"github.com/kudmo/CoolPA/internal/statistics"
	"github.com/kudmo/CoolPA/logger"
	"github.com/kudmo/CoolPA/utils"
)

// ResourceOptimizerFitness composes a FeatureBuilder and Predictor.
type ResourceOptimizerFitness struct {
	metricsProvider metrics.MetricsRepository
	histStore       *statistics.HistStore

	Builder   LatencyDeltaPredictorFeatureBuilder
	Predictor *LatencyDeltaPredictor
	config    FitnessConfig
}

func NewResourceOptimizerFitness(metricsProvider metrics.MetricsRepository, histStore *statistics.HistStore, config FitnessConfig) *ResourceOptimizerFitness {
	predictor, err := NewSLOPredictor("latency_model.onnx", 11)
	if err != nil {
		logger.Error("slo_predictor", "error init predictor", "error", err)
	}

	return &ResourceOptimizerFitness{
		Builder:         LatencyDeltaPredictorFeatureBuilder{metricsProvider: metricsProvider, config: config},
		Predictor:       predictor,
		config:          config,
		metricsProvider: metricsProvider,
		histStore:       histStore,
	}
}

func (f *ResourceOptimizerFitness) Close() {
	f.Predictor.Close()
}

func (f *ResourceOptimizerFitness) calculateR2(ctx context.Context, gen *genome.ReactionGenome) float64 {
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

func (f *ResourceOptimizerFitness) calculateR1(ctx context.Context, g *genome.ReactionGenome, deltas []float64) float64 {
	prod := 1.0

	for i, gene := range g.Genes {
		lat_95_curr, _ := f.metricsProvider.GetServiceAverageLatency95Range(ctx, gene.ServiceName, f.config.Window)
		lat_95_curr_avg := utils.Avg(lat_95_curr)
		lat_95_new := lat_95_curr_avg * math.Pow(math.E, deltas[i])

		risk := f.histStore.GetHistogram(gene.ServiceName).Risk(lat_95_new)
		prod *= (1.0 - risk)
	}
	return prod
}

func (f *ResourceOptimizerFitness) collectFeaturesForBatch(ctx context.Context, pop []*genome.ReactionGenome) [][]float64 {
	allX := make([][]float64, 0, len(pop))
	for _, g := range pop {
		X := f.Builder.Build(ctx, g)
		allX = append(allX, X...)
	}
	return allX
}

func (f *ResourceOptimizerFitness) splitDeltasByGenome(pop []*genome.ReactionGenome, flatDeltas []float64) [][]float64 {
	deltasByGenome := make([][]float64, len(pop))
	idx := 0
	for i, g := range pop {
		genomeLen := len(g.Genes)
		deltasByGenome[i] = flatDeltas[idx : idx+genomeLen]
		idx += genomeLen
	}
	return deltasByGenome
}

func (f *ResourceOptimizerFitness) EvaluateBatch(ctx context.Context, pop []*genome.ReactionGenome) []float64 {
	n := len(pop)
	scores := make([]float64, n)

	allX := f.collectFeaturesForBatch(ctx, pop)
	flatDeltas := f.Predictor.PredictBatch(allX)
	deltasByGenome := f.splitDeltasByGenome(pop, flatDeltas)

	r2s := make([]float64, n)
	var wgR2 sync.WaitGroup
	wgR2.Add(n)
	for i, g := range pop {
		go func(idx int, genome *genome.ReactionGenome) {
			defer wgR2.Done()
			r2s[idx] = f.calculateR2(ctx, genome)
		}(i, g)
	}
	wgR2.Wait()

	r1s := make([]float64, n)

	var wg sync.WaitGroup
	wg.Add(n)

	for i, g := range pop {
		go func(idx int, genome *genome.ReactionGenome, delta []float64) {
			defer wg.Done()
			r1s[idx] = f.calculateR1(ctx, genome, delta)
		}(i, g, deltasByGenome[i])
	}

	wg.Wait()

	for i := range scores {
		scores[i] = f.config.Lambda*r1s[i] + (1-f.config.Lambda)*r2s[i]
	}

	return scores
}
