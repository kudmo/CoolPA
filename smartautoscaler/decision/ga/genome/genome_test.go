package genome

import (
	"math/rand"
	"reflect"
	"testing"
)

type stubConstraints struct{}

func (s *stubConstraints) GetAllowedReactions(service string) []ReactionType {
	return []ReactionType{VPA_CPU}
}
func (s *stubConstraints) Repair(g *ReactionGenome)        {}
func (s *stubConstraints) Validate(g *ReactionGenome) bool { return true }

func TestMutateZeroesInactive(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	g := &ReactionGenome{Genes: []*ServiceGene{{ServiceName: "svc", ReactionType: HPA, DeltaReplicas: 0.1, DeltaCPU: 5.0}}}
	g.Mutate(rng, 0.0, 0.0, nil)
	if g.Genes[0].DeltaCPU != 0 {
		t.Fatalf("inactive DeltaCPU not zeroed: %v", g.Genes[0].DeltaCPU)
	}
}

func TestTypeMutationRespectsAllowed(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	g := &ReactionGenome{Genes: []*ServiceGene{{ServiceName: "svc", ReactionType: HPA, DeltaReplicas: 0.2, DeltaCPU: 0.5}}}
	cs := &stubConstraints{}
	g.Mutate(rng, 0.0, 1.0, cs)
	if g.Genes[0].ReactionType != VPA_CPU {
		t.Fatalf("expected ReactionType VPA_CPU after forced type mutation, got %v", g.Genes[0].ReactionType)
	}
	if g.Genes[0].DeltaReplicas != 0 || g.Genes[0].DeltaCPU != 0 {
		t.Fatalf("expected deltas reset after type mutation, got rep=%v cpu=%v", g.Genes[0].DeltaReplicas, g.Genes[0].DeltaCPU)
	}
}

func TestCrossoverAtomic(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	p1 := &ReactionGenome{Genes: []*ServiceGene{{ServiceName: "svc", ReactionType: HPA, DeltaReplicas: 0.3, DeltaCPU: 0}}}
	p2 := &ReactionGenome{Genes: []*ServiceGene{{ServiceName: "svc", ReactionType: VPA_CPU, DeltaReplicas: 0, DeltaCPU: 0.2}}}
	c1, c2 := p1.Crossover(p2, rng)
	if c1 == nil || c2 == nil {
		t.Fatal("crossover returned nil child")
	}
	// child gene must match entirely one of parents
	matchParent := func(g *ServiceGene) bool {
		return reflect.DeepEqual(g, p1.Genes[0]) || reflect.DeepEqual(g, p2.Genes[0])
	}
	if !matchParent(c1.Genes[0]) || !matchParent(c2.Genes[0]) {
		t.Fatalf("child genes not atomic from parents: c1=%v c2=%v", c1.Genes[0], c2.Genes[0])
	}
}

func TestDecodeBehavior(t *testing.T) {
	g := &ReactionGenome{Genes: []*ServiceGene{
		{ServiceName: "a", ReactionType: HPA, DeltaReplicas: 0.5, DeltaCPU: 0},
		{ServiceName: "b", ReactionType: VPA_CPU, DeltaReplicas: 0, DeltaCPU: 0.2},
	}}
	current := []ServiceState{{Name: "a", CurrentReplicas: 2, CurrentCPURel: 1.0}, {Name: "b", CurrentReplicas: 3, CurrentCPURel: 0.5}}
	cand := g.Decode(current)
	if cand.Services[0].Replicas != 3 { // 2 * (1 + 0.5) = 3
		t.Fatalf("unexpected replicas for a: %d", cand.Services[0].Replicas)
	}
	if cand.Services[0].CPURel != 1.0 {
		t.Fatalf("unexpected cpu for a: %v", cand.Services[0].CPURel)
	}
	if cand.Services[1].Replicas != 3 {
		t.Fatalf("unexpected replicas for b: %d", cand.Services[1].Replicas)
	}
	if cand.Services[1].CPURel != 0.5*1.2 {
		t.Fatalf("unexpected cpu for b: %v", cand.Services[1].CPURel)
	}
}
