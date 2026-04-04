package decision

import (
	"context"
	"fmt"
	"log/slog"

	gaconfig "github.com/kudmo/CoolPA/decision/ga/config"
	"github.com/kudmo/CoolPA/decision/ga/constraints"
	"github.com/kudmo/CoolPA/decision/ga/engine"
	"github.com/kudmo/CoolPA/decision/ga/genome"
	slopredictor "github.com/kudmo/CoolPA/decision/slo_predictor"
	reactionapplier "github.com/kudmo/CoolPA/reaction_applier"
	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/CoolPA/storage/metrics"
	"github.com/kudmo/CoolPA/storage/quotas"
)

type ScaleMode int

const (
	ScaleUpMode ScaleMode = iota
	ScaleDownMode
)

type ReactionOptimizer struct {
	store   *storage.Storage
	applier reactionapplier.Applier
}

func NewReactionOptimizer(store *storage.Storage, applier reactionapplier.Applier) *ReactionOptimizer {
	return &ReactionOptimizer{
		store:   store,
		applier: applier,
	}
}

func (ro *ReactionOptimizer) ScaleUp(services []string) {
	ro.runOptimization(services, ScaleUpMode)
}

func (ro *ReactionOptimizer) ScaleDown(services []string) {
	ro.runOptimization(services, ScaleDownMode)
}

func (ro *ReactionOptimizer) runOptimization(
	services []string,
	mode ScaleMode,
) {
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

	constraintsMap := ro.buildConstraints(services, ro.store, mode)

	ce := &constraints.ConstraintEngine{
		ServicePolicies: constraintsMap,
		GlobalPolicy: constraints.GlobalPolicy{
			ClusterCPULimit:     float64(ro.store.Limits.NamespaceLimits[quotas.MaxCpu]),
			ClusterMemoryLimit:  float64(ro.store.Limits.NamespaceLimits[quotas.MaxMem]),
			ClusterReplicaLimit: int(ro.store.Limits.NamespaceLimits[quotas.MaxPods]),
		},
	}

	fit := slopredictor.NewResourceOptimizerFitness(ro.store)

	fit.Config = slopredictor.ResourceOptimizerFitnessConfig{
		CpuMax:      float64(ro.store.Limits.NamespaceLimits[quotas.MaxCpu]),
		MemMax:      float64(ro.store.Limits.NamespaceLimits[quotas.MaxMem]),
		ReplicasMax: float64(ro.store.Limits.NamespaceLimits[quotas.MaxPods]),
	}

	eng := &engine.Engine{
		Config:      cfg,
		Fitness:     fit,
		Constraints: ce,
	}

	seedGenome := ro.buildSeedGenome(services, ce, mode)

	best, err := eng.Run(seedGenome)
	if err != nil {
		slog.Error("GA engine run error", "error", err)
		return
	}

	ro.logGenome(best)

	cand := best.Decode()

	ro.applyCandidate(context.Background(), &cand)
}

func (ro *ReactionOptimizer) choosePossibleReactions(service string) []genome.ReactionType {
	return []genome.ReactionType{genome.HPA, genome.VPA}
}

func (ro *ReactionOptimizer) buildConstraints(
	services []string,
	store *storage.Storage,
	mode ScaleMode,
) map[string]constraints.ServicePolicy {

	result := map[string]constraints.ServicePolicy{}

	for _, svc := range services {
		pods := store.ResourceMetrics.GetServicePods(svc)
		if len(pods) == 0 {
			slog.Warn("No pods found for service", "service", svc)
			continue
		}

		cpu, _, _ := store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.CPUQuota)
		mem, _, _ := store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.MemoryLimit)

		policy := constraints.ServicePolicy{
			AllowedReactions: ro.choosePossibleReactions(svc),
		}

		replicasMinStep := 1
		cpuMinStep := 100.0
		memMinStep := 128.0
		if mode == ScaleUpMode {
			policy.MaxReplicas = len(pods) + 10
			policy.MinReplicas = len(pods) + replicasMinStep
			policy.MaxCPU = float64(ro.store.Limits.ServiceLimits[quotas.ServiceMaxCpu])
			policy.MinCPU = cpu + cpuMinStep
			policy.MaxMemory = float64(ro.store.Limits.ServiceLimits[quotas.ServiceMaxMem])
			policy.MinMemory = mem + memMinStep
		} else {
			policy.MaxReplicas = len(pods) - replicasMinStep
			policy.MinReplicas = max(1, len(pods)-10)
			policy.MaxCPU = cpu - cpuMinStep
			policy.MinCPU = float64(ro.store.Limits.ServiceLimits[quotas.ServiceMinCpu])
			policy.MaxMemory = mem - memMinStep
			policy.MinMemory = float64(ro.store.Limits.ServiceLimits[quotas.ServiceMinMem])
		}

		result[svc] = policy
	}

	return result
}

func (ro *ReactionOptimizer) buildSeedGenome(
	services []string,
	ce *constraints.ConstraintEngine,
	mode ScaleMode,
) *genome.ReactionGenome {

	seed := &genome.ReactionGenome{Genes: []*genome.ServiceGene{}}

	for _, svc := range services {
		sp, exists := ce.ServicePolicies[svc]
		if !exists {
			continue
		}

		pods := ro.store.ResourceMetrics.GetServicePods(svc)
		if len(pods) == 0 {
			slog.Warn("No pods found for service", "service", svc)
			continue
		}

		cpu, _, _ := ro.store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.CPUQuota)
		mem, _, _ := ro.store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.MemoryLimit)

		rt := genome.HPA
		if len(sp.AllowedReactions) == 1 {
			rt = sp.AllowedReactions[0]
		}

		g := &genome.ServiceGene{
			ServiceName:     svc,
			ReactionType:    rt,
			CurrentReplicas: float64(len(pods)),
			CurrentCPU:      cpu,
			CurrentMemory:   mem,
		}

		sign := 1.0
		if mode == ScaleDownMode {
			sign = -1.0
		}

		switch rt {
		case genome.HPA:
			g.DeltaReplicas = 2.0 * sign
			g.DeltaCPU = 0
			g.DeltaMemory = 0
		case genome.VPA:
			g.DeltaReplicas = 0
			g.DeltaCPU = 500.0 * sign
			g.DeltaMemory = 512.0 * sign
		}

		seed.Genes = append(seed.Genes, g)
	}

	return seed
}

func (ro *ReactionOptimizer) logGenome(g *genome.ReactionGenome) {
	for _, gene := range g.Genes {
		slog.Debug("Best genome",
			"service", gene.ServiceName,
			"reaction", gene.ReactionType,
			"delta replicas", gene.DeltaReplicas,
			"delta cpu", gene.DeltaCPU,
			"delta memory", gene.DeltaMemory,
		)
	}
}

func (ro *ReactionOptimizer) applyCandidate(
	ctx context.Context,
	cand *genome.CandidateState,
) {
	namespace := "microservices-demo"
	if ro.applier == nil {
		slog.Error("Failed apply reaction: applier is nil")
		return
	}

	for _, s := range cand.Services {
		slog.Info("Applying candidate",
			"service", s.ServiceName,
			"reaction", s.Reaction,
			"replicas", s.Replicas,
			"cpu", s.CPURel,
			"memory", s.MemoryRel,
		)

		switch s.Reaction {
		case genome.HPA:
			if err := ro.applier.ApplyHPS(ctx, namespace, s.ServiceName, int32(s.Replicas)); err != nil {
				slog.Error("Failed to apply HPA", "service", s.ServiceName, "error", err)
			}
		case genome.VPA:
			cpuStr := fmt.Sprintf("%dm", int(s.CPURel/1000))
			memStr := fmt.Sprintf("%dMi", int(s.MemoryRel/(1024*1024)))
			if err := ro.applier.ApplyVPS(ctx, namespace, s.ServiceName, cpuStr, memStr); err != nil {
				slog.Error("Failed to apply VPA", "service", s.ServiceName, "error", err)
			}
		}
	}
}
