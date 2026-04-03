package constraints

import (
	"errors"
	"math"
	"sort"

	"github.com/kudmo/CoolPA/decision/ga/genome"
)

func (ce *ConstraintEngine) ApplyGlobalScalingIfExceeded(g *genome.ReactionGenome) error {
	if ce == nil || g == nil {
		return errors.New("nil input")
	}

	totalCPU := 0.0
	totalMemory := 0.0
	totalReplicas := 0

	for _, sg := range g.Genes {
		if sg == nil {
			continue
		}

		sp, hasPolicy := ce.ServicePolicies[sg.ServiceName]

		currentReplicas := sg.CurrentReplicas
		if currentReplicas <= 0 {
			currentReplicas = float64(ce.ServicePolicies[sg.ServiceName].MinReplicas)
		}

		currentCPU := sg.CurrentCPU
		if currentCPU <= 0 {
			currentCPU = ce.ServicePolicies[sg.ServiceName].MinCPU
		}

		currentMemory := sg.CurrentMemory
		if currentMemory <= 0 {
			currentMemory = ce.ServicePolicies[sg.ServiceName].MinMemory
		}

		var newRep int
		var newCPU, newMemory float64

		switch sg.ReactionType {
		case genome.HPA:
			newRep = int(math.Max(0, math.Round(currentReplicas+sg.DeltaReplicas)))
			if hasPolicy && sp.MaxReplicas > 0 && newRep > sp.MaxReplicas {
				newRep = sp.MaxReplicas
			}
			if hasPolicy && sp.MinReplicas >= 0 && newRep < sp.MinReplicas {
				newRep = sp.MinReplicas
			}
			newCPU = currentCPU
			newMemory = currentMemory
		case genome.VPA:
			newCPU = math.Max(0, currentCPU+sg.DeltaCPU)
			newMemory = math.Max(0, currentMemory+sg.DeltaMemory)
			if hasPolicy {
				if sp.MaxCPU > 0 && newCPU > sp.MaxCPU {
					newCPU = sp.MaxCPU
				}
				if sp.MinCPU >= 0 && newCPU < sp.MinCPU {
					newCPU = sp.MinCPU
				}
				if sp.MaxMemory > 0 && newMemory > sp.MaxMemory {
					newMemory = sp.MaxMemory
				}
				if sp.MinMemory >= 0 && newMemory < sp.MinMemory {
					newMemory = sp.MinMemory
				}
			}
			newRep = int(currentReplicas)
		default:
			newRep = int(currentReplicas)
			newCPU = currentCPU
			newMemory = currentMemory
		}

		totalReplicas += newRep
		totalCPU += newCPU
		totalMemory += newMemory
	}

	if ce.GlobalPolicy.ClusterCPULimit > 0 && totalCPU > ce.GlobalPolicy.ClusterCPULimit {
		factor := ce.GlobalPolicy.ClusterCPULimit / totalCPU
		for _, sg := range g.Genes {
			if sg == nil {
				continue
			}
			switch sg.ReactionType {
			case genome.VPA:
				sg.DeltaCPU = (sg.CurrentCPU+sg.DeltaCPU)*factor - sg.CurrentCPU
				sg.DeltaMemory = (sg.CurrentMemory+sg.DeltaMemory)*factor - sg.CurrentMemory
			}
		}
	}

	if ce.GlobalPolicy.ClusterMemoryLimit > 0 && totalMemory > ce.GlobalPolicy.ClusterMemoryLimit {
		factor := ce.GlobalPolicy.ClusterMemoryLimit / totalMemory
		for _, sg := range g.Genes {
			if sg == nil {
				continue
			}
			switch sg.ReactionType {
			case genome.VPA:
				sg.DeltaMemory = (sg.CurrentMemory+sg.DeltaMemory)*factor - sg.CurrentMemory
				sg.DeltaCPU = (sg.CurrentCPU+sg.DeltaCPU)*factor - sg.CurrentCPU
			}
		}
	}

	if ce.GlobalPolicy.ClusterReplicaLimit > 0 && totalReplicas > ce.GlobalPolicy.ClusterReplicaLimit {
		type idxVal struct{ i, val int }
		arr := make([]idxVal, 0, len(g.Genes))
		for i, sg := range g.Genes {
			if sg == nil {
				continue
			}
			var rep int
			if sg.ReactionType == genome.HPA {
				rep = int(math.Max(0, math.Round(sg.CurrentReplicas+sg.DeltaReplicas)))
			} else {
				rep = int(sg.CurrentReplicas)
			}
			arr = append(arr, idxVal{i: i, val: rep})
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].val > arr[j].val })

		excess := totalReplicas - ce.GlobalPolicy.ClusterReplicaLimit
		for _, e := range arr {
			if excess <= 0 {
				break
			}
			sg := g.Genes[e.i]
			if sg.ReactionType == genome.HPA {
				reduction := math.Min(float64(excess), float64(e.val)/2)
				sg.DeltaReplicas -= reduction
				excess -= int(reduction)
			}
		}
	}

	return nil
}
