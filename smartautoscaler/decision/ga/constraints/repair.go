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

			// Clamp replica delta so that current + delta falls into [MinReplicas, MaxReplicas]
			if sp.MaxReplicas > 0 && sp.MinReplicas >= 0 {
				deltaMin := float64(sp.MinReplicas) - 1.0
				deltaMax := float64(sp.MaxReplicas) - 1.0
				if sg.DeltaReplicas < deltaMin {
					sg.DeltaReplicas = deltaMin
				}
				if sg.DeltaReplicas > deltaMax {
					sg.DeltaReplicas = deltaMax
				}
			}

			// Clamp CPU deltas relative to baseline
			if sp.MinCPU >= 0 && sp.MaxCPU > 0 {
				cpuMin := sp.MinCPU - 1.0
				cpuMax := sp.MaxCPU - 1.0
				if sg.DeltaCPU < cpuMin {
					sg.DeltaCPU = cpuMin
				}
				if sg.DeltaCPU > cpuMax {
					sg.DeltaCPU = cpuMax
				}
			}

			// Clamp Memory deltas relative to baseline
			if sp.MinMemory >= 0 && sp.MaxMemory > 0 {
				memMin := sp.MinMemory - 1.0
				memMax := sp.MaxMemory - 1.0
				if sg.DeltaMemory < memMin {
					sg.DeltaMemory = memMin
				}
				if sg.DeltaMemory > memMax {
					sg.DeltaMemory = memMax
				}
			}
		}

		// enforce inactive parameter zeroing
		switch sg.ReactionType {
		case genome.HPA:
			sg.DeltaCPU = 0
			sg.DeltaMemory = 0
		case genome.VPA:
			sg.DeltaReplicas = 0
		}

		// general safety clamps
		if math.IsNaN(sg.DeltaReplicas) || math.IsInf(sg.DeltaReplicas, 0) {
			sg.DeltaReplicas = 0
		}
		if math.IsNaN(sg.DeltaCPU) || math.IsInf(sg.DeltaCPU, 0) {
			sg.DeltaCPU = 0
		}
		if math.IsNaN(sg.DeltaMemory) || math.IsInf(sg.DeltaMemory, 0) {
			sg.DeltaMemory = 0
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

			// check numerical bounds
			// Replicas bound check
			if sp.MaxReplicas > 0 {
				if int(math.Round(1.0+sg.DeltaReplicas)) > sp.MaxReplicas {
					return false
				}
			}

			// CPU bound check
			if sp.MaxCPU > 0 {
				if 1.0+sg.DeltaCPU > sp.MaxCPU {
					return false
				}
			}

			// Memory bound check
			if sp.MaxMemory > 0 {
				if 1.0+sg.DeltaMemory > sp.MaxMemory {
					return false
				}
			}
		}
	}
	return true
}
