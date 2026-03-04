package engine

import (
	"math/rand"

	"github.com/kudmo/CoolPA/decision/ga/constraints"
	"github.com/kudmo/CoolPA/decision/ga/genome"
)

// ApplyMutation delegates to genome.Mutate with provided rates and constraint accessor.
func ApplyMutation(g *genome.ReactionGenome, rng *rand.Rand, mutationRate, typeMutationRate float64, ce *constraints.ConstraintEngine) {
	if g == nil {
		return
	}
	// genome.Mutate is reaction-aware and will call back to ce for allowed reactions/repair.
	g.Mutate(rng, mutationRate, typeMutationRate, ce)
}
