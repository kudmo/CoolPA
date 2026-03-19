package collector

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type PrometheusCollectorConfig struct {
	PrometheusURL string
	Interval      time.Duration
	Queries       []MetricQuery
	Timeout       time.Duration
}

type MetricQuery struct {
	Name  string
	Query string
	Help  string
}

type MetricResult struct {
	QueryName string
	Help      string
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
	Error     error
}

type PrometheusCollector struct {
	client    v1.API
	config    PrometheusCollectorConfig
	handler   MetricHandler
	stopChan  chan struct{}
	isRunning bool
}

func NewPrometheusCollector(config PrometheusCollectorConfig, opts ...PrometheusOption) (*PrometheusCollector, error) {
	pc := &PrometheusCollector{
		config:   config,
		stopChan: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(pc)
	}

	client, err := api.NewClient(api.Config{
		Address: config.PrometheusURL,
	})
	if err != nil {
		return nil, err
	}

	pc.client = v1.NewAPI(client)
	return pc, nil
}

type PrometheusOption func(*PrometheusCollector)

func WithHandler(handler MetricHandler) PrometheusOption {
	return func(pc *PrometheusCollector) {
		pc.handler = handler
	}
}

func (pc *PrometheusCollector) Collect(ctx context.Context) ([]MetricResult, error) {

	ctx, cancel := context.WithTimeout(ctx, pc.config.Timeout)
	defer cancel()

	var allResults []MetricResult

	for _, query := range pc.config.Queries {

		results, err := pc.collectQuery(ctx, query)
		if err != nil {
			slog.Error("Failed to collect metric", query.Name, "error", err)
			continue
		}

		allResults = append(allResults, results...)
	}

	if pc.handler != nil {
		pc.handler.HandleBatch(allResults)
	}
	return allResults, nil
}
func (pc *PrometheusCollector) collectQuery(ctx context.Context, query MetricQuery) ([]MetricResult, error) {

	promResult, warnings, err := pc.client.Query(ctx, query.Query, time.Now())
	if err != nil {
		return nil, err
	}

	if len(warnings) > 0 {
		slog.Info("Warnings for query", query.Name, warnings)
	}

	return extractResults(query, promResult)
}

func extractResults(query MetricQuery, val model.Value) ([]MetricResult, error) {
	now := time.Now()
	var results []MetricResult

	switch val.Type() {

	case model.ValScalar:
		scalar := val.(*model.Scalar)
		results = append(results, MetricResult{
			QueryName: query.Name,
			Help:      query.Help,
			Value:     float64(scalar.Value),
			Timestamp: now,
			Labels:    map[string]string{},
		})

	case model.ValVector:
		vector := val.(model.Vector)

		for _, sample := range vector {
			labels := make(map[string]string)
			for name, value := range sample.Metric {
				labels[string(name)] = string(value)
			}

			results = append(results, MetricResult{
				QueryName: query.Name,
				Help:      query.Help,
				Value:     float64(sample.Value),
				Timestamp: now,
				Labels:    labels,
			})
		}

	case model.ValMatrix:
		matrix := val.(model.Matrix)

		for _, series := range matrix {
			if len(series.Values) == 0 {
				continue
			}

			last := series.Values[len(series.Values)-1]

			labels := make(map[string]string)
			for name, value := range series.Metric {
				labels[string(name)] = string(value)
			}

			results = append(results, MetricResult{
				QueryName: query.Name,
				Help:      query.Help,
				Value:     float64(last.Value),
				Timestamp: now,
				Labels:    labels,
			})
		}

	default:
		return nil, errors.New("unsupported value type")
	}

	return results, nil
}

func (pc *PrometheusCollector) Start(ctx context.Context) error {
	if pc.isRunning {
		return errors.New("collector is already running")
	}

	pc.isRunning = true

	go func() {
		ticker := time.NewTicker(pc.config.Interval)
		defer ticker.Stop()

		if _, err := pc.Collect(ctx); err != nil {
			slog.Error("Initial collection failed", "error", err)
		}

		for {
			select {
			case <-ticker.C:
				if _, err := pc.Collect(ctx); err != nil {
					slog.Error("Periodic collection failed", "error", err)
				}
			case <-pc.stopChan:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (pc *PrometheusCollector) Stop() error {
	if !pc.isRunning {
		return nil
	}

	close(pc.stopChan)
	pc.isRunning = false
	return nil
}
