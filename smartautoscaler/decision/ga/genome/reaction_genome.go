package genome

import (
	"math"
	"math/rand"
)

// ReactionType identifies the scaling reaction used for a service.
type ReactionType int

const (
	HPA ReactionType = iota
	VPA              // Vertical scaling affecting both CPU and memory
)

// ServiceGene represents a per-service atomic gene. Only one delta is active
// depending on ReactionType: DeltaReplicas for HPA, DeltaCPU and DeltaMemory for VPA.
type ServiceGene struct {
	ServiceName  string
	ReactionType ReactionType

	DeltaReplicas float64
	DeltaCPU      float64
	DeltaMemory   float64
}

// ReactionGenome encodes scaling decisions for several services.
type ReactionGenome struct {
	Genes []*ServiceGene
}

// Clone returns a deep copy of the genome.
func (g *ReactionGenome) Clone() *ReactionGenome {
	ng := &ReactionGenome{Genes: make([]*ServiceGene, len(g.Genes))}
	for i, sg := range g.Genes {
		if sg == nil {
			continue
		}
		cp := *sg
		ng.Genes[i] = &cp
	}
	return ng
}

// Mutate performs reaction-aware mutation.
// constraints is an accessor that provides allowed reactions and repair/validate methods.
func (g *ReactionGenome) Mutate(rng *rand.Rand, mutationRate, typeMutationRate float64, constraints interface {
	GetAllowedReactions(service string) []ReactionType
	Repair(*ReactionGenome)
	Validate(*ReactionGenome) bool
}) {
	for _, gene := range g.Genes {
		if gene == nil {
			continue
		}

		// Type mutation: possibly switch reaction type while respecting allowed reactions.
		if rng.Float64() < typeMutationRate {
			// query allowed reactions
			var allowed []ReactionType
			if constraints != nil {
				allowed = constraints.GetAllowedReactions(gene.ServiceName)
			}
			// choose a new reaction from allowed set (or both if empty)
			var candidates []ReactionType
			if len(allowed) == 0 {
				candidates = []ReactionType{HPA, VPA}
			} else {
				candidates = append(candidates, allowed...)
			}
			if len(candidates) > 0 {
				newType := candidates[rng.Intn(len(candidates))]
				if newType != gene.ReactionType {
					gene.ReactionType = newType
					// after type change reset all deltas
					gene.DeltaReplicas = 0
					gene.DeltaCPU = 0
					gene.DeltaMemory = 0
				}
			}
		}

		// Mutate only the active parameters
		if rng.Float64() < mutationRate {
			switch gene.ReactionType {
			case HPA:
				gene.DeltaReplicas += rng.NormFloat64() * 0.1
				gene.DeltaCPU = 0
				gene.DeltaMemory = 0
			case VPA:
				gene.DeltaCPU += rng.NormFloat64() * 0.05
				gene.DeltaMemory += rng.NormFloat64() * 0.05
				gene.DeltaReplicas = 0
			}
		} else {
			// ensure inactive params are zeroed
			switch gene.ReactionType {
			case HPA:
				gene.DeltaCPU = 0
				gene.DeltaMemory = 0
			case VPA:
				gene.DeltaReplicas = 0
			}
		}

		// clamp NaNs/Infs and keep numerically stable
		if math.IsNaN(gene.DeltaReplicas) || math.IsInf(gene.DeltaReplicas, 0) {
			gene.DeltaReplicas = 0
		}
		if math.IsNaN(gene.DeltaCPU) || math.IsInf(gene.DeltaCPU, 0) {
			gene.DeltaCPU = 0
		}
		if math.IsNaN(gene.DeltaMemory) || math.IsInf(gene.DeltaMemory, 0) {
			gene.DeltaMemory = 0
		}
	}

	// allow constraint engine to repair
	if constraints != nil {
		constraints.Repair(g)
		_ = constraints.Validate(g)
	}
}

// Crossover performs service-level atomic crossover between two parents and returns two children.
func (g *ReactionGenome) Crossover(other *ReactionGenome, rng *rand.Rand) (*ReactionGenome, *ReactionGenome) {
	n := len(g.Genes)
	if other == nil || len(other.Genes) != n {
		return g.Clone(), other.Clone()
	}
	c1 := &ReactionGenome{Genes: make([]*ServiceGene, n)}
	c2 := &ReactionGenome{Genes: make([]*ServiceGene, n)}
	for i := 0; i < n; i++ {
		if rng.Float64() < 0.5 {
			if g.Genes[i] != nil {
				c1.Genes[i] = copyGene(g.Genes[i])
			}
			if other.Genes[i] != nil {
				c2.Genes[i] = copyGene(other.Genes[i])
			}
		} else {
			if other.Genes[i] != nil {
				c1.Genes[i] = copyGene(other.Genes[i])
			}
			if g.Genes[i] != nil {
				c2.Genes[i] = copyGene(g.Genes[i])
			}
		}
	}
	return c1, c2
}

func copyGene(s *ServiceGene) *ServiceGene {
	if s == nil {
		return nil
	}
	cp := *s
	return &cp
}

// ServiceState represents the current service runtime state (a minimal view needed for decode).
type ServiceState struct {
	Name             string
	CurrentReplicas  int
	CurrentCPURel    float64
	CurrentMemoryRel float64
}

// CandidateServiceState is a decoded proposed state for a single service.
type CandidateServiceState struct {
	ServiceName string
	Replicas    int
	CPURel      float64
	MemoryRel   float64
	Reaction    ReactionType
}

// CandidateState aggregates decoded per-service candidate states.
type CandidateState struct {
	Services []CandidateServiceState
}

// Decode applies the genome deltas to the provided current state to produce a candidate state.
func (g *ReactionGenome) Decode(currentStates []ServiceState) CandidateState {
	idx := make(map[string]*ServiceGene)
	for _, sg := range g.Genes {
		if sg != nil {
			idx[sg.ServiceName] = sg
		}
	}
	out := CandidateState{Services: make([]CandidateServiceState, 0, len(currentStates))}
	for _, cs := range currentStates {
		var cand CandidateServiceState
		cand.ServiceName = cs.Name
		if sg, ok := idx[cs.Name]; ok {
			switch sg.ReactionType {
			case HPA:
				newRepF := float64(cs.CurrentReplicas) + sg.DeltaReplicas
				cand.Replicas = int(math.Max(0, math.Round(newRepF)))
				cand.CPURel = cs.CurrentCPURel
				cand.MemoryRel = cs.CurrentMemoryRel
			case VPA:
				cand.CPURel = math.Max(0, cs.CurrentCPURel+sg.DeltaCPU)
				cand.MemoryRel = math.Max(0, cs.CurrentMemoryRel+sg.DeltaMemory)
				cand.Replicas = cs.CurrentReplicas
			default:
				cand.Replicas = cs.CurrentReplicas
				cand.CPURel = cs.CurrentCPURel
				cand.MemoryRel = cs.CurrentMemoryRel
			}
			cand.Reaction = sg.ReactionType
		} else {
			// No decision -> keep current
			cand.Replicas = cs.CurrentReplicas
			cand.CPURel = cs.CurrentCPURel
			cand.MemoryRel = cs.CurrentMemoryRel
			cand.Reaction = HPA
		}
		out.Services = append(out.Services, cand)
	}
	return out
}
