package scaler

import (
	"context"
	"fmt"
	"time"

	contextutil "github.com/kudmo/CoolPA/context"
	"github.com/kudmo/CoolPA/internal/analyzer"
	"github.com/kudmo/CoolPA/internal/applier"
	"github.com/kudmo/CoolPA/internal/metrics"
	"github.com/kudmo/CoolPA/internal/optimizer"
	"github.com/kudmo/CoolPA/internal/statistics"
	"github.com/kudmo/CoolPA/logger"
)

type Scaler struct {
	stopChan         chan struct{}
	config           ScalerConfig
	isRunning        bool
	lastReactionTime time.Time

	metricsProvider metrics.MetricsRepository
	histStore       *statistics.HistStore

	analyzer        *analyzer.Analyzer
	optimizer       *optimizer.ReactionOptimizer
	reactionApplier applier.Applier
}

func NewScaler(config ScalerConfig, metricsProvider metrics.MetricsRepository, applier applier.Applier) *Scaler {
	histStore := &statistics.HistStore{}
	return &Scaler{
		stopChan:        make(chan struct{}),
		metricsProvider: metricsProvider,
		config:          config,
		histStore:       histStore,
		reactionApplier: applier,
		analyzer: analyzer.NewAnalyzer(
			analyzer.AnalyzerConfig{
				SLO:                  config.SLO,
				Confidence:           0.05,
				Window:               time.Duration(60 * time.Second),
				AnomalyServicesCount: config.AnomalyServicesCount,
				Alpha:                0.05,
			},
			metricsProvider,
			histStore,
		),
		optimizer: optimizer.NewReactionOptimizer(
			optimizer.ReactionOptimizerConfig{
				CpuStep:              100,
				MemoryStep:           256,
				ReplicasStep:         1,
				TargetCpuUtilization: 0.40,
				Lambda:               config.Lambda,
				TimeWindow:           time.Duration(60 * time.Second),
			},
			metricsProvider,
			histStore,
		),
	}
}

func (d *Scaler) Start(ctx context.Context) error {
	if d.isRunning {
		return fmt.Errorf("analyzer is already running")
	}

	d.isRunning = true
	d.lastReactionTime = time.Now()

	go func() {
		ticker := time.NewTicker(d.config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				analizys_ctx := contextutil.WithAnalysisTime(ctx, time.Now())
				timeSinceLastReaction := time.Since(d.lastReactionTime)
				if timeSinceLastReaction < d.config.Cooldown {
					logger.Info("decision", "in cooldown period",
						"remaining", d.config.Cooldown-timeSinceLastReaction)
					continue
				}
				result := d.analyzer.Analyze(analizys_ctx)

				if result.Scale != 0 {
					mode := optimizer.ScaleUpMode
					if result.Scale < 0 {
						mode = optimizer.ScaleDownMode
					}
					logger.Info("scaler", "proposing scale for services", "services", result.Services)
					optimizedState, _ := d.optimizer.RunOptimization(analizys_ctx, result.Services, mode)
					err := d.scale(analizys_ctx, optimizedState)
					if err != nil {
						logger.Error("scaler", "error while scaling", "error", err.Error())
					}
					d.lastReactionTime = time.Now()
				} else {
					logger.Info("scaler", "no scaling action proposed")
				}
			case <-d.stopChan:
				return

			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (d *Scaler) scale(ctx context.Context, state optimizer.OptimizedState) error {
	if d.reactionApplier == nil {
		return fmt.Errorf("no reaction applier provided")
	}

	for _, s := range state.Services {
		switch s.Reaction {
		case optimizer.HPA:
			continue
		case optimizer.VPA:
			continue
		default:
			return fmt.Errorf("unsupported scaler reaction")
		}
	}

	// TODO make transaction
	for _, s := range state.Services {
		logger.Info("scaler", "applying candidate",
			"service", s.ServiceName,
			"reaction", s.Reaction,
			"replicas", s.Replicas,
			"cpu", s.AppCPU,
			"memory", s.AppMemory,
		)

		switch s.Reaction {
		case optimizer.HPA:
			return d.reactionApplier.ApplyHPS(ctx, d.config.Namespace, s.ServiceName, int32(s.Replicas))
		case optimizer.VPA:
			cpuStr := fmt.Sprintf("%dm", int(s.AppCPU))
			memStr := fmt.Sprintf("%dMi", int(s.AppMemory))
			return d.reactionApplier.ApplyVPS(ctx, d.config.Namespace, s.ServiceName, cpuStr, memStr)
		}
	}

	return nil
}

func (d *Scaler) Stop() error {
	if !d.isRunning {
		return nil
	}

	close(d.stopChan)
	d.isRunning = false
	return nil
}
