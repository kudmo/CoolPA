package interfaces

import (
	"context"
	"time"
)

type MetricsRepository interface {
	ListServices(ctx context.Context) ([]string, error)

	GetServiceReplicasCountValue(ctx context.Context, serviceName string) (float64, error)

	GetServiceCpuUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceMemoryUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceNetworkReceiveRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceNetworkTransmitRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceAverageLatency95Range(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)

	GetServiceCpuQuota(ctx context.Context, serviceName string) (float64, error)
	GetServiceMemoryQuota(ctx context.Context, serviceName string) (float64, error)
	GetServicePodCpuQuota(ctx context.Context, serviceName string) (float64, error)
	GetServicePodMemoryQuota(ctx context.Context, serviceName string) (float64, error)

	GetGlobalTotalMemoryLimit(ctx context.Context) (float64, error)
	GetGlobalTotalCpuLimit(ctx context.Context) (float64, error)
	GetGlobalTotalPodsLimit(ctx context.Context) (float64, error)

	GetGlobalServiceMinCpu(ctx context.Context) (float64, error)
	GetGlobalServiceMaxCpu(ctx context.Context) (float64, error)

	GetGlobalServiceMinMemory(ctx context.Context) (float64, error)
	GetGlobalServiceMaxMemory(ctx context.Context) (float64, error)
}
