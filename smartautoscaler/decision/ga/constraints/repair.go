package constraints

import (
	"math"

	"github.com/kudmo/CoolPA/decision/ga/genome"
)

// Repair adjusts genome genes to obey service-level policy bounds.
func (ce *ConstraintEngine) Repair(g *genome.ReactionGenome) {
	if ce == nil || g == nil {
		return
	}
	for _, sg := range g.Genes {
		if sg == nil {
			continue
		}
		// service-level policy
		sp, ok := ce.ServicePolicies[sg.ServiceName]
		if ok {
			// Reaction constraints: if only one allowed reaction, enforce it
			if len(sp.AllowedReactions) == 1 {
				sg.ReactionType = sp.AllowedReactions[0]
			}

			// Clamp replica delta so that (1*(1+delta)) falls into [MinReplicas, MaxReplicas]
			if sp.MaxReplicas > 0 && sp.MinReplicas >= 0 {
				deltaMin := float64(sp.MinReplicas)/1.0 - 1.0
				deltaMax := float64(sp.MaxReplicas)/1.0 - 1.0
				if sg.DeltaReplicas < deltaMin {
					sg.DeltaReplicas = deltaMin
				}
				if sg.DeltaReplicas > deltaMax {
					sg.DeltaReplicas = deltaMax
				}
			}

			// Clamp CPU deltas relative to 1.0 baseline
			if sp.MinCPU >= 0 && sp.MaxCPU > 0 {
				cpuMin := sp.MinCPU/1.0 - 1.0
				cpuMax := sp.MaxCPU/1.0 - 1.0
				if sg.DeltaCPU < cpuMin {
					sg.DeltaCPU = cpuMin
				}
				if sg.DeltaCPU > cpuMax {
					sg.DeltaCPU = cpuMax
				}
			}
		}

		// enforce inactive parameter zeroing
		if sg.ReactionType == genome.HPA {
			sg.DeltaCPU = 0
		} else {
			sg.DeltaReplicas = 0
		}

		// general safety clamps
		if math.IsNaN(sg.DeltaReplicas) || math.IsInf(sg.DeltaReplicas, 0) {
			sg.DeltaReplicas = 0
		}
		if math.IsNaN(sg.DeltaCPU) || math.IsInf(sg.DeltaCPU, 0) {
			sg.DeltaCPU = 0
		}
	}
}

// Validate returns true if the genome appears to respect policies.
func (ce *ConstraintEngine) Validate(g *genome.ReactionGenome) bool {
	if ce == nil || g == nil {
		return false
	}
	for _, sg := range g.Genes {
		if sg == nil {
			continue
		}
		if sp, ok := ce.ServicePolicies[sg.ServiceName]; ok {
			// check reaction allowed
			if len(sp.AllowedReactions) > 0 {
				found := false
				for _, r := range sp.AllowedReactions {
					if r == sg.ReactionType {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
			// check numerical bounds (conservative using baseline=1)
			if sp.MaxReplicas > 0 {
				if int(math.Round(1.0*(1.0+sg.DeltaReplicas))) > sp.MaxReplicas {
					return false
				}
			}
		}
	}
	return true
}
