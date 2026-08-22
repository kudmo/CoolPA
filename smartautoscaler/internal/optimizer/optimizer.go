package optimizer

import (
	"context"
	"math"
	"time"

	"github.com/kudmo/CoolPA/logger"
	"github.com/kudmo/CoolPA/utils"

	gaconfig "github.com/kudmo/CoolPA/internal/optimizer/ga/config"
	"github.com/kudmo/CoolPA/internal/optimizer/ga/constraints"
	"github.com/kudmo/CoolPA/internal/optimizer/ga/engine"
	"github.com/kudmo/CoolPA/internal/optimizer/ga/genome"
	"github.com/kudmo/CoolPA/internal/optimizer/interfaces"
	slopredictor "github.com/kudmo/CoolPA/internal/optimizer/slo_predictor"
	"github.com/kudmo/CoolPA/internal/statistics"
)

type ReactionOptimizer struct {
	metricsProvider interfaces.MetricsRepository
	histStore       *statistics.HistStore

	config ReactionOptimizerConfig
}

func NewReactionOptimizer(metricsProvider interfaces.MetricsRepository, histStore *statistics.HistStore, config ReactionOptimizerConfig) *ReactionOptimizer {
	return &ReactionOptimizer{
		metricsProvider: metricsProvider,
		histStore:       histStore,
		config:          config,
	}
}

func (ro *ReactionOptimizer) RunOptimization(ctx context.Context, services []string, mode ScaleMode) (OptimizedState, error) {
	seed := int64(time.Now().Unix())

	cfg := gaconfig.Config{
		PopulationSize:   20,
		Generations:      15,
		EliteRatio:       0.05,
		MutationRate:     0.6,
		TypeMutationRate: 0.1,
		CrossoverRate:    0.7,
		TournamentSize:   3,
		RandomSeed:       seed,
	}

	constraintsMap := ro.buildConstraints(ctx, services, mode)

	cpu_sum := 0.0
	mem_sum := 0.0
	replics_sum := 0

	for _, svc := range services {
		replicas_count, _ := ro.metricsProvider.GetServiceReplicasCountValue(ctx, svc)
		cpu_pod_quota, _ := ro.metricsProvider.GetServicePodCpuQuota(ctx, svc)
		mem_pod_quota, _ := ro.metricsProvider.GetServicePodMemoryQuota(ctx, svc)
		cpu_sum += cpu_pod_quota
		mem_sum += mem_pod_quota
		replics_sum += int(replicas_count)
	}

	total_cpu_limit, _ := ro.metricsProvider.GetGlobalTotalCpuLimit(ctx)
	total_mem_limit, _ := ro.metricsProvider.GetGlobalTotalMemoryLimit(ctx)
	total_replicas_limit, _ := ro.metricsProvider.GetGlobalTotalPodsLimit(ctx)

	ce := &constraints.ConstraintEngine{
		ServicePolicies: constraintsMap,
		GlobalPolicy: constraints.GlobalPolicy{
			NamespaceCPUQuotaUnused:     total_cpu_limit - cpu_sum,
			NamespaceMemoryQuotaUnused:  total_mem_limit - mem_sum,
			NamespaceReplicaQuotaUnused: int(total_replicas_limit) - replics_sum,
		},
	}

	fit := slopredictor.NewResourceOptimizerFitness(
		ro.metricsProvider,
		ro.histStore,
		slopredictor.FitnessConfig{Lambda: ro.config.Lambda, Window: time.Duration(time.Minute)},
	)
	defer fit.Close()

	eng := &engine.Engine{
		Config:      cfg,
		Fitness:     fit,
		Constraints: ce,
	}

	seedGenome := ro.buildSeedGenome(ctx, services, ce, mode)

	best, err := eng.Run(ctx, time.Now(), seedGenome)
	if err != nil {
		logger.Error("optimizer", "ga engine run error", "error", err)
		return OptimizedState{}, err
	}

	ro.logGenome(best)

	cand := best.Decode()

	state := OptimizedState{Services: make([]OptimizedServiceState, len(cand.Services))}

	for _, service := range cand.Services {
		state.Services = append(state.Services,
			OptimizedServiceState{
				ServiceName: service.ServiceName,
				Replicas:    service.Replicas,
				AppCPU:      service.AppCPU,
				PodCPU:      service.PodCPU,
				AppMemory:   service.AppMemory,
				PodMemory:   service.PodMemory,
				Reaction:    ReactionType(service.Reaction),
			},
		)
	}
	return state, nil
}

func (ro *ReactionOptimizer) choosePossibleReactions(service string) []genome.ReactionType {
	return []genome.ReactionType{genome.HPA /*, genome.VPA*/}
}

func (ro *ReactionOptimizer) buildConstraints(
	ctx context.Context,
	services []string,
	mode ScaleMode,
) map[string]constraints.ServicePolicy {

	result := map[string]constraints.ServicePolicy{}

	for _, svc := range services {
		replicas_count, _ := ro.metricsProvider.GetServiceReplicasCountValue(ctx, svc)
		if replicas_count == 0 {
			logger.Warn("optimizer", "no pods found for service", "service", svc)
			continue
		}

		cpu, _ := ro.metricsProvider.GetServiceCpuQuota(ctx, svc)
		mem, _ := ro.metricsProvider.GetServiceMemoryQuota(ctx, svc)

		policy := constraints.ServicePolicy{
			AllowedReactions: ro.choosePossibleReactions(svc),
		}

		if mode == ScaleUpMode {
			time_now := time.Now()
			time_now_begin := time_now.Add(-time.Minute)
			cpuUsageRange, _ := ro.metricsProvider.GetServiceCpuUsageRange(ctx, svc, time_now_begin, time_now)
			cpuU := utils.Avg(cpuUsageRange)

			desiredReplicas := int(math.Ceil(
				replicas_count * (cpuU / ro.config.TargetCpuUtilization),
			))
			policy.MinReplicas = int(replicas_count) + ro.config.ReplicasStep
			policy.MaxReplicas = max(policy.MinReplicas, min(int(replicas_count)+10, desiredReplicas+1))
			policy.MaxCPU, _ = ro.metricsProvider.GetGlobalServiceMaxCpu(ctx)
			policy.MinCPU = cpu + ro.config.CpuStep
			policy.MaxMemory, _ = ro.metricsProvider.GetGlobalServiceMaxMemory(ctx)
			policy.MinMemory = mem + ro.config.MemoryStep
		} else {
			policy.MaxReplicas = max(1, int(replicas_count)-ro.config.ReplicasStep)
			policy.MinReplicas = max(1, int(replicas_count)-ro.config.ReplicasStep)
			policy.MaxCPU = cpu - ro.config.CpuStep
			policy.MinCPU, _ = ro.metricsProvider.GetGlobalServiceMinCpu(ctx)
			policy.MaxMemory = mem - ro.config.MemoryStep
			policy.MinMemory, _ = ro.metricsProvider.GetGlobalServiceMinMemory(ctx)
		}

		result[svc] = policy
	}

	return result
}

func (ro *ReactionOptimizer) buildSeedGenome(
	ctx context.Context,
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

		replicas_count, _ := ro.metricsProvider.GetServiceReplicasCountValue(ctx, svc)
		if replicas_count == 0 {
			logger.Warn("optimizer", "no pods found for service", "service", svc)
			continue
		}

		cpu_app_quota, _ := ro.metricsProvider.GetServiceCpuQuota(ctx, svc)
		cpu_pod_quota, _ := ro.metricsProvider.GetServicePodCpuQuota(ctx, svc)
		mem_app_quota, _ := ro.metricsProvider.GetServiceMemoryQuota(ctx, svc)
		mem_pod_quota, _ := ro.metricsProvider.GetServicePodMemoryQuota(ctx, svc)

		rt := genome.HPA
		if len(sp.AllowedReactions) == 1 {
			rt = sp.AllowedReactions[0]
		}

		g := &genome.ServiceGene{
			ServiceName:      svc,
			ReactionType:     rt,
			CurrentReplicas:  replicas_count,
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
		logger.Debug("optimizer", "best genome",
			"service", gene.ServiceName,
			"reaction", gene.ReactionType,
			"delta replicas", gene.DeltaReplicas,
			"delta cpu", gene.DeltaCPU,
			"delta memory", gene.DeltaMemory,
		)
	}
}
