package constraints

import (
	"errors"
	"math"
	"sort"

	"github.com/kudmo/CoolPA/decision/ga/genome"
)

// ApplyGlobalScalingIfExceeded inspects a genome and applies conservative scaling to respect global limits.
func (ce *ConstraintEngine) ApplyGlobalScalingIfExceeded(g *genome.ReactionGenome) error {
	if ce == nil || g == nil {
		return errors.New("nil input")
	}
	// Compute cluster totals using conservative baseline assumptions.
	totalCPU := 0.0
	totalReplicas := 0
	for _, sg := range g.Genes {
		if sg == nil {
			continue
		}
		baseRep := 1
		baseCPU := 1.0
		newRep := int(math.Max(0, math.Round(float64(baseRep)*(1.0+sg.DeltaReplicas))))
		totalReplicas += newRep
		totalCPU += baseCPU * (1.0 + sg.DeltaCPU)
	}

	// If exceed cluster limits, apply simple proportional reductions.
	if ce.GlobalPolicy.ClusterCPULimit > 0 && totalCPU > ce.GlobalPolicy.ClusterCPULimit {
		factor := ce.GlobalPolicy.ClusterCPULimit / totalCPU
		for _, sg := range g.Genes {
			if sg == nil {
				continue
			}
			sg.DeltaCPU = (sg.DeltaCPU) * factor
		}
	}
	if ce.GlobalPolicy.ClusterReplicaLimit > 0 && totalReplicas > ce.GlobalPolicy.ClusterReplicaLimit {
		// reduce replicas of services with largest replica proposals first
		type idxVal struct{ i, val int }
		arr := make([]idxVal, 0, len(g.Genes))
		for i, sg := range g.Genes {
			if sg == nil {
				continue
			}
			rp := int(math.Max(0, math.Round(float64(1)*(1.0+sg.DeltaReplicas))))
			arr = append(arr, idxVal{i: i, val: rp})
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].val > arr[j].val })
		excess := totalReplicas - ce.GlobalPolicy.ClusterReplicaLimit
		for _, e := range arr {
			if excess <= 0 {
				break
			}
			sg := g.Genes[e.i]
			// pull delta down by at most 50% per step
			sg.DeltaReplicas -= 0.5
			excess -= e.val / 2
		}
	}
	return nil
}
