package engine

import (
	"math/rand"
)

// TournamentSelection implements k-way tournament selection.
type TournamentSelection struct {
	K int
}

// Select chooses one index from population according to tournament on fitness (higher is better).
func (t *TournamentSelection) Select(rng *rand.Rand, fitness []float64) int {
	if len(fitness) == 0 {
		return -1
	}
	bestIdx := -1
	bestVal := -1.0
	for i := 0; i < t.K; i++ {
		idx := rng.Intn(len(fitness))
		if bestIdx == -1 || fitness[idx] > bestVal {
			bestIdx = idx
			bestVal = fitness[idx]
		}
	}
	return bestIdx
}
