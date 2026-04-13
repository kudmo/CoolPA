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

			if sp.MaxReplicas > 0 && sp.MinReplicas >= 0 {
				minDelta := float64(sp.MinReplicas) - sg.CurrentReplicas
				maxDelta := float64(sp.MaxReplicas) - sg.CurrentReplicas
				if sg.DeltaReplicas < minDelta {
					sg.DeltaReplicas = minDelta
				}
				if sg.DeltaReplicas > maxDelta {
					sg.DeltaReplicas = maxDelta
				}
			}

			if sp.MinCPU >= 0 && sp.MaxCPU > 0 {
				minDelta := sp.MinCPU - sg.CurrentAppCPU
				maxDelta := sp.MaxCPU - sg.CurrentAppCPU
				if sg.DeltaCPU < minDelta {
					sg.DeltaCPU = minDelta
				}
				if sg.DeltaCPU > maxDelta {
					sg.DeltaCPU = maxDelta
				}
			}

			if sp.MinMemory >= 0 && sp.MaxMemory > 0 {
				minDelta := sp.MinMemory - sg.CurrentAppMemory
				maxDelta := sp.MaxMemory - sg.CurrentAppMemory
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

func (ce *ConstraintEngine) Validate(g *genome.ReactionGenome) {
	totalCPUdelta := 0.0
	totalMemorydelta := 0.0
	totalReplicasdelta := 0

	for _, sg := range g.Genes {
		totalCPUdelta += ((sg.CurrentPodCPU + sg.DeltaCPU) * (sg.CurrentReplicas + sg.DeltaReplicas)) - (sg.CurrentPodCPU * sg.CurrentReplicas)
		totalMemorydelta += ((sg.CurrentPodMemory + sg.DeltaMemory) * (sg.CurrentReplicas + sg.DeltaReplicas)) - (sg.CurrentPodMemory * sg.CurrentReplicas)
		totalReplicasdelta += int(sg.DeltaReplicas)
	}

	cpuFactor := 1.0
	if totalCPUdelta > ce.GlobalPolicy.NamespaceCPUQuotaUnused {
		cpuFactor = ce.GlobalPolicy.NamespaceCPUQuotaUnused / totalCPUdelta
	}

	memFactor := 1.0
	if totalMemorydelta > ce.GlobalPolicy.NamespaceMemoryQuotaUnused {
		memFactor = ce.GlobalPolicy.NamespaceMemoryQuotaUnused / totalMemorydelta
	}

	repFactor := 1.0
	if totalReplicasdelta > ce.GlobalPolicy.NamespaceReplicaQuotaUnused {
		repFactor = float64(ce.GlobalPolicy.NamespaceReplicaQuotaUnused) / float64(totalReplicasdelta)
	}

	for _, sg := range g.Genes {
		if sg == nil {
			continue
		}

		switch sg.ReactionType {
		case genome.HPA:
			strictestRepFactor := min(cpuFactor, min(memFactor, repFactor))
			sg.DeltaReplicas = math.Floor(sg.DeltaReplicas * strictestRepFactor)

		case genome.VPA:
			sg.DeltaCPU *= cpuFactor
			sg.DeltaMemory *= memFactor
		}
	}
}
