package fitness

import "github.com/kudmo/CoolPA/decision/ga/genome"

// Fitness evaluates genomes by building features, predicting SLO risk per-service
// and aggregating to a single score (to maximize).
type Fitness interface {
	EvaluateBatch([]*genome.ReactionGenome) []float64
}

// DefaultFitness composes a FeatureBuilder and Predictor.
type DefaultFitness struct {
	Builder   FeatureBuilder
	Predictor Predictor
}

func (f *DefaultFitness) EvaluateBatch(pop []*genome.ReactionGenome) []float64 {
	scores := make([]float64, len(pop))
	for i, g := range pop {
		X := f.Builder.Build(g)
		risks := f.Predictor.PredictBatch(X)
		// aggregate P(SLO violation) = 1 - Π(1 - Ri)
		prod := 1.0
		for _, r := range risks {
			prod *= (1.0 - r)
		}
		p := 1.0 - prod
		// Fitness to maximize: we invert risk -> prefer low risk by returning (1 - p)
		scores[i] = 1.0 - p
	}
	return scores
}
