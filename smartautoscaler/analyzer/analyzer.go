package analyzer

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	sloviolation "github.com/kudmo/CoolPA/analyzer/slo_violation"
	"github.com/kudmo/CoolPA/analyzer/welchtest"
	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/toporank/api"
	"github.com/kudmo/toporank/types"
)

type Config struct {
	Interval time.Duration

	Confidence float64

	WelchOldIntervalBegin time.Duration
	WelchNowIntervalBegin time.Duration
}

type Analyzer struct {
	Store     *storage.Storage
	stopChan  chan struct{}
	logger    *log.Logger
	config    Config
	isRunning bool
}

type AnalysisResult struct {
	Services []string // List of anomalous services
	Scale    int      // Scale can be -1 (scale down), 0 (no change), or 1 (scale up)s
}

func (a *Analyzer) analyzeWithSLOViolation() []string {
	params := sloviolation.AbnormalParams{
		Window: 1 * time.Minute,
		SLO:    200,
		Alpha:  0.2,
	}
	TOPORANK_THRESHOLD := 0.5
	result := make([]string, 0)

	graph, err := sloviolation.BuildAbnormalCorrelationGraph(time.Now(), params, a.Store.Graph)
	if err != nil {
		a.logger.Printf("Failed to build abnormal correlation graph: %v", err)
	} else {
		if graph == nil {
			a.logger.Printf("Empty correlation graph\n")
			return result
		}
		a.logger.Printf("Built abnormal correlation graph with %d nodes", len(graph.Nodes))
	}
	a.logger.Printf("[DEBUG]: \n")
	anomalys := api.RunTopoRank(graph, types.DefaultConfig())

	for _, service := range anomalys {
		a.logger.Printf("Service: %s, Anomaly Score: %.4f\n", service.ID, service.Rank)
		if service.Rank > TOPORANK_THRESHOLD {
			result = append(result, service.ID)
		}
	}
	return result
}

func (a *Analyzer) analyzeUnderutilization() []string {
	result := make([]string, 0)
	a.logger.Printf("[DEBUG]: \n")
	for _, service := range a.Store.Graph.GetServices() {
		n, _ := a.Store.Graph.GetNode(service)
		time_now := time.Now()
		time_now_begin := time_now.Add(-a.config.WelchNowIntervalBegin)
		time_old_begin := time_now.Add(-a.config.WelchOldIntervalBegin)

		old := n.RequestCount.SeriesRange(time_old_begin, time_now_begin)
		new := n.RequestCount.SeriesRange(time_now_begin, time_now)
		welch_result, _ := welchtest.TwoSampleWelch(new, old)
		a.logger.Printf("Welch_res: t:%f, p:%f\n", welch_result.TStatistic, welch_result.PValueOneSided())

		if welch_result.TStatistic > 0 && welch_result.PValueOneSided() <= a.config.Confidence {
			a.logger.Printf("Service %s has underutilization\n", service)
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

func NewAnalyzer(config Config, store *storage.Storage) *Analyzer {
	return &Analyzer{
		stopChan: make(chan struct{}),
		logger:   log.New(os.Stdout, "", log.LstdFlags),
		Store:    store,
		config:   config,
	}
}

func (a *Analyzer) Start(ctx context.Context) error {
	if a.isRunning {
		return fmt.Errorf("collector is already running")
	}

	a.isRunning = true
	a.logger.Printf("Starting analyzer with interval %v", a.config.Interval)

	go func() {
		ticker := time.NewTicker(a.config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				a.logger.Printf("Running analysis\n")
				result := a.Analyze()
				if len(result.Services) > 0 {
					a.logger.Printf("Anomalous services: %v, Scale: %d\n", result.Services, result.Scale)
				} else {
					a.logger.Printf("No anomalies detected\n")
				}
			case <-a.stopChan:
				a.logger.Printf("Stopping metric collector\n")
				return

			case <-ctx.Done():
				a.logger.Printf("Context cancelled, stopping collector\n")
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
