package cache

import "time"

type CachedMetricsProviderConfig struct {
	TTL          time.Duration
	MaxCacheSize int
}
