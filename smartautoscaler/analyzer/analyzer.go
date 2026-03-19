package analyzer

import (
	"context"
	"errors"
	"log/slog"
	"time"

	sloviolation "github.com/kudmo/CoolPA/analyzer/slo_violation"
	"github.com/kudmo/CoolPA/analyzer/welchtest"
	"github.com/kudmo/CoolPA/decision"
	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/toporank/api"
	"github.com/kudmo/toporank/types"
)

type AnalyzerConfig struct {
	Interval time.Duration

	Confidence float64

	WelchOldIntervalBegin time.Duration
	WelchNowIntervalBegin time.Duration
}

type Analyzer struct {
	Store     *storage.Storage
	stopChan  chan struct{}
	config    AnalyzerConfig
	isRunning bool
}

type AnalysisResult struct {
	Services []string // List of anomalous services
	Scale    int      // Scale can be -1 (scale down), 0 (no change), or 1 (scale up)s
}

func (a *Analyzer) analyzeWithSLOViolation() []string {
	params := sloviolation.AbnormalParams{
		Window: 1 * time.Minute,
		SLO:    100,
		Alpha:  0.2,
	}
	TOPORANK_THRESHOLD := 0.5
	result := make([]string, 0)

	graph, err := sloviolation.BuildAbnormalCorrelationGraph(time.Now(), params, a.Store.Graph)
	if err != nil {
		slog.Error("Failed to build abnormal correlation graph", "error", err)
	} else {
		if graph == nil {
			return result
		}
	}
	anomalys := api.RunTopoRank(graph, types.DefaultConfig())

	for _, service := range anomalys {
		slog.Debug("Calculated anomaly", "service", service.ID, "Anomaly Score", service.Rank)
		if service.Rank > TOPORANK_THRESHOLD {
			result = append(result, service.ID)
		}
	}
	return result
}

func (a *Analyzer) analyzeUnderutilization() []string {
	result := make([]string, 0)
	for _, service := range a.Store.Graph.GetServices() {
		n, _ := a.Store.Graph.GetNode(service)
		time_now := time.Now()
		time_now_begin := time_now.Add(-a.config.WelchNowIntervalBegin)
		time_old_begin := time_now.Add(-a.config.WelchOldIntervalBegin)

		old := n.RequestCount.SeriesRange(time_old_begin, time_now_begin)
		new := n.RequestCount.SeriesRange(time_now_begin, time_now)
		welch_result, _ := welchtest.TwoSampleWelch(new, old)

		if welch_result.TStatistic > 0 && welch_result.PValueOneSided() <= a.config.Confidence {
			slog.Debug("Calculated underutilization", "service", service)
			result = append(result, service)
		}
	}
	return result
}

func (a *Analyzer) Analyze() AnalysisResult {
	bottlenecks := a.analyzeWithSLOViolation()
	if len(bottlenecks) > 0 {
		return AnalysisResult{
			Services: bottlenecks,
			Scale:    1,
		}
	}

	underutilized := a.analyzeUnderutilization()
	if len(underutilized) > 0 {
		return AnalysisResult{
			Services: underutilized,
			Scale:    -1,
		}
	}

	return AnalysisResult{
		Services: []string{},
		Scale:    0,
	}
}

func NewAnalyzer(config AnalyzerConfig, store *storage.Storage) *Analyzer {
	return &Analyzer{
		stopChan: make(chan struct{}),
		Store:    store,
		config:   config,
	}
}

func (a *Analyzer) Start(ctx context.Context) error {
	if a.isRunning {
		return errors.New("collector is already running")
	}

	a.isRunning = true

	go func() {
		ticker := time.NewTicker(a.config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				result := a.Analyze()

				reactor := decision.ReactionOptimizer{Store: a.Store}
				switch result.Scale {
				case 1:
					slog.Info("Proposing scale up for services", "services", result.Services)
					reactor.ScaleUp(result.Services, a.Store)
				case -1:
					slog.Info("Proposing scale down for services", "services", result.Services)
					reactor.ScaleDown(result.Services, a.Store)
				default:
					slog.Info("No scaling action proposed")
				}
			case <-a.stopChan:
				return

			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (a *Analyzer) Stop() error {
	if !a.isRunning {
		return nil
	}

	close(a.stopChan)
	a.isRunning = false
	return nil
}
