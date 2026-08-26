package fitness

import (
	"context"

	"github.com/kudmo/CoolPA/internal/optimizer/ga/genome"
)

// FeatureBuilder converts genomes into per-service feature vectors suitable for predictors.
type FeatureBuilder interface {
	Build(ctx context.Context, genome *genome.ReactionGenome) [][]float64
}
