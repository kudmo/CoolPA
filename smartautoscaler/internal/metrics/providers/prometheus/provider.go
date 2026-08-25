package prometheus

import (
	"github.com/kudmo/CoolPA/internal/metrics/providers/prometheus/collector"
)

type PrometheusMetricsProvider struct {
	config              PrometheusMetricsProviderConfig
	prometheusCollector *collector.PrometheusCollector
}
