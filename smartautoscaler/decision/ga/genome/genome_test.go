package genome

import (
	"math/rand"
	"reflect"
	"testing"
)

type stubConstraints struct{}

func (s *stubConstraints) GetAllowedReactions(service string) []ReactionType {
	return []ReactionType{VPA} // VPA_CPU заменён на VPA
}
func (s *stubConstraints) Repair(g *ReactionGenome)        {}
func (s *stubConstraints) Validate(g *ReactionGenome) bool { return true }

func TestMutateZeroesInactive(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	g := &ReactionGenome{Genes: []*ServiceGene{{
		ServiceName:   "svc",
		ReactionType:  HPA,
		DeltaReplicas: 0.1,
		DeltaCPU:      5.0,
		DeltaMemory:   3.0,
	}}}
	g.Mutate(rng, 0.0, 0.0, nil)
	if g.Genes[0].DeltaCPU != 0 {
		t.Fatalf("inactive DeltaCPU not zeroed: %v", g.Genes[0].DeltaCPU)
	}
	if g.Genes[0].DeltaMemory != 0 {
		t.Fatalf("inactive DeltaMemory not zeroed: %v", g.Genes[0].DeltaMemory)
	}
}

func TestTypeMutationRespectsAllowed(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	g := &ReactionGenome{Genes: []*ServiceGene{{
		ServiceName:   "svc",
		ReactionType:  HPA,
		DeltaReplicas: 0.2,
		DeltaCPU:      0.5,
		DeltaMemory:   0.3,
	}}}
	cs := &stubConstraints{}
	g.Mutate(rng, 0.0, 1.0, cs)
	if g.Genes[0].ReactionType != VPA {
		t.Fatalf("expected ReactionType VPA after forced type mutation, got %v", g.Genes[0].ReactionType)
	}
	if g.Genes[0].DeltaReplicas != 0 || g.Genes[0].DeltaCPU != 0 || g.Genes[0].DeltaMemory != 0 {
		t.Fatalf("expected deltas reset after type mutation, got rep=%v cpu=%v mem=%v",
			g.Genes[0].DeltaReplicas, g.Genes[0].DeltaCPU, g.Genes[0].DeltaMemory)
	}
}

func TestCrossoverAtomic(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	p1 := &ReactionGenome{Genes: []*ServiceGene{{
		ServiceName:   "svc",
		ReactionType:  HPA,
		DeltaReplicas: 0.3,
		DeltaCPU:      0,
		DeltaMemory:   0,
	}}}
	p2 := &ReactionGenome{Genes: []*ServiceGene{{
		ServiceName:   "svc",
		ReactionType:  VPA,
		DeltaReplicas: 0,
		DeltaCPU:      0.2,
		DeltaMemory:   0.1,
	}}}
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
		{
			ServiceName:   "a",
			ReactionType:  HPA,
			DeltaReplicas: 0.5,
			DeltaCPU:      0,
			DeltaMemory:   0,
		},
		{
			ServiceName:   "b",
			ReactionType:  VPA,
			DeltaReplicas: 0,
			DeltaCPU:      0.2,
			DeltaMemory:   0.1,
		},
	}}
	current := []ServiceState{
		{Name: "a", CurrentReplicas: 2, CurrentCPURel: 1.0, CurrentMemoryRel: 0.8},
		{Name: "b", CurrentReplicas: 3, CurrentCPURel: 0.5, CurrentMemoryRel: 0.6},
	}
	cand := g.Decode(current)

	// Test HPA case (service a)
	if cand.Services[0].Replicas != 3 { // 2 + 0.5 = 2.5 -> rounded to 3
		t.Fatalf("unexpected replicas for a: got %d, want 3", cand.Services[0].Replicas)
	}
	if cand.Services[0].CPURel != 1.0 {
		t.Fatalf("unexpected cpu for a: got %v, want 1.0", cand.Services[0].CPURel)
	}
	if cand.Services[0].MemoryRel != 0.8 {
		t.Fatalf("unexpected memory for a: got %v, want 0.8", cand.Services[0].MemoryRel)
	}
	if cand.Services[0].Reaction != HPA {
		t.Fatalf("unexpected reaction type for a: got %v, want HPA", cand.Services[0].Reaction)
	}

	// Test VPA case (service b)
	if cand.Services[1].Replicas != 3 {
		t.Fatalf("unexpected replicas for b: got %d, want 3", cand.Services[1].Replicas)
	}
	expectedCPU := 0.5 + 0.2 // 0.7
	if cand.Services[1].CPURel != expectedCPU {
		t.Fatalf("unexpected cpu for b: got %v, want %v", cand.Services[1].CPURel, expectedCPU)
	}
	expectedMem := 0.6 + 0.1 // 0.7
	if cand.Services[1].MemoryRel != expectedMem {
		t.Fatalf("unexpected memory for b: got %v, want %v", cand.Services[1].MemoryRel, expectedMem)
	}
	if cand.Services[1].Reaction != VPA {
		t.Fatalf("unexpected reaction type for b: got %v, want VPA", cand.Services[1].Reaction)
	}
}

func TestDecodeNoDecision(t *testing.T) {
	// Service not in genome should keep current state
	g := &ReactionGenome{Genes: []*ServiceGene{
		{ServiceName: "a", ReactionType: HPA, DeltaReplicas: 0.5, DeltaCPU: 0, DeltaMemory: 0},
	}}
	current := []ServiceState{
		{Name: "a", CurrentReplicas: 2, CurrentCPURel: 1.0, CurrentMemoryRel: 0.8},
		{Name: "b", CurrentReplicas: 3, CurrentCPURel: 0.5, CurrentMemoryRel: 0.6},
	}
	cand := g.Decode(current)

	// Service a should have changes
	if cand.Services[0].Replicas != 3 {
		t.Fatalf("unexpected replicas for a: got %d, want 3", cand.Services[0].Replicas)
	}

	// Service b should be unchanged (not in genome)
	if cand.Services[1].Replicas != 3 {
		t.Fatalf("unexpected replicas for b: got %d, want 3", cand.Services[1].Replicas)
	}
	if cand.Services[1].CPURel != 0.5 {
		t.Fatalf("unexpected cpu for b: got %v, want 0.5", cand.Services[1].CPURel)
	}
	if cand.Services[1].MemoryRel != 0.6 {
		t.Fatalf("unexpected memory for b: got %v, want 0.6", cand.Services[1].MemoryRel)
	}
	if cand.Services[1].Reaction != HPA {
		t.Fatalf("unexpected reaction type for b: got %v, want HPA", cand.Services[1].Reaction)
	}
}
