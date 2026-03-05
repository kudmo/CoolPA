package constraints

import (
	"testing"

	"github.com/kudmo/CoolPA/decision/ga/genome"
)

func TestRepairZeroesInactive(t *testing.T) {
	ce := &ConstraintEngine{ServicePolicies: map[string]ServicePolicy{"svc": {AllowedReactions: []genome.ReactionType{genome.HPA}, MinReplicas: 1, MaxReplicas: 5, MinCPU: 0.1, MaxCPU: 2.0}}}
	g := &genome.ReactionGenome{Genes: []*genome.ServiceGene{{ServiceName: "svc", ReactionType: genome.HPA, DeltaReplicas: 0.5, DeltaCPU: 1.0}}}
	ce.Repair(g)
	if g.Genes[0].DeltaCPU != 0 {
		t.Fatalf("expected DeltaCPU zeroed for HPA gene, got %v", g.Genes[0].DeltaCPU)
	}
}
