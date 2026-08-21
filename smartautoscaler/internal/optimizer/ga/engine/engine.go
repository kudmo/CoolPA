package engine

import (
	"context"
	"math/rand"
	"time"

	"github.com/kudmo/CoolPA/internal/optimizer/ga/config"
	"github.com/kudmo/CoolPA/internal/optimizer/ga/constraints"
	"github.com/kudmo/CoolPA/internal/optimizer/ga/fitness"
	"github.com/kudmo/CoolPA/internal/optimizer/ga/genome"
)

// Engine orchestrates the evolutionary loop.
type Engine struct {
	Config      config.Config
	Fitness     fitness.Fitness
	Constraints *constraints.ConstraintEngine
}

// InitPopulation builds an initial population by copying a seed genome or randomizing.
func (e *Engine) InitPopulation(seed *genome.ReactionGenome) []*genome.ReactionGenome {
	pop := make([]*genome.ReactionGenome, e.Config.PopulationSize)
	rng := rand.New(rand.NewSource(e.Config.RandomSeed))
	for i := 0; i < e.Config.PopulationSize; i++ {
		if seed != nil {
			pop[i] = seed.Clone()
			// small randomization
			pop[i].Mutate(rng, e.Config.MutationRate, e.Config.TypeMutationRate, e.Constraints)
		} else {
			// create a minimal random genome with no genes (user should seed realistically)
			pop[i] = &genome.ReactionGenome{Genes: []*genome.ServiceGene{}}
		}
	}
	return pop
}

// Run executes the GA loop and returns best genome found.
func (e *Engine) Run(ctx context.Context, now time.Time, seed *genome.ReactionGenome) (*genome.ReactionGenome, error) {
	rng := rand.New(rand.NewSource(e.Config.RandomSeed))
	if e.Config.RandomSeed == 0 {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	pop := e.InitPopulation(seed)

	var best *genome.ReactionGenome
	for gen := 0; gen < e.Config.Generations; gen++ {
		// Evaluate batch fitness
		scores := e.Fitness.EvaluateBatch(ctx, now, pop)

		// keep elite
		eliteCount := int(float64(len(pop)) * e.Config.EliteRatio)
		elitesIdx := TopNIndices(scores, eliteCount)
		elites := make([]*genome.ReactionGenome, 0, len(elitesIdx))
		for _, i := range elitesIdx {
			elites = append(elites, pop[i].Clone())
		}

		// selection
		selector := TournamentSelection{K: e.Config.TournamentSize}

		// produce next generation
		newPop := make([]*genome.ReactionGenome, 0, len(pop))
		// copy elites first
		for _, el := range elites {
			newPop = append(newPop, el)
		}

		for len(newPop) < len(pop) {
			i1 := selector.Select(rng, scores)
			i2 := selector.Select(rng, scores)
			if i1 < 0 || i2 < 0 {
				break
			}

			p1 := pop[i1].Clone()
			p2 := pop[i2].Clone()

			// crossover
			if rng.Float64() < e.Config.CrossoverRate {
				c1, c2 := UniformCrossover(p1, p2, rng)
				p1, p2 = c1, c2
			}

			// mutation (reaction-aware)
			ApplyMutation(p1, rng, e.Config.MutationRate, e.Config.TypeMutationRate, e.Constraints)
			ApplyMutation(p2, rng, e.Config.MutationRate, e.Config.TypeMutationRate, e.Constraints)

			newPop = append(newPop, p1)
			if len(newPop) < len(pop) {
				newPop = append(newPop, p2)
			}
		}

		pop = newPop
		// Post-generation constraint enforcement
		for _, ind := range pop {
			if e.Constraints != nil {
				e.Constraints.Repair(ind)
			}
		}

		// track best
		scores = e.Fitness.EvaluateBatch(ctx, now, pop)
		bestIdxs := TopNIndices(scores, 1)
		if len(bestIdxs) > 0 {
			best = pop[bestIdxs[0]].Clone()
		}
	}

	return best, nil
}
