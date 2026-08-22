package metricscache

import "github.com/kudmo/CoolPA/internal/metrics/providers/metrics_cache/storage"

type MetricsCacheProvider struct {
}

func NewCacheMetricsProvider(config storage.CacheConfig) *MetricsCacheProvider {
	return &MetricsCacheProvider{}
}
