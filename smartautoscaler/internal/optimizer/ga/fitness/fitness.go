package fitness

import (
	"context"

	"github.com/kudmo/CoolPA/internal/optimizer/ga/genome"
)

// Fitness evaluates genomes by building features, predicting SLO risk per-service
// and aggregating to a single score (to maximize).
type Fitness interface {
	EvaluateBatch(ctx context.Context, genome []*genome.ReactionGenome) []float64
}
