package decision

import (
	"context"
	"fmt"
	"time"

	"github.com/kudmo/CoolPA/logger"

	"github.com/kudmo/CoolPA/analyzer"
	reactionapplier "github.com/kudmo/CoolPA/reaction_applier"
	"github.com/kudmo/CoolPA/storage"
)

type DecisionMakerConfig struct {
	Interval time.Duration
	Cooldown time.Duration
	SLO      float64
	Lambda   float64
}

type DecisionMaker struct {
	stopChan         chan struct{}
	config           DecisionMakerConfig
	isRunning        bool
	lastReactionTime time.Time

	Store           *storage.Storage
	ServiceAnalyzer *analyzer.Analyzer
	Optimizer       *ReactionOptimizer
}

func NewDecisionMaker(config DecisionMakerConfig, store *storage.Storage, applier reactionapplier.Applier) *DecisionMaker {
	return &DecisionMaker{
		stopChan: make(chan struct{}),
		Store:    store,
		config:   config,

		ServiceAnalyzer: analyzer.NewAnalyzer(
			analyzer.AnalyzerConfig{
				SLO:                   config.SLO,
				Confidence:            0.05,
				WelchOldIntervalBegin: time.Duration(300 * time.Second),
				WelchNowIntervalBegin: time.Duration(60 * time.Second),
				AnomalyServicesCount:  store.GlobalConfig.AnomalyServicesCount,
			},
			store),
		Optimizer: NewReactionOptimizer(store, applier, ReactionOptimizerConfig{
			CpuStep:              100,
			MemoryStep:           256,
			ReplicasStep:         1,
			TargetCpuUtilization: 0.40,
			Lambda:               config.Lambda,
		}),
	}
}

func (d *DecisionMaker) Start(ctx context.Context) error {
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
				result := d.ServiceAnalyzer.Analyze(ctx)

				switch result.Scale {
				case 1:
					logger.Info("decision", "proposing scale up for services", "services", result.Services)
					d.Optimizer.ScaleUp(result.Services)
					d.lastReactionTime = time.Now()
				case -1:
					logger.Info("decision", "proposing scale down for services", "services", result.Services)
					d.Optimizer.ScaleDown(result.Services)
					d.lastReactionTime = time.Now()
				default:
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

func (d *DecisionMaker) Stop() error {
	if !d.isRunning {
		return nil
	}

	close(d.stopChan)
	d.isRunning = false
	return nil
}
