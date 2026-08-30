// Package scaler implements the main autoscaling loop. It periodically
// analyzes service metrics, detects anomalies or underutilization,
// runs an optimizer to determine target resource states, and applies
// the recommended scaling actions.
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

// Scaler orchestrates the autoscaling loop. It uses an Analyzer to
// detect services requiring scaling, an Optimizer to compute target
// states, and an Applier to execute scaling actions.
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

// NewScaler creates a Scaler with the given configuration, metrics
// provider, and reaction applier. It initializes the analyzer and
// optimizer components with default parameters.
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
				Window:               60 * time.Second,
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
				TimeWindow:           60 * time.Second,
			},
			metricsProvider,
			histStore,
		),
	}
}

// Start launches the autoscaling loop in a separate goroutine. It runs
// analysis at the configured interval, respects the cooldown period
// between scaling actions, and stops when the context is cancelled or
// Stop is called. Returns an error if the scaler is already running.
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
				analysisCtx := contextutil.WithAnalysisTime(ctx, time.Now())

				// Skip if still in cooldown period.
				timeSinceLastReaction := time.Since(d.lastReactionTime)
				if timeSinceLastReaction < d.config.Cooldown {
					logger.Info("decision", "in cooldown period",
						"remaining", d.config.Cooldown-timeSinceLastReaction)
					continue
				}

				// Run analysis to detect services requiring scaling.
				result := d.analyzer.Analyze(analysisCtx)

				if result.Scale != 0 {
					mode := optimizer.ScaleUpMode
					if result.Scale < 0 {
						mode = optimizer.ScaleDownMode
					}

					logger.Info("scaler", "proposing scale for services", "services", result.Services)
					optimizedState, _ := d.optimizer.RunOptimization(analysisCtx, result.Services, mode)
					if err := d.scale(analysisCtx, optimizedState); err != nil {
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

// scale applies the optimized state by invoking the appropriate
// reaction applier for each service. It currently supports HPA
// (horizontal scaling) and VPA (vertical scaling) reactions.
// Returns an error if no applier is set or an unsupported reaction
// is encountered.
func (d *Scaler) scale(ctx context.Context, state optimizer.OptimizedState) error {
	if d.reactionApplier == nil {
		return fmt.Errorf("no reaction applier provided")
	}

	// Validate all reactions before applying any.
	for _, s := range state.Services {
		switch s.Reaction {
		case optimizer.HPA, optimizer.VPA:
			// supported
		default:
			return fmt.Errorf("unsupported scaler reaction")
		}
	}

	// Apply scaling actions (not transactional yet).
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

// Stop gracefully stops the autoscaling loop. It is safe to call
// multiple times. If the scaler is not running, it does nothing.
func (d *Scaler) Stop() error {
	if !d.isRunning {
		return nil
	}

	close(d.stopChan)
	d.isRunning = false
	return nil
}
