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

func (ro *ReactionOptimizer) ScaleUp(services []string, store *storage.Storage) {
	ro.runOptimization(services, store, ScaleUpMode)
}

func (ro *ReactionOptimizer) ScaleDown(services []string, store *storage.Storage) {
	ro.runOptimization(services, store, ScaleDownMode)
}

func (ro *ReactionOptimizer) runOptimization(
	services []string,
	store *storage.Storage,
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

	constraintsMap := ro.buildConstraints(services, store, mode)

	ce := &constraints.ConstraintEngine{
		ServicePolicies: constraintsMap,
		GlobalPolicy: constraints.GlobalPolicy{
			ClusterCPULimit:     5000000.0,
			ClusterReplicaLimit: 100,
		},
	}

	fit := slopredictor.NewResourceOptimizerFitness(store)
	fit.Config = slopredictor.ResourceOptimizerFitnessConfig{
		CpuMax:      1200000,
		ReplicasMax: 10,
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

	current := ro.buildCurrentState(services, store)
	cand := best.Decode(current)

	ro.applyCandidate(context.Background(), &cand)
}

func (ro *ReactionOptimizer) buildConstraints(
	services []string,
	store *storage.Storage,
	mode ScaleMode,
) map[string]constraints.ServicePolicy {

	result := map[string]constraints.ServicePolicy{}

	for _, svc := range services {
		pods := store.ResourceMetrics.GetServicePods(svc)
		cpu, _, _ := store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.CPUQuota)

		policy := constraints.ServicePolicy{
			AllowedReactions: []genome.ReactionType{genome.HPA /*, genome.VPA_CPU*/},
			MaxReplicas:      len(pods) + 10,
			MaxCPU:           cpu * 4,
		}

		if mode == ScaleUpMode {
			policy.MinReplicas = len(pods) + 1
			policy.MinCPU = cpu
		} else {
			policy.MinReplicas = 1
			policy.MinCPU = cpu * 0.25
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
		sp := ce.ServicePolicies[svc]

		rt := genome.HPA
		if len(sp.AllowedReactions) == 1 {
			rt = sp.AllowedReactions[0]
		}

		g := &genome.ServiceGene{
			ServiceName:  svc,
			ReactionType: rt,
		}

		sign := 1.0
		if mode == ScaleDownMode {
			sign = -1.0
		}

		if rt == genome.HPA {
			g.DeltaReplicas = 0.2 * sign
		} else {
			g.DeltaCPU = 0.05 * sign
		}

		seed.Genes = append(seed.Genes, g)
	}

	return seed
}

func (ro *ReactionOptimizer) buildCurrentState(
	services []string,
	store *storage.Storage,
) []genome.ServiceState {

	result := make([]genome.ServiceState, 0, len(services))

	for _, svc := range services {
		pods := store.ResourceMetrics.GetServicePods(svc)
		cpu, _, _ := store.ResourceMetrics.GetPodMetricHeadValue(svc, pods[0], metrics.CPUQuota)

		result = append(result, genome.ServiceState{
			Name:            svc,
			CurrentReplicas: len(pods),
			CurrentCPURel:   cpu,
		})
	}

	return result
}

func (ro *ReactionOptimizer) logGenome(g *genome.ReactionGenome) {
	for _, gene := range g.Genes {
		slog.Debug("Best genome",
			"service", gene.ServiceName,
			"reaction", gene.ReactionType,
			"delta replicas", gene.DeltaReplicas,
			"delta cpu", gene.DeltaCPU,
		)
	}
}

func (ro *ReactionOptimizer) applyCandidate(
	ctx context.Context,
	cand *genome.CandidateState,
) {
	// TODO: MAKE CONFIGURABLE
	namespace := "microservices-demo"
	if ro.applier == nil {
		slog.Error("Failed apply reaction: applyer is nil")
		return
	}
	for _, s := range cand.Services {

		slog.Info("Candidate",
			"service", s.ServiceName,
			"replicas", s.Replicas,
			"cpu", s.CPURel,
		)

		switch s.Reaction {

		case genome.HPA:
			_ = ro.applier.ApplyHPS(ctx, namespace, s.ServiceName, int32(s.Replicas))

		case genome.VPA_CPU:
			cpuStr := fmt.Sprintf("%dm", int(s.CPURel/100))
			_ = ro.applier.ApplyVPS(ctx, namespace, s.ServiceName, cpuStr, "")
		}
	}
}
