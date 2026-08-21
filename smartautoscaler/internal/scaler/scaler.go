package scaler

import (
	"context"
	"fmt"
	"time"

	"github.com/kudmo/CoolPA/internal/analyzer"
	"github.com/kudmo/CoolPA/internal/applier"
	"github.com/kudmo/CoolPA/internal/optimizer"
	"github.com/kudmo/CoolPA/internal/scaler/interfaces"
	"github.com/kudmo/CoolPA/logger"
)

type Scaler struct {
	stopChan         chan struct{}
	config           ScalerConfig
	isRunning        bool
	lastReactionTime time.Time

	metricsProvider interfaces.MetricsProvider
	analyzer        *analyzer.Analyzer
	optimizer       *optimizer.ReactionOptimizer
	reactionApplier applier.Applier
}

func NewScaler(config ScalerConfig, metricsProvider interfaces.MetricsProvider, applier applier.Applier) *Scaler {
	return &Scaler{
		stopChan:        make(chan struct{}),
		metricsProvider: metricsProvider,
		config:          config,

		analyzer: analyzer.NewAnalyzer(
			analyzer.AnalyzerConfig{
				SLO:                   config.SLO,
				Confidence:            0.05,
				WelchOldIntervalBegin: time.Duration(300 * time.Second),
				WelchNowIntervalBegin: time.Duration(60 * time.Second),
				AnomalyServicesCount:  config.AnomalyServicesCount,
			},
			metricsProvider),
		optimizer: optimizer.NewReactionOptimizer(metricsProvider, optimizer.ReactionOptimizerConfig{
			CpuStep:              100,
			MemoryStep:           256,
			ReplicasStep:         1,
			TargetCpuUtilization: 0.40,
			Lambda:               config.Lambda,
		}),
	}
}

func (d *Scaler) Start(ctx context.Context) error {
	if d.isRunning {
		return fmt.Errorf("Analyzer is already running")
	}

	d.isRunning = true
	d.lastReactionTime = time.Now()

	go func() {
		ticker := time.NewTicker(d.config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				timeSinceLastReaction := time.Since(d.lastReactionTime)
				if timeSinceLastReaction < d.config.Cooldown {
					logger.Info("decision", "in cooldown period",
						"remaining", d.config.Cooldown-timeSinceLastReaction)
					continue
				}
				result := d.analyzer.Analyze(ctx)

				if result.Scale != 0 {
					mode := optimizer.ScaleUpMode
					if result.Scale < 0 {
						mode = optimizer.ScaleDownMode
					}
					logger.Info("decision", "proposing scale for services", "services", result.Services)
					optimizedState, _ := d.optimizer.RunOptimization(ctx, result.Services, mode)
					d.scale(ctx, optimizedState)
				} else {
					logger.Info("decision", "no scaling action proposed")
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
		return fmt.Errorf("No reaction applier provided")
	}

	for _, s := range state.Services {
		switch s.Reaction {
		case optimizer.HPA:
			continue
		case optimizer.VPA:
			continue
		default:
			return fmt.Errorf("Unsupported scaler reaction")
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
