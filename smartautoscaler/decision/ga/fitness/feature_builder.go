package fitness

import (
	"math"

	"github.com/kudmo/CoolPA/decision/ga/genome"
)

// FeatureBuilder converts genomes into per-service feature vectors suitable for predictors.
type FeatureBuilder interface {
	Build(genome *genome.ReactionGenome) [][]float64
}

// DefaultFeatureBuilder implements a basic builder that uses:
// - one-hot reaction type (HPA/VPA)
// - active relative delta (replicas for HPA, cpu for VPA)
type DefaultFeatureBuilder struct {
}

func (b *DefaultFeatureBuilder) Build(g *genome.ReactionGenome) [][]float64 {
	if g == nil {
		return nil
	}
	out := make([][]float64, 0, len(g.Genes))
	for _, sg := range g.Genes {
		if sg == nil {
			continue
		}
		// reaction one-hot: [isHPA, isVPA]
		isHPA := 0.0
		isVPA := 0.0
		if sg.ReactionType == genome.HPA {
			isHPA = 1.0
		} else {
			isVPA = 1.0
		}
		// active delta only
		var active float64
		if sg.ReactionType == genome.HPA {
			active = sg.DeltaReplicas
		} else {
			active = sg.DeltaCPU
		}
		out = append(out, []float64{isHPA, isVPA, safeNorm(active)})
	}
	return out
}

func safeNorm(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	// small normalisation: squeeze into [-1,1] via tanh-like mapping
	return math.Tanh(v)
}
