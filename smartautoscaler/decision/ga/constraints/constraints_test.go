package constraints

import (
	"testing"

	"github.com/kudmo/CoolPA/decision/ga/genome"
)

func TestRepairZeroesInactive(t *testing.T) {
	ce := &ConstraintEngine{
		ServicePolicies: map[string]ServicePolicy{
			"svc": {
				AllowedReactions: []genome.ReactionType{genome.HPA},
				MinReplicas:      1,
				MaxReplicas:      5,
				MinCPU:           0.1,
				MaxCPU:           2.0,
				MinMemory:        0.1,
				MaxMemory:        2.0,
			},
		},
	}

	g := &genome.ReactionGenome{
		Genes: []*genome.ServiceGene{
			{
				ServiceName:   "svc",
				ReactionType:  genome.HPA,
				DeltaReplicas: 0.5,
				DeltaCPU:      1.0,
				DeltaMemory:   1.0,
			},
		},
	}

	ce.Repair(g)

	// Для HPA: DeltaCPU и DeltaMemory должны быть обнулены
	if g.Genes[0].DeltaCPU != 0 {
		t.Fatalf("expected DeltaCPU zeroed for HPA gene, got %v", g.Genes[0].DeltaCPU)
	}
	if g.Genes[0].DeltaMemory != 0 {
		t.Fatalf("expected DeltaMemory zeroed for HPA gene, got %v", g.Genes[0].DeltaMemory)
	}
	// DeltaReplicas должен остаться неизменным
	if g.Genes[0].DeltaReplicas != 0.5 {
		t.Fatalf("expected DeltaReplicas unchanged for HPA gene, got %v", g.Genes[0].DeltaReplicas)
	}
}

// Дополнительный тест для проверки VPA
func TestRepairZeroesInactiveForVPA(t *testing.T) {
	ce := &ConstraintEngine{
		ServicePolicies: map[string]ServicePolicy{
			"svc": {
				AllowedReactions: []genome.ReactionType{genome.VPA},
				MinReplicas:      1,
				MaxReplicas:      5,
				MinCPU:           0.1,
				MaxCPU:           2.0,
				MinMemory:        0.1,
				MaxMemory:        2.0,
			},
		},
	}

	g := &genome.ReactionGenome{
		Genes: []*genome.ServiceGene{
			{
				ServiceName:   "svc",
				ReactionType:  genome.VPA,
				DeltaReplicas: 0.5,
				DeltaCPU:      1.0,
				DeltaMemory:   1.0,
			},
		},
	}

	ce.Repair(g)

	// Для VPA: DeltaReplicas должен быть обнулён
	if g.Genes[0].DeltaReplicas != 0 {
		t.Fatalf("expected DeltaReplicas zeroed for VPA gene, got %v", g.Genes[0].DeltaReplicas)
	}
	// DeltaCPU и DeltaMemory должны остаться неизменными
	if g.Genes[0].DeltaCPU != 1.0 {
		t.Fatalf("expected DeltaCPU unchanged for VPA gene, got %v", g.Genes[0].DeltaCPU)
	}
	if g.Genes[0].DeltaMemory != 1.0 {
		t.Fatalf("expected DeltaMemory unchanged for VPA gene, got %v", g.Genes[0].DeltaMemory)
	}
}

// Тест для проверки ограничений памяти
func TestRepairMemoryBounds(t *testing.T) {
	ce := &ConstraintEngine{
		ServicePolicies: map[string]ServicePolicy{
			"svc": {
				AllowedReactions: []genome.ReactionType{genome.VPA},
				MinMemory:        0.5,
				MaxMemory:        2.0,
			},
		},
	}

	g := &genome.ReactionGenome{
		Genes: []*genome.ServiceGene{
			{
				ServiceName:   "svc",
				ReactionType:  genome.VPA,
				DeltaMemory:   3.0, // Превышает MaxMemory
				DeltaReplicas: 0,
				DeltaCPU:      0,
			},
		},
	}

	ce.Repair(g)

	// DeltaMemory должен быть ограничен: MaxMemory - 1 = 2.0 - 1 = 1.0
	if g.Genes[0].DeltaMemory > 1.0 {
		t.Fatalf("expected DeltaMemory clamped to max, got %v", g.Genes[0].DeltaMemory)
	}
}
