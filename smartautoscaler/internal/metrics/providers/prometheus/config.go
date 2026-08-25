package prometheus

import "github.com/kudmo/CoolPA/internal/metrics/providers/prometheus/collector"

type PrometheusMetricsProviderConfig struct {
	ScalingNamespace string
	PrometheusConfig collector.PrometheusCollectorConfig
}
