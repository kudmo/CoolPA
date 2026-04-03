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
				ServiceName:     "svc",
				ReactionType:    genome.HPA,
				DeltaReplicas:   0.5,
				DeltaCPU:        1.0,
				DeltaMemory:     1.0,
				CurrentReplicas: 1,
				CurrentCPU:      1,
				CurrentMemory:   1,
			},
		},
	}

	ce.Repair(g)

	if g.Genes[0].DeltaCPU != 0 {
		t.Fatalf("expected DeltaCPU zeroed for HPA gene, got %v", g.Genes[0].DeltaCPU)
	}
	if g.Genes[0].DeltaMemory != 0 {
		t.Fatalf("expected DeltaMemory zeroed for HPA gene, got %v", g.Genes[0].DeltaMemory)
	}
	if g.Genes[0].DeltaReplicas != 0.5 {
		t.Fatalf("expected DeltaReplicas unchanged for HPA gene, got %v", g.Genes[0].DeltaReplicas)
	}
}

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
				ServiceName:     "svc",
				ReactionType:    genome.VPA,
				DeltaReplicas:   0.5,
				DeltaCPU:        1.0,
				DeltaMemory:     1.0,
				CurrentReplicas: 1,
				CurrentCPU:      1,
				CurrentMemory:   1,
			},
		},
	}

	ce.Repair(g)

	if g.Genes[0].DeltaReplicas != 0 {
		t.Fatalf("expected DeltaReplicas zeroed for VPA gene, got %v", g.Genes[0].DeltaReplicas)
	}
	if g.Genes[0].DeltaCPU != 1.0 {
		t.Fatalf("expected DeltaCPU unchanged for VPA gene, got %v", g.Genes[0].DeltaCPU)
	}
	if g.Genes[0].DeltaMemory != 1.0 {
		t.Fatalf("expected DeltaMemory unchanged for VPA gene, got %v", g.Genes[0].DeltaMemory)
	}
}

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
				ServiceName:     "svc",
				ReactionType:    genome.VPA,
				DeltaMemory:     3.0,
				DeltaReplicas:   0,
				DeltaCPU:        0,
				CurrentReplicas: 1,
				CurrentCPU:      1,
				CurrentMemory:   1,
			},
		},
	}

	ce.Repair(g)

	expectedDelta := 2.0 - 1.0
	if g.Genes[0].DeltaMemory > expectedDelta {
		t.Fatalf("expected DeltaMemory clamped to %v, got %v", expectedDelta, g.Genes[0].DeltaMemory)
	}
}

func TestRepairReplicasBounds(t *testing.T) {
	ce := &ConstraintEngine{
		ServicePolicies: map[string]ServicePolicy{
			"svc": {
				AllowedReactions: []genome.ReactionType{genome.HPA},
				MinReplicas:      2,
				MaxReplicas:      10,
			},
		},
	}

	g := &genome.ReactionGenome{
		Genes: []*genome.ServiceGene{
			{
				ServiceName:     "svc",
				ReactionType:    genome.HPA,
				DeltaReplicas:   5.0,
				CurrentReplicas: 3,
			},
		},
	}

	ce.Repair(g)

	expectedDelta := 10.0 - 3.0
	if g.Genes[0].DeltaReplicas > expectedDelta {
		t.Fatalf("expected DeltaReplicas clamped to %v, got %v", expectedDelta, g.Genes[0].DeltaReplicas)
	}

	g.Genes[0].DeltaReplicas = -4.0
	ce.Repair(g)

	expectedDelta = 2.0 - 3.0
	if g.Genes[0].DeltaReplicas < expectedDelta {
		t.Fatalf("expected DeltaReplicas clamped to %v, got %v", expectedDelta, g.Genes[0].DeltaReplicas)
	}
}

func TestValidateWithCurrentValues(t *testing.T) {
	ce := &ConstraintEngine{
		ServicePolicies: map[string]ServicePolicy{
			"svc": {
				AllowedReactions: []genome.ReactionType{genome.HPA},
				MinReplicas:      2,
				MaxReplicas:      10,
				MinCPU:           0.5,
				MaxCPU:           2.0,
				MinMemory:        0.5,
				MaxMemory:        2.0,
			},
		},
	}

	g := &genome.ReactionGenome{
		Genes: []*genome.ServiceGene{
			{
				ServiceName:     "svc",
				ReactionType:    genome.HPA,
				DeltaReplicas:   2.0,
				CurrentReplicas: 3,
			},
		},
	}

	if !ce.Validate(g) {
		t.Fatal("expected valid genome")
	}

	g.Genes[0].DeltaReplicas = 10.0
	if ce.Validate(g) {
		t.Fatal("expected invalid genome (replicas exceed max)")
	}
}

func TestApplyGlobalScalingIfExceeded(t *testing.T) {
	ce := &ConstraintEngine{
		GlobalPolicy: GlobalPolicy{
			ClusterCPULimit:     5.0,
			ClusterMemoryLimit:  10.0,
			ClusterReplicaLimit: 8,
		},
		ServicePolicies: map[string]ServicePolicy{
			"svc1": {
				MaxCPU:    3.0,
				MaxMemory: 6.0,
			},
			"svc2": {
				MaxCPU:    3.0,
				MaxMemory: 6.0,
			},
		},
	}

	g := &genome.ReactionGenome{
		Genes: []*genome.ServiceGene{
			{
				ServiceName:     "svc1",
				ReactionType:    genome.VPA,
				DeltaCPU:        2.0,
				DeltaMemory:     3.0,
				CurrentReplicas: 1,
				CurrentCPU:      1,
				CurrentMemory:   1,
			},
			{
				ServiceName:     "svc2",
				ReactionType:    genome.VPA,
				DeltaCPU:        2.0,
				DeltaMemory:     3.0,
				CurrentReplicas: 1,
				CurrentCPU:      1,
				CurrentMemory:   1,
			},
		},
	}

	err := ce.ApplyGlobalScalingIfExceeded(g)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if g.Genes[0].DeltaCPU+g.Genes[1].DeltaCPU+2.0 > 5.0 {
		t.Fatalf("CPU limit exceeded")
	}
}

func TestRepairAllowsOnlyAllowedReaction(t *testing.T) {
	ce := &ConstraintEngine{
		ServicePolicies: map[string]ServicePolicy{
			"svc": {
				AllowedReactions: []genome.ReactionType{genome.VPA},
			},
		},
	}

	g := &genome.ReactionGenome{
		Genes: []*genome.ServiceGene{
			{
				ServiceName:  "svc",
				ReactionType: genome.HPA,
			},
		},
	}

	ce.Repair(g)

	if g.Genes[0].ReactionType != genome.VPA {
		t.Fatalf("expected reaction type forced to VPA, got %v", g.Genes[0].ReactionType)
	}
}
