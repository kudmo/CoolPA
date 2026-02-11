package collector

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type Config struct {
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

type MetricHandler interface {
	Handle(result MetricResult)
	HandleBatch(results []MetricResult)
}

type PrometheusCollector struct {
	client    v1.API
	config    Config
	logger    *log.Logger
	handler   MetricHandler
	stopChan  chan struct{}
	isRunning bool
}

func NewPrometheusCollector(config Config, opts ...PrometheusOption) (*PrometheusCollector, error) {
	pc := &PrometheusCollector{
		config:   config,
		logger:   log.New(os.Stdout, "", log.LstdFlags),
		stopChan: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(pc)
	}

	client, err := api.NewClient(api.Config{
		Address: config.PrometheusURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
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

	var results []MetricResult
	for _, query := range pc.config.Queries {
		result, err := pc.collectQuery(ctx, query)
		if err != nil {
			pc.logger.Printf("Failed to collect metric %s: %v", query.Name, err)
			result.Error = err
		}

		results = append(results, result)

		if pc.handler != nil {
			pc.handler.Handle(result)
		}
	}

	return results, nil
}

func (pc *PrometheusCollector) CollectSingle(ctx context.Context, query MetricQuery) (MetricResult, error) {
	ctx, cancel := context.WithTimeout(ctx, pc.config.Timeout)
	defer cancel()

	return pc.collectQuery(ctx, query)
}

func (pc *PrometheusCollector) collectQuery(ctx context.Context, query MetricQuery) (MetricResult, error) {
	result := MetricResult{
		QueryName: query.Name,
		Help:      query.Help,
		Timestamp: time.Now(),
	}

	promResult, warnings, err := pc.client.Query(ctx, query.Query, time.Now())
	if err != nil {
		return result, fmt.Errorf("query failed: %w", err)
	}

	if len(warnings) > 0 {
		pc.logger.Printf("Warnings for query %s: %v", query.Name, warnings)
	}

	value, labels, err := extractValueAndLabels(promResult)
	if err != nil {
		return result, fmt.Errorf("failed to extract value: %w", err)
	}

	result.Value = value
	result.Labels = labels

	pc.logger.Printf("Collected metric %s = %f", query.Name, value)
	return result, nil
}

func extractValueAndLabels(val model.Value) (float64, map[string]string, error) {
	labels := make(map[string]string)

	switch val.Type() {
	case model.ValScalar:
		scalar := val.(*model.Scalar)
		return float64(scalar.Value), labels, nil

	case model.ValVector:
		vector := val.(model.Vector)
		if len(vector) == 0 {
			return 0, labels, fmt.Errorf("no data in vector")
		}

		for name, value := range vector[0].Metric {
			labels[string(name)] = string(value)
		}

		return float64(vector[0].Value), labels, nil

	case model.ValMatrix:
		matrix := val.(model.Matrix)
		if len(matrix) == 0 || len(matrix[0].Values) == 0 {
			return 0, labels, fmt.Errorf("no data in matrix")
		}

		// Берем последнее значение из первого ряда
		lastValue := matrix[0].Values[len(matrix[0].Values)-1]
		for name, value := range matrix[0].Metric {
			labels[string(name)] = string(value)
		}

		return float64(lastValue.Value), labels, nil

	default:
		return 0, labels, fmt.Errorf("unsupported value type: %s", val.Type())
	}
}

func (pc *PrometheusCollector) Start(ctx context.Context) error {
	if pc.isRunning {
		return fmt.Errorf("collector is already running")
	}

	pc.isRunning = true
	pc.logger.Printf("Starting metric collector with interval %v", pc.config.Interval)

	go func() {
		ticker := time.NewTicker(pc.config.Interval)
		defer ticker.Stop()

		if _, err := pc.Collect(ctx); err != nil {
			pc.logger.Printf("Initial collection failed: %v", err)
		}

		for {
			select {
			case <-ticker.C:
				if _, err := pc.Collect(ctx); err != nil {
					pc.logger.Printf("Periodic collection failed: %v", err)
				}

			case <-pc.stopChan:
				pc.logger.Printf("Stopping metric collector")
				return

			case <-ctx.Done():
				pc.logger.Printf("Context cancelled, stopping collector")
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
