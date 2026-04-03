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

		sp, ok := ce.ServicePolicies[sg.ServiceName]
		if ok {
			if len(sp.AllowedReactions) == 1 {
				sg.ReactionType = sp.AllowedReactions[0]
			}

			currentReplicas := sg.CurrentReplicas
			if currentReplicas <= 0 {
				currentReplicas = 1
			}

			currentCPU := sg.CurrentCPU
			if currentCPU <= 0 {
				currentCPU = 1
			}

			currentMemory := sg.CurrentMemory
			if currentMemory <= 0 {
				currentMemory = 1
			}

			if sp.MaxReplicas > 0 && sp.MinReplicas >= 0 {
				minDelta := float64(sp.MinReplicas) - currentReplicas
				maxDelta := float64(sp.MaxReplicas) - currentReplicas
				if sg.DeltaReplicas < minDelta {
					sg.DeltaReplicas = minDelta
				}
				if sg.DeltaReplicas > maxDelta {
					sg.DeltaReplicas = maxDelta
				}
			}

			if sp.MinCPU >= 0 && sp.MaxCPU > 0 {
				minDelta := sp.MinCPU - currentCPU
				maxDelta := sp.MaxCPU - currentCPU
				if sg.DeltaCPU < minDelta {
					sg.DeltaCPU = minDelta
				}
				if sg.DeltaCPU > maxDelta {
					sg.DeltaCPU = maxDelta
				}
			}

			if sp.MinMemory >= 0 && sp.MaxMemory > 0 {
				minDelta := sp.MinMemory - currentMemory
				maxDelta := sp.MaxMemory - currentMemory
				if sg.DeltaMemory < minDelta {
					sg.DeltaMemory = minDelta
				}
				if sg.DeltaMemory > maxDelta {
					sg.DeltaMemory = maxDelta
				}
			}
		}

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

		sp, ok := ce.ServicePolicies[sg.ServiceName]
		if !ok {
			continue
		}

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

		currentReplicas := sg.CurrentReplicas
		if currentReplicas <= 0 {
			currentReplicas = 1
		}

		currentCPU := sg.CurrentCPU
		if currentCPU <= 0 {
			currentCPU = 1
		}

		currentMemory := sg.CurrentMemory
		if currentMemory <= 0 {
			currentMemory = 1
		}

		if sp.MaxReplicas > 0 {
			if int(math.Round(currentReplicas+sg.DeltaReplicas)) > sp.MaxReplicas {
				return false
			}
			if int(math.Round(currentReplicas+sg.DeltaReplicas)) < sp.MinReplicas {
				return false
			}
		}

		if sp.MaxCPU > 0 {
			if currentCPU+sg.DeltaCPU > sp.MaxCPU {
				return false
			}
			if currentCPU+sg.DeltaCPU < sp.MinCPU {
				return false
			}
		}

		if sp.MaxMemory > 0 {
			if currentMemory+sg.DeltaMemory > sp.MaxMemory {
				return false
			}
			if currentMemory+sg.DeltaMemory < sp.MinMemory {
				return false
			}
		}
	}

	return true
}
