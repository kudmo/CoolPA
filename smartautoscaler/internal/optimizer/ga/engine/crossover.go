package engine

import (
	"math/rand"

	"github.com/kudmo/CoolPA/internal/optimizer/ga/genome"
)

// UniformCrossover applies uniform crossover between two parents to produce two children.
func UniformCrossover(p1, p2 *genome.ReactionGenome, rng *rand.Rand) (*genome.ReactionGenome, *genome.ReactionGenome) {
	if p1 == nil || p2 == nil {
		if p1 != nil {
			return p1.Clone(), p1.Clone()
		}
		if p2 != nil {
			return p2.Clone(), p2.Clone()
		}
		return nil, nil
	}
	return p1.Crossover(p2, rng)
}
