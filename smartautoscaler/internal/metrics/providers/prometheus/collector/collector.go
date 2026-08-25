package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/kudmo/CoolPA/logger"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type PrometheusCollector struct {
	client v1.API
	config PrometheusCollectorConfig
}

func NewPrometheusCollector(config PrometheusCollectorConfig) (*PrometheusCollector, error) {
	pc := &PrometheusCollector{
		config: config,
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

func (pc *PrometheusCollector) CollectQuery(ctx context.Context, query MetricQuery) ([]MetricResult, error) {

	promResult, warnings, err := pc.client.Query(ctx, query.Query, time.Now())
	if err != nil {
		return nil, err
	}

	if len(warnings) > 0 {
		logger.Info("collector", "warnings for query", "query", query.Name, "warnings", warnings)
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
				Value:     float64(last.Value),
				Timestamp: now,
				Labels:    labels,
			})
		}

	default:
		return nil, fmt.Errorf("unsupported value type")
	}

	return results, nil
}
