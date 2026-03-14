package fitness

import (
	"github.com/kudmo/CoolPA/decision/ga/genome"
	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/CoolPA/storage/metrics"
)

// Fitness evaluates genomes by building features, predicting SLO risk per-service
// and aggregating to a single score (to maximize).
type Fitness interface {
	EvaluateBatch([]*genome.ReactionGenome) []float64
}

// DefaultFitness composes a FeatureBuilder and Predictor.
type DefaultFitness struct {
	Builder   FeatureBuilder
	Predictor Predictor
	Store     *storage.Storage
}

func (f *DefaultFitness) calculateR2(gen *genome.ReactionGenome) float64 {
	n := len(f.Store.Graph.GetServices())
	CPU_MAX := 1200000
	MAX_REPLICES := 10

	current := make([]genome.ServiceState, 0, len(gen.Genes))
	for _, svc := range f.Store.Graph.GetServices() {
		pods := f.Store.ResourceMetrics.GetServicePods(svc)
		cpu_quota, _, _ := f.Store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.CPUQuota)
		current = append(current, genome.ServiceState{Name: svc, CurrentReplicas: len(pods), CurrentCPURel: cpu_quota})
	}
	res := gen.Decode(current)

	cpu_sum := 0.0
	replics_sum := 0.0

	for _, svc := range res.Services {
		cpu_sum += float64(svc.Replicas) * svc.CPURel
		replics_sum += float64(svc.Replicas)
	}
	cpu_percent := cpu_sum / float64(n*CPU_MAX)
	replics_percent := replics_sum / float64(n*MAX_REPLICES)

	return 1 - max(cpu_percent, replics_percent)
}

func (f *DefaultFitness) EvaluateBatch(pop []*genome.ReactionGenome) []float64 {
	LAMBDA := 0.05

	scores := make([]float64, len(pop))
	for i, g := range pop {
		X := f.Builder.Build(g)
		risks := f.Predictor.PredictBatch(X)
		// aggregate P(SLO violation) = 1 - Π(1 - Ri)
		prod := 1.0
		for _, r := range risks {
			prod *= (1.0 - r)
		}
		R1 := 1.0 - prod
		R2 := f.calculateR2(g)
		scores[i] = LAMBDA*R1 + (1-LAMBDA)*R2
	}
	return scores
}
