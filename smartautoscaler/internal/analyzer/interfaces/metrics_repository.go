package interfaces

import (
	"context"
	"time"

	"github.com/kudmo/CoolPA/internal/metrics"
)

type MetricsRepository interface {
	ListServices(ctx context.Context) ([]metrics.ServiceInfo, error)
	GetService(ctx context.Context, serviceName string) (*metrics.ServiceInfo, error)

	// Service metrics - Values
	GetServiceReplicasCountValue(ctx context.Context, serviceName string) (float64, error)

	GetServiceCpuUsageValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceMemoryUsageValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceCpuQuotaValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceMemoryQuotaValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceFSUsageValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceFSWriteValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceFSReadValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceNetworkReceiveValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceNetworkTransmitValue(ctx context.Context, serviceName string) (float64, error)
	GetServicePodCpuQuotaValue(ctx context.Context, serviceName string) (float64, error)
	GetServicePodMemoryQuotaValue(ctx context.Context, serviceName string) (float64, error)
	GetServiceRequestsCountValue(ctx context.Context, serviceName string) (float64, error)

	GetServiceAverageLatency95Value(ctx context.Context, serviceName string) (float64, error)

	// Service metrics - Ranges
	GetServiceReplicasCountRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)

	GetServiceCpuUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceMemoryUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceCpuQuotaRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceMemoryQuotaRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceFSUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceFSWriteRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceFSReadRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceNetworkReceiveRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceNetworkTransmitRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)
	GetServiceRequestsCountRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)

	GetServiceAverageLatency95Range(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error)

	// Graph metrics - Values
	GetGraphRequestsCountValue(ctx context.Context, serviceFrom, serviceTo string) (float64, error)
	GetGraphLatencyP95Value(ctx context.Context, serviceFrom, serviceTo string) (float64, error)
	GetGraphLatencyP50Value(ctx context.Context, serviceFrom, serviceTo string) (float64, error)

	// Graph metrics - Ranges
	GetGraphRequestsCountRange(ctx context.Context, serviceFrom, serviceTo string, from, to time.Time) ([]float64, error)
	GetGraphLatencyP95Range(ctx context.Context, serviceFrom, serviceTo string, from, to time.Time) ([]float64, error)
	GetGraphLatencyP50Range(ctx context.Context, serviceFrom, serviceTo string, from, to time.Time) ([]float64, error)

	// Global metrics
	GetGlobalTotalMemoryLimit(ctx context.Context) (float64, error)
	GetGlobalTotalCpuLimit(ctx context.Context) (float64, error)
	GetGlobalTotalPodsLimit(ctx context.Context) (float64, error)
}
