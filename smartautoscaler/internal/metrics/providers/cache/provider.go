package cache

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/kudmo/CoolPA/internal/metrics"
)

type CachedMetricsProvider struct {
	repo   metrics.MetricsRepository
	config CachedMetricsProviderConfig
	mu     sync.RWMutex
	cache  map[string]cacheEntry
	sf     singleflight.Group
}

func NewCachedMetricsRepository(repo metrics.MetricsRepository, config CachedMetricsProviderConfig) *CachedMetricsProvider {
	return &CachedMetricsProvider{
		repo:   repo,
		config: config,
		cache:  make(map[string]cacheEntry),
	}
}
func (c *CachedMetricsProvider) get(ctx context.Context, key string, fetch func() (interface{}, error)) (interface{}, error) {
	c.mu.RLock()
	entry, exists := c.cache[key]
	c.mu.RUnlock()

	if exists && time.Now().Before(entry.expiresAt) {
		return entry.value, nil
	}

	value, err, _ := c.sf.Do(key, func() (interface{}, error) {
		c.mu.RLock()
		entry, exists := c.cache[key]
		c.mu.RUnlock()
		if exists && time.Now().Before(entry.expiresAt) {
			return entry.value, nil
		}

		fetched, err := fetch()
		if err != nil {
			return nil, err
		}

		c.mu.Lock()
		c.cache[key] = cacheEntry{
			value:     fetched,
			expiresAt: time.Now().Add(c.config.TTL),
		}
		c.mu.Unlock()

		return fetched, nil
	})

	return value, err
}

func (c *CachedMetricsProvider) ListServices(ctx context.Context) ([]string, error) {
	value, err := c.get(ctx, "list_services", func() (interface{}, error) {
		return c.repo.ListServices(ctx)
	})
	if err != nil {
		return nil, err
	}
	return value.([]string), nil
}

func (c *CachedMetricsProvider) GetService(ctx context.Context, serviceName string) (metrics.ServiceInfo, error) {
	key := "service_info_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetService(ctx, serviceName)
	})
	if err != nil {
		return metrics.ServiceInfo{}, err
	}
	return value.(metrics.ServiceInfo), nil
}

func (c *CachedMetricsProvider) GetServiceReplicasCountValue(ctx context.Context, serviceName string) (float64, error) {
	key := "replicas_count_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceReplicasCountValue(ctx, serviceName)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceCpuUsageValue(ctx context.Context, serviceName string) (float64, error) {
	key := "cpu_usage_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceCpuUsageValue(ctx, serviceName)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceMemoryUsageValue(ctx context.Context, serviceName string) (float64, error) {
	key := "memory_usage_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceMemoryUsageValue(ctx, serviceName)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceCpuQuota(ctx context.Context, serviceName string) (float64, error) {
	key := "cpu_quota_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceCpuQuota(ctx, serviceName)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceMemoryQuota(ctx context.Context, serviceName string) (float64, error) {
	key := "memory_quota_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceMemoryQuota(ctx, serviceName)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceFSUsageValue(ctx context.Context, serviceName string) (float64, error) {
	key := "fs_usage_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceFSUsageValue(ctx, serviceName)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceFSWriteValue(ctx context.Context, serviceName string) (float64, error) {
	key := "fs_write_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceFSWriteValue(ctx, serviceName)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceFSReadValue(ctx context.Context, serviceName string) (float64, error) {
	key := "fs_read_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceFSReadValue(ctx, serviceName)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceNetworkReceiveValue(ctx context.Context, serviceName string) (float64, error) {
	key := "network_receive_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceNetworkReceiveValue(ctx, serviceName)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceNetworkTransmitValue(ctx context.Context, serviceName string) (float64, error) {
	key := "network_transmit_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceNetworkTransmitValue(ctx, serviceName)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServicePodCpuQuota(ctx context.Context, serviceName string) (float64, error) {
	key := "pod_cpu_quota_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServicePodCpuQuota(ctx, serviceName)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServicePodMemoryQuota(ctx context.Context, serviceName string) (float64, error) {
	key := "pod_memory_quota_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServicePodMemoryQuota(ctx, serviceName)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceAverageLatency95Value(ctx context.Context, serviceName string) (float64, error) {
	key := "avg_latency_95_" + serviceName
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceAverageLatency95Value(ctx, serviceName)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceCpuUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	key := "cpu_usage_range_" + serviceName + "_" + from.String() + "_" + to.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceCpuUsageRange(ctx, serviceName, from, to)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceMemoryUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	key := "memory_usage_range_" + serviceName + "_" + from.String() + "_" + to.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceMemoryUsageRange(ctx, serviceName, from, to)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceFSUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	key := "fs_usage_range_" + serviceName + "_" + from.String() + "_" + to.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceFSUsageRange(ctx, serviceName, from, to)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceFSWriteRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	key := "fs_write_range_" + serviceName + "_" + from.String() + "_" + to.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceFSWriteRange(ctx, serviceName, from, to)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceFSReadRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	key := "fs_read_range_" + serviceName + "_" + from.String() + "_" + to.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceFSReadRange(ctx, serviceName, from, to)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceNetworkReceiveRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	key := "network_receive_range_" + serviceName + "_" + from.String() + "_" + to.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceNetworkReceiveRange(ctx, serviceName, from, to)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceNetworkTransmitRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	key := "network_transmit_range_" + serviceName + "_" + from.String() + "_" + to.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceNetworkTransmitRange(ctx, serviceName, from, to)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceRequestsCountRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	key := "requests_count_range_" + serviceName + "_" + from.String() + "_" + to.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceRequestsCountRange(ctx, serviceName, from, to)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetGraphRequestsCountValue(ctx context.Context, serviceFrom, serviceTo string) (float64, error) {
	key := "graph_requests_count_" + serviceFrom + "_" + serviceTo
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGraphRequestsCountValue(ctx, serviceFrom, serviceTo)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGraphLatencyP95Value(ctx context.Context, serviceFrom, serviceTo string) (float64, error) {
	key := "graph_latency_p95_" + serviceFrom + "_" + serviceTo
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGraphLatencyP95Value(ctx, serviceFrom, serviceTo)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGraphLatencyP50Value(ctx context.Context, serviceFrom, serviceTo string) (float64, error) {
	key := "graph_latency_p50_" + serviceFrom + "_" + serviceTo
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGraphLatencyP50Value(ctx, serviceFrom, serviceTo)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGraphRequestsCountRange(ctx context.Context, serviceFrom, serviceTo string, from, to time.Time) ([]float64, error) {
	key := "graph_requests_count_range_" + serviceFrom + "_" + serviceTo + "_" + from.String() + "_" + to.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGraphRequestsCountRange(ctx, serviceFrom, serviceTo, from, to)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetGraphLatencyP95Range(ctx context.Context, serviceFrom, serviceTo string, from, to time.Time) ([]float64, error) {
	key := "graph_latency_p95_range_" + serviceFrom + "_" + serviceTo + "_" + from.String() + "_" + to.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGraphLatencyP95Range(ctx, serviceFrom, serviceTo, from, to)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetGraphLatencyP50Range(ctx context.Context, serviceFrom, serviceTo string, from, to time.Time) ([]float64, error) {
	key := "graph_latency_p50_range_" + serviceFrom + "_" + serviceTo + "_" + from.String() + "_" + to.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGraphLatencyP50Range(ctx, serviceFrom, serviceTo, from, to)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceAverageLatency95Range(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	key := "avg_latency_95_range_" + serviceName + "_" + from.String() + "_" + to.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceAverageLatency95Range(ctx, serviceName, from, to)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetGlobalTotalMemoryLimit(ctx context.Context) (float64, error) {
	key := "global_total_memory_limit"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalTotalMemoryLimit(ctx)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGlobalTotalCpuLimit(ctx context.Context) (float64, error) {
	key := "global_total_cpu_limit"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalTotalCpuLimit(ctx)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGlobalTotalPodsLimit(ctx context.Context) (float64, error) {
	key := "global_total_pods_limit"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalTotalPodsLimit(ctx)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGlobalServiceMinCpu(ctx context.Context) (float64, error) {
	key := "service_min_cpu"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalServiceMinCpu(ctx)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGlobalServiceMaxCpu(ctx context.Context) (float64, error) {
	key := "service_max_cpu"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalServiceMaxCpu(ctx)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGlobalServiceMinMemory(ctx context.Context) (float64, error) {
	key := "service_min_memory"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalServiceMinMemory(ctx)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGlobalServiceMaxMemory(ctx context.Context) (float64, error) {
	key := "service_max_memory"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalServiceMaxMemory(ctx)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) ClearCache() {
	c.mu.Lock()
	c.cache = make(map[string]cacheEntry)
	c.mu.Unlock()
}

func (c *CachedMetricsProvider) GetCacheSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}
