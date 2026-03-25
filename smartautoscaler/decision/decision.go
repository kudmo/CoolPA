package decision

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/kudmo/CoolPA/analyzer"
	reactionapplier "github.com/kudmo/CoolPA/reaction_applier"
	"github.com/kudmo/CoolPA/storage"
)

type DecisionMakerConfig struct {
	Interval time.Duration
	Cooldown time.Duration
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
				Confidence:            0.05,
				WelchOldIntervalBegin: time.Duration(300 * time.Second),
				WelchNowIntervalBegin: time.Duration(60 * time.Second),
			},
			store),
		Optimizer: NewReactionOptimizer(store, applier),
	}
}

func (d *DecisionMaker) Start(ctx context.Context) error {
	if d.isRunning {
		return errors.New("Analyzer is already running")
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
					slog.Info("In cooldown period",
						"remaining", d.config.Cooldown-timeSinceLastReaction)
					continue
				}
				result := d.ServiceAnalyzer.Analyze()

				switch result.Scale {
				case 1:
					slog.Info("Proposing scale up for services", "services", result.Services)
					d.Optimizer.ScaleUp(result.Services)
					d.lastReactionTime = time.Now()
				case -1:
					slog.Info("Proposing scale down for services", "services", result.Services)
					d.Optimizer.ScaleDown(result.Services)
					d.lastReactionTime = time.Now()
				default:
					slog.Info("No scaling action proposed")
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
