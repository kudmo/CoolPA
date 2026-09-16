package cache

import (
	"context"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/sync/singleflight"

	"github.com/kudmo/CoolPA/internal/metrics"
)

type CachedMetricsProvider struct {
	repo   metrics.MetricsRepository
	config CachedMetricsProviderConfig
	cache  *lru.Cache[string, cacheEntry]
	sf     singleflight.Group
}

func NewCachedMetricsRepository(repo metrics.MetricsRepository, config CachedMetricsProviderConfig) *CachedMetricsProvider {
	cache, _ := lru.New[string, cacheEntry](config.MaxCacheSize)
	return &CachedMetricsProvider{
		repo:   repo,
		config: config,
		cache:  cache,
	}
}
func (c *CachedMetricsProvider) get(ctx context.Context, key string, fetch func() (interface{}, error)) (interface{}, error) {
	entry, exists := c.cache.Get(key)

	if exists && time.Now().Before(entry.expiresAt) {
		return entry.value, nil
	}

	value, err, _ := c.sf.Do(key, func() (interface{}, error) {
		entry, exists := c.cache.Get(key)
		if exists && time.Now().Before(entry.expiresAt) {
			return entry.value, nil
		}

		fetched, err := fetch()
		if err != nil {
			return nil, err
		}

		c.cache.Add(key, cacheEntry{
			value:     fetched,
			expiresAt: time.Now().Add(c.config.TTL),
		})

		return fetched, nil
	})

	return value, err
}

func (c *CachedMetricsProvider) ListServices(ctx context.Context, namespace string) ([]string, error) {
	value, err := c.get(ctx, "list_services", func() (interface{}, error) {
		return c.repo.ListServices(ctx, namespace)
	})
	if err != nil {
		return nil, err
	}
	return value.([]string), nil
}

func (c *CachedMetricsProvider) GetService(ctx context.Context, svc metrics.ServiceID) (metrics.ServiceInfo, error) {
	key := "service_info_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetService(ctx, svc)
	})
	if err != nil {
		return metrics.ServiceInfo{}, err
	}
	return value.(metrics.ServiceInfo), nil
}

func (c *CachedMetricsProvider) GetServiceReplicasCountValue(ctx context.Context, svc metrics.ServiceID) (float64, error) {
	key := "replicas_count_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceReplicasCountValue(ctx, svc)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceCpuUsageValue(ctx context.Context, svc metrics.ServiceID) (float64, error) {
	key := "cpu_usage_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceCpuUsageValue(ctx, svc)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceMemoryUsageValue(ctx context.Context, svc metrics.ServiceID) (float64, error) {
	key := "memory_usage_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceMemoryUsageValue(ctx, svc)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceCpuQuota(ctx context.Context, svc metrics.ServiceID) (float64, error) {
	key := "cpu_quota_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceCpuQuota(ctx, svc)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceMemoryQuota(ctx context.Context, svc metrics.ServiceID) (float64, error) {
	key := "memory_quota_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceMemoryQuota(ctx, svc)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceFSUsageValue(ctx context.Context, svc metrics.ServiceID) (float64, error) {
	key := "fs_usage_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceFSUsageValue(ctx, svc)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceFSWriteValue(ctx context.Context, svc metrics.ServiceID) (float64, error) {
	key := "fs_write_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceFSWriteValue(ctx, svc)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceFSReadValue(ctx context.Context, svc metrics.ServiceID) (float64, error) {
	key := "fs_read_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceFSReadValue(ctx, svc)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceNetworkReceiveValue(ctx context.Context, svc metrics.ServiceID) (float64, error) {
	key := "network_receive_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceNetworkReceiveValue(ctx, svc)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceNetworkTransmitValue(ctx context.Context, svc metrics.ServiceID) (float64, error) {
	key := "network_transmit_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceNetworkTransmitValue(ctx, svc)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServicePodCpuQuota(ctx context.Context, svc metrics.ServiceID) (float64, error) {
	key := "pod_cpu_quota_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServicePodCpuQuota(ctx, svc)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServicePodMemoryQuota(ctx context.Context, svc metrics.ServiceID) (float64, error) {
	key := "pod_memory_quota_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServicePodMemoryQuota(ctx, svc)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceAverageLatency95Value(ctx context.Context, svc metrics.ServiceID) (float64, error) {
	key := "avg_latency_95_" + svc.Namespace + "_" + svc.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceAverageLatency95Value(ctx, svc)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetServiceCpuUsageRange(ctx context.Context, svc metrics.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	key := "cpu_usage_range_" + svc.Namespace + "_" + svc.Name + "_" + rangeWindow.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceCpuUsageRange(ctx, svc, rangeWindow)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceMemoryUsageRange(ctx context.Context, svc metrics.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	key := "memory_usage_range_" + svc.Namespace + "_" + svc.Name + "_" + rangeWindow.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceMemoryUsageRange(ctx, svc, rangeWindow)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceFSUsageRange(ctx context.Context, svc metrics.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	key := "fs_usage_range_" + svc.Namespace + "_" + svc.Name + "_" + rangeWindow.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceFSUsageRange(ctx, svc, rangeWindow)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceFSWriteRange(ctx context.Context, svc metrics.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	key := "fs_write_range_" + svc.Namespace + "_" + svc.Name + "_" + rangeWindow.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceFSWriteRange(ctx, svc, rangeWindow)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceFSReadRange(ctx context.Context, svc metrics.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	key := "fs_read_range_" + svc.Namespace + "_" + svc.Name + "_" + rangeWindow.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceFSReadRange(ctx, svc, rangeWindow)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceNetworkReceiveRange(ctx context.Context, svc metrics.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	key := "network_receive_range_" + svc.Namespace + "_" + svc.Name + "_" + rangeWindow.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceNetworkReceiveRange(ctx, svc, rangeWindow)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceNetworkTransmitRange(ctx context.Context, svc metrics.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	key := "network_transmit_range_" + svc.Namespace + "_" + svc.Name + "_" + rangeWindow.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceNetworkTransmitRange(ctx, svc, rangeWindow)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceRequestsCountRange(ctx context.Context, svc metrics.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	key := "requests_count_range_" + svc.Namespace + "_" + svc.Name + "_" + rangeWindow.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceRequestsCountRange(ctx, svc, rangeWindow)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetGraphRequestsCountValue(ctx context.Context, svcFrom, svcTo metrics.ServiceID) (float64, error) {
	key := "graph_requests_count_" + svcFrom.Namespace + "_" + svcFrom.Name + svcTo.Namespace + "_" + svcTo.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGraphRequestsCountValue(ctx, svcFrom, svcTo)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGraphLatencyP95Value(ctx context.Context, svcFrom, svcTo metrics.ServiceID) (float64, error) {
	key := "graph_latency_p95_" + svcFrom.Namespace + "_" + svcFrom.Name + svcTo.Namespace + "_" + svcTo.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGraphLatencyP95Value(ctx, svcFrom, svcTo)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGraphLatencyP50Value(ctx context.Context, svcFrom, svcTo metrics.ServiceID) (float64, error) {
	key := "graph_latency_p50_" + svcFrom.Namespace + "_" + svcFrom.Name + svcTo.Namespace + "_" + svcTo.Name
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGraphLatencyP50Value(ctx, svcFrom, svcTo)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGraphRequestsCountRange(ctx context.Context, svcFrom, svcTo metrics.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	key := "graph_requests_count_range_" + svcFrom.Namespace + "_" + svcFrom.Name + svcTo.Namespace + "_" + svcTo.Name + "_" + rangeWindow.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGraphRequestsCountRange(ctx, svcFrom, svcTo, rangeWindow)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetGraphLatencyP95Range(ctx context.Context, svcFrom, svcTo metrics.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	key := "graph_latency_p95_range_" + svcFrom.Namespace + "_" + svcFrom.Name + svcTo.Namespace + "_" + svcTo.Name + "_" + rangeWindow.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGraphLatencyP95Range(ctx, svcFrom, svcTo, rangeWindow)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetGraphLatencyP50Range(ctx context.Context, svcFrom, svcTo metrics.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	key := "graph_latency_p50_range_" + svcFrom.Namespace + "_" + svcFrom.Name + svcTo.Namespace + "_" + svcTo.Name + "_" + rangeWindow.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGraphLatencyP50Range(ctx, svcFrom, svcTo, rangeWindow)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetServiceAverageLatency95Range(ctx context.Context, svc metrics.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	key := "avg_latency_95_range_" + svc.Namespace + "_" + svc.Name + "_" + rangeWindow.String()
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetServiceAverageLatency95Range(ctx, svc, rangeWindow)
	})
	if err != nil {
		return nil, err
	}
	return value.([]float64), nil
}

func (c *CachedMetricsProvider) GetGlobalTotalMemoryLimit(ctx context.Context, namespace string) (float64, error) {
	key := "global_total_memory_limit"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalTotalMemoryLimit(ctx, namespace)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGlobalTotalCpuLimit(ctx context.Context, namespace string) (float64, error) {
	key := "global_total_cpu_limit"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalTotalCpuLimit(ctx, namespace)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGlobalTotalPodsLimit(ctx context.Context, namespace string) (float64, error) {
	key := "global_total_pods_limit"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalTotalPodsLimit(ctx, namespace)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGlobalServiceMinCpu(ctx context.Context, namespace string) (float64, error) {
	key := "service_min_cpu"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalServiceMinCpu(ctx, namespace)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGlobalServiceMaxCpu(ctx context.Context, namespace string) (float64, error) {
	key := "service_max_cpu"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalServiceMaxCpu(ctx, namespace)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGlobalServiceMinMemory(ctx context.Context, namespace string) (float64, error) {
	key := "service_min_memory"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalServiceMinMemory(ctx, namespace)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) GetGlobalServiceMaxMemory(ctx context.Context, namespace string) (float64, error) {
	key := "service_max_memory"
	value, err := c.get(ctx, key, func() (interface{}, error) {
		return c.repo.GetGlobalServiceMaxMemory(ctx, namespace)
	})
	if err != nil {
		return 0, err
	}
	return value.(float64), nil
}

func (c *CachedMetricsProvider) ClearCache() {
	c.cache.Purge()
}

func (c *CachedMetricsProvider) GetCacheSize() int {
	return c.cache.Len()
}
