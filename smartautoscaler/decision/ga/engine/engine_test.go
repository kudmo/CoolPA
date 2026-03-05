package engine

import (
	"testing"

	"github.com/kudmo/CoolPA/decision/ga/config"
	"github.com/kudmo/CoolPA/decision/ga/constraints"
	"github.com/kudmo/CoolPA/decision/ga/fitness"
	"github.com/kudmo/CoolPA/decision/ga/genome"
)

func TestEngineRunBasic(t *testing.T) {
	cfg := config.Config{PopulationSize: 6, Generations: 3, EliteRatio: 0.2, MutationRate: 0.1, TypeMutationRate: 0.05, CrossoverRate: 0.7, TournamentSize: 2, RandomSeed: 1}
	ce := &constraints.ConstraintEngine{ServicePolicies: map[string]constraints.ServicePolicy{"s": {AllowedReactions: []genome.ReactionType{genome.HPA, genome.VPA_CPU}, MinReplicas: 1, MaxReplicas: 5, MinCPU: 0.1, MaxCPU: 2.0}}}
	fb := &fitness.DefaultFeatureBuilder{}
	pred := &fitness.DummyPredictor{}
	fit := &fitness.DefaultFitness{Builder: fb, Predictor: pred}

	eng := &Engine{Config: cfg, Fitness: fit, Constraints: ce}

	// seed genome for single service
	seed := &genome.ReactionGenome{Genes: []*genome.ServiceGene{{ServiceName: "s", ReactionType: genome.HPA, DeltaReplicas: 0.1}}}

	// Run should not error and return a best genome
	best, err := eng.Run(seed)
	if err != nil {
		t.Fatalf("engine run error: %v", err)
	}
	if best == nil {
		t.Fatalf("expected non-nil best genome")
	}
	if len(best.Genes) != len(seed.Genes) {
		t.Fatalf("unexpected gene count in best: %d", len(best.Genes))
	}
	// ensure inactive params zeroed
	for _, g := range best.Genes {
		if g.ReactionType == genome.HPA && g.DeltaCPU != 0 {
			t.Fatalf("inactive cpu not zeroed in best: %v", g.DeltaCPU)
		}
		if g.ReactionType == genome.VPA_CPU && g.DeltaReplicas != 0 {
			t.Fatalf("inactive replicas not zeroed in best: %v", g.DeltaReplicas)
		}
	}
}
