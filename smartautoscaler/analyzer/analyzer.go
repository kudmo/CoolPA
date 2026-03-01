package analyzer

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	sloviolation "github.com/kudmo/CoolPA/analyzer/slo_violation"
	"github.com/kudmo/CoolPA/storage"
	"github.com/kudmo/toporank/api"
	"github.com/kudmo/toporank/types"
)

type Config struct {
	Interval time.Duration
}

type Analyzer struct {
	Store     *storage.Storage
	stopChan  chan struct{}
	logger    *log.Logger
	config    Config
	isRunning bool
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

		params := sloviolation.AbnormalParams{
			Window: 1 * time.Minute,
			SLO:    0.5,
			Alpha:  0.2,
		}
		for {
			select {
			case <-ticker.C:
				graph, err := sloviolation.BuildAbnormalCorrelationGraph(time.Now(), params, a.Store.Graph)
				if err != nil {
					a.logger.Printf("Failed to build abnormal correlation graph: %v", err)
				} else {
					a.logger.Printf("Built abnormal correlation graph with %d nodes", len(graph.Nodes))
				}
				fmt.Printf("[DEBUG]: \n")
				anomalys := api.RunTopoRank(graph, types.DefaultConfig())
				for _, service := range anomalys {
					a.logger.Printf("Service: %s, Anomaly Score: %.4f", service.ID, service.Rank)
				}
			case <-a.stopChan:
				a.logger.Printf("Stopping metric collector")
				return

			case <-ctx.Done():
				a.logger.Printf("Context cancelled, stopping collector")
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
