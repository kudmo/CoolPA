package fitness

import (
	"github.com/kudmo/CoolPA/decision/ga/genome"
)

// FeatureBuilder converts genomes into per-service feature vectors suitable for predictors.
type FeatureBuilder interface {
	Build(genome *genome.ReactionGenome) [][]float64
}
