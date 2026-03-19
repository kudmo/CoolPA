package decision

import (
	"fmt"

	gaconfig "github.com/kudmo/CoolPA/decision/ga/config"
	"github.com/kudmo/CoolPA/decision/ga/constraints"
	"github.com/kudmo/CoolPA/decision/ga/engine"
	"github.com/kudmo/CoolPA/decision/ga/genome"
	slopredictor "github.com/kudmo/CoolPA/decision/slo_predictor"
	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/CoolPA/storage/metrics"
)

type ReactionOptimizer struct {
	Store *storage.Storage
}

// ScaleUp runs the GA to propose scaling decisions biased towards increases.
func (ro *ReactionOptimizer) ScaleUp(services []string, store *storage.Storage) {
	seed := int64(42)

	cfg := gaconfig.Config{
		PopulationSize:   20,
		Generations:      15,
		EliteRatio:       0.1,
		MutationRate:     0.2,
		TypeMutationRate: 0.02,
		CrossoverRate:    0.7,
		TournamentSize:   3,
		RandomSeed:       seed,
	}

	// Build service policies
	constraintsmap := map[string]constraints.ServicePolicy{}
	for _, svc := range services {
		pods := store.ResourceMetrics.GetServicePods(svc)
		cpu_quota, _, _ := store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.CPUQuota)
		constraintsmap[svc] = constraints.ServicePolicy{
			AllowedReactions: []genome.ReactionType{genome.HPA, genome.VPA_CPU},
			MinReplicas:      len(pods) + 1,
			MaxReplicas:      len(pods) + 10,
			MinCPU:           cpu_quota,
			MaxCPU:           cpu_quota * 4,
		}
	}

	ce := &constraints.ConstraintEngine{ServicePolicies: constraintsmap, GlobalPolicy: constraints.GlobalPolicy{ClusterCPULimit: 5000000.0, ClusterReplicaLimit: 100}}

	fit := slopredictor.NewResourceOptimizerFitness(store)
	fit.Config = slopredictor.ResourceOptimizerFitnessConfig{CpuMax: 1200000, ReplicasMax: 10}

	eng := &engine.Engine{Config: cfg, Fitness: fit, Constraints: ce}

	// Seed genome: choose initial reaction according to policy and bias toward increases
	seedGenome := &genome.ReactionGenome{Genes: []*genome.ServiceGene{}}
	for _, svc := range services {
		sp := ce.ServicePolicies[svc]
		rt := genome.HPA
		if len(sp.AllowedReactions) == 1 {
			rt = sp.AllowedReactions[0]
		}
		g := &genome.ServiceGene{ServiceName: svc, ReactionType: rt}
		if rt == genome.HPA {
			g.DeltaReplicas = 0.2
			g.DeltaCPU = 0
		} else {
			g.DeltaCPU = 0.05
			g.DeltaReplicas = 0
		}
		seedGenome.Genes = append(seedGenome.Genes, g)
	}

	best, err := eng.Run(seedGenome)
	if err != nil {
		fmt.Println("engine run error:", err)
		return
	}

	fmt.Println("Best genome:")
	for _, g := range best.Genes {
		fmt.Printf("- %s: reaction=%v, deltaRep=%.3f, cpu=%.3f\n", g.ServiceName, g.ReactionType, g.DeltaReplicas, g.DeltaCPU)
	}

	// Build current service states and decode candidate decisions
	current := make([]genome.ServiceState, 0, len(services))
	for _, svc := range services {
		pods := store.ResourceMetrics.GetServicePods(svc)
		cpu_quota, _, _ := store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.CPUQuota)
		current = append(current, genome.ServiceState{Name: svc, CurrentReplicas: len(pods), CurrentCPURel: cpu_quota})
	}
	cand := best.Decode(current)
	fmt.Println("Candidate state:")
	for _, s := range cand.Services {
		fmt.Printf("- %s -> replicas=%d, cpu=%.3f, reaction=%v\n", s.ServiceName, s.Replicas, s.CPURel, s.Reaction)
	}
}

// ScaleDown runs the GA to propose scaling decisions biased towards decreases.
func (ro *ReactionOptimizer) ScaleDown(services []string, store *storage.Storage) {
	seed := int64(42)

	cfg := gaconfig.Config{
		PopulationSize:   20,
		Generations:      15,
		EliteRatio:       0.1,
		MutationRate:     0.2,
		TypeMutationRate: 0.02,
		CrossoverRate:    0.7,
		TournamentSize:   3,
		RandomSeed:       seed,
	}

	constraintsmap := map[string]constraints.ServicePolicy{}
	for _, svc := range services {
		pods := store.ResourceMetrics.GetServicePods(svc)
		cpu_quota, _, _ := store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.CPUQuota)
		constraintsmap[svc] = constraints.ServicePolicy{
			AllowedReactions: []genome.ReactionType{genome.HPA, genome.VPA_CPU},
			MinReplicas:      1,
			MaxReplicas:      len(pods) + 10,
			MinCPU:           cpu_quota * 0.25,
			MaxCPU:           cpu_quota * 4,
		}
	}
	ce := &constraints.ConstraintEngine{ServicePolicies: constraintsmap, GlobalPolicy: constraints.GlobalPolicy{ClusterCPULimit: 5000000.0, ClusterReplicaLimit: 100}}

	fit := slopredictor.NewResourceOptimizerFitness(store)
	fit.Config = slopredictor.ResourceOptimizerFitnessConfig{CpuMax: 1200000, ReplicasMax: 10}

	eng := &engine.Engine{Config: cfg, Fitness: fit, Constraints: ce}

	// Seed genome preferring reductions
	seedGenome := &genome.ReactionGenome{Genes: []*genome.ServiceGene{}}
	for _, svc := range services {
		sp := ce.ServicePolicies[svc]
		rt := genome.HPA
		if len(sp.AllowedReactions) == 1 {
			rt = sp.AllowedReactions[0]
		}
		g := &genome.ServiceGene{ServiceName: svc, ReactionType: rt}
		if rt == genome.HPA {
			g.DeltaReplicas = -0.2
			g.DeltaCPU = 0
		} else {
			g.DeltaCPU = -0.05
			g.DeltaReplicas = 0
		}
		seedGenome.Genes = append(seedGenome.Genes, g)
	}

	best, err := eng.Run(seedGenome)
	if err != nil {
		fmt.Println("engine run error:", err)
		return
	}

	fmt.Println("Best genome (scale down):")
	for _, g := range best.Genes {
		fmt.Printf("- %s: reaction=%v, deltaRep=%.3f, cpu=%.3f\n", g.ServiceName, g.ReactionType, g.DeltaReplicas, g.DeltaCPU)
	}

	current := make([]genome.ServiceState, 0, len(services))
	for _, svc := range services {
		pods := store.ResourceMetrics.GetServicePods(svc)
		cpu_quota, _, _ := store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.CPUQuota)
		current = append(current, genome.ServiceState{Name: svc, CurrentReplicas: len(pods), CurrentCPURel: cpu_quota})
	}
	cand := best.Decode(current)
	fmt.Println("Candidate state (scale down):")
	for _, s := range cand.Services {
		fmt.Printf("- %s -> replicas=%d, cpu=%.3f, reaction=%v\n", s.ServiceName, s.Replicas, s.CPURel, s.Reaction)
	}
}
