package fitness

import (
	"github.com/kudmo/CoolPA/decision/ga/genome"
)

// Fitness evaluates genomes by building features, predicting SLO risk per-service
// and aggregating to a single score (to maximize).
type Fitness interface {
	EvaluateBatch([]*genome.ReactionGenome) []float64
}
