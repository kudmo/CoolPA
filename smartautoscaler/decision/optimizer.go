package decision

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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
	seed := int64(time.Now().Unix())

	cfg := gaconfig.Config{
		PopulationSize:   10,
		Generations:      8,
		EliteRatio:       0.05,
		MutationRate:     0.8,
		TypeMutationRate: 0.1,
		CrossoverRate:    0.7,
		TournamentSize:   3,
		RandomSeed:       seed,
	}

	constraintsMap := ro.buildConstraints(services, ro.store, mode)

	cpu_sum := 0.0
	mem_sum := 0.0
	replics_sum := 0

	for _, svc := range ro.store.ResourceMetrics.GetServices() {
		pods := ro.store.ResourceMetrics.GetServicePods(svc)
		cpu_pod_quota, _, _ := ro.store.ResourceMetrics.GetServiceMetricAvgHead(svc, metrics.PodCPUQuota)
		mem_pod_quota, _, _ := ro.store.ResourceMetrics.GetServiceMetricAvgHead(svc, metrics.PodMemoryLimit)
		cpu_sum += cpu_pod_quota
		mem_sum += mem_pod_quota
		replics_sum += len(pods)
	}
	ce := &constraints.ConstraintEngine{
		ServicePolicies: constraintsMap,
		GlobalPolicy: constraints.GlobalPolicy{
			NamespaceCPUQuotaUnused:     float64(ro.store.Limits.NamespaceLimits[quotas.NamespaceMaxCpu]) - cpu_sum,
			NamespaceMemoryQuotaUnused:  float64(ro.store.Limits.NamespaceLimits[quotas.NamespaceMaxMem]) - mem_sum,
			NamespaceReplicaQuotaUnused: int(ro.store.Limits.NamespaceLimits[quotas.NamespaceMaxPods]) - replics_sum,
		},
	}

	fit := slopredictor.NewResourceOptimizerFitness(ro.store)
	defer fit.Close()

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
	return []genome.ReactionType{genome.HPA /*, genome.VPA*/}
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

		cpu_app_quota, _, _ := ro.store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.CPUQuota)
		cpu_pod_quota, _, _ := ro.store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.PodCPUQuota)
		mem_app_quota, _, _ := ro.store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.MemoryLimit)
		mem_pod_quota, _, _ := ro.store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.PodMemoryLimit)

		rt := genome.HPA
		if len(sp.AllowedReactions) == 1 {
			rt = sp.AllowedReactions[0]
		}

		g := &genome.ServiceGene{
			ServiceName:      svc,
			ReactionType:     rt,
			CurrentReplicas:  float64(len(pods)),
			CurrentAppCPU:    cpu_app_quota,
			CurrentPodCPU:    cpu_pod_quota,
			CurrentAppMemory: mem_app_quota,
			CurrentPodMemory: mem_pod_quota,
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
			"cpu", s.AppCPU,
			"memory", s.AppMemory,
		)

		switch s.Reaction {
		case genome.HPA:
			if err := ro.applier.ApplyHPS(ctx, namespace, s.ServiceName, int32(s.Replicas)); err != nil {
				slog.Error("Failed to apply HPA", "service", s.ServiceName, "error", err)
			}
		case genome.VPA:
			cpuStr := fmt.Sprintf("%dm", int(s.AppCPU))
			memStr := fmt.Sprintf("%dMi", int(s.AppMemory))
			if err := ro.applier.ApplyVPS(ctx, namespace, s.ServiceName, cpuStr, memStr); err != nil {
				slog.Error("Failed to apply VPA", "service", s.ServiceName, "error", err)
			}
		}
	}
}
