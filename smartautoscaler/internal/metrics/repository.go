package metrics

import (
	"context"
	"time"
)

type MetricsRepository interface {
	ListServices(ctx context.Context) ([]string, error)
	GetService(ctx context.Context, serviceName string) (ServiceInfo, error)

	GetServiceReplicasCountValue(ctx context.Context, serviceName string) (float64, error)

	GetServiceCpuUsageValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceMemoryUsageValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceCpuQuota(ctx context.Context, serviceName string) (float64, error)
	GetServiceMemoryQuota(ctx context.Context, serviceName string) (float64, error)
	GetServiceFSUsageValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceFSWriteValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceFSReadValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceNetworkReceiveValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceNetworkTransmitValue(ctx context.Context, serviceName string) (float64, error)
	GetServicePodCpuQuota(ctx context.Context, serviceName string) (float64, error)
	GetServicePodMemoryQuota(ctx context.Context, serviceName string) (float64, error)

	GetServiceAverageLatency95Value(ctx context.Context, serviceName string) (float64, error)

	GetServiceCpuUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceMemoryUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceFSUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceFSWriteRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceFSReadRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceNetworkReceiveRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceNetworkTransmitRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceRequestsCountRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)

	GetGraphRequestsCountValue(ctx context.Context, serviceFrom, serviceTo string) (float64, error)
	GetGraphLatencyP95Value(ctx context.Context, serviceFrom, serviceTo string) (float64, error)
	GetGraphLatencyP50Value(ctx context.Context, serviceFrom, serviceTo string) (float64, error)

	GetGraphRequestsCountRange(ctx context.Context, serviceFrom, serviceTo string, from, to time.Time) ([]float64, error)
	GetGraphLatencyP95Range(ctx context.Context, serviceFrom, serviceTo string, from, to time.Time) ([]float64, error)
	GetGraphLatencyP50Range(ctx context.Context, serviceFrom, serviceTo string, from, to time.Time) ([]float64, error)

	GetServiceAverageLatency95Range(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)

	GetGlobalTotalMemoryLimit(ctx context.Context) (float64, error)
	GetGlobalTotalCpuLimit(ctx context.Context) (float64, error)
	GetGlobalTotalPodsLimit(ctx context.Context) (float64, error)

	GetServiceMinCpu(ctx context.Context) (float64, error)
	GetServiceMaxCpu(ctx context.Context) (float64, error)

	GetServiceMinMemory(ctx context.Context) (float64, error)
	GetServiceMaxMemory(ctx context.Context) (float64, error)
}
