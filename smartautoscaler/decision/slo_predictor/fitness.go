package slopredictor

import (
	"github.com/kudmo/CoolPA/decision/ga/genome"
	"github.com/kudmo/CoolPA/storage"
)

type ResourceOptimizerFitnessConfig struct {
	CpuMax      float64
	MemMax      float64
	ReplicasMax float64
}

// ResourceOptimizerFitness composes a FeatureBuilder and Predictor.
type ResourceOptimizerFitness struct {
	Builder   SLOPredictorFeatureBuilder
	Predictor SLOPredictor
	Store     *storage.Storage

	Config ResourceOptimizerFitnessConfig
}

func NewResourceOptimizerFitness(store *storage.Storage) *ResourceOptimizerFitness {
	return &ResourceOptimizerFitness{
		Builder:   SLOPredictorFeatureBuilder{Store: store},
		Predictor: SLOPredictor{},
		Store:     store,
	}
}

func (f *ResourceOptimizerFitness) calculateR2(gen *genome.ReactionGenome) float64 {
	res := gen.Decode()

	cpu_sum := 0.0
	mem_sum := 0.0
	replics_sum := 0.0

	for _, svc := range res.Services {
		cpu_sum += float64(svc.Replicas) * svc.CPURel
		mem_sum += float64(svc.Replicas) * svc.MemoryRel
		replics_sum += float64(svc.Replicas)
	}
	cpu_percent := cpu_sum / (f.Config.CpuMax)
	mem_percent := mem_sum / (f.Config.MemMax)
	replics_percent := replics_sum / (f.Config.ReplicasMax)

	return 1 - max(cpu_percent, replics_percent, mem_percent)
}

func (f *ResourceOptimizerFitness) EvaluateBatch(pop []*genome.ReactionGenome) []float64 {
	LAMBDA := 0.05

	scores := make([]float64, len(pop))
	for i, g := range pop {
		X := f.Builder.Build(g)
		risks := f.Predictor.PredictBatch(X)
		// aggregate P(SLO) = Π(1 - Ri)
		prod := 1.0
		for _, r := range risks {
			prod *= (1.0 - r)
		}
		R1 := prod
		R2 := f.calculateR2(g)
		scores[i] = LAMBDA*R1 + (1-LAMBDA)*R2
	}
	return scores
}
