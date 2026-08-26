package interfaces

import (
	"context"
	"time"

	"github.com/kudmo/CoolPA/internal/metrics"
)

type MetricsRepository interface {
	ListServices(ctx context.Context) ([]string, error)
	GetService(ctx context.Context, serviceName string) (metrics.ServiceInfo, error)

	GetServiceReplicasCountValue(ctx context.Context, serviceName string) (float64, error)

	GetServiceAverageLatency95Value(ctx context.Context, serviceName string) (float64, error)

	GetServiceCpuUsageRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceMemoryUsageRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceFSUsageRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceFSWriteRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceFSReadRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceNetworkReceiveRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceNetworkTransmitRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceRequestsCountRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)

	GetGraphLatencyP95Range(ctx context.Context, serviceFrom, serviceTo string, rangeWindow time.Duration) ([]float64, error)
	GetGraphLatencyP50Range(ctx context.Context, serviceFrom, serviceTo string, rangeWindow time.Duration) ([]float64, error)

	GetServiceCpuQuota(ctx context.Context, serviceName string) (float64, error)
	GetServiceMemoryQuota(ctx context.Context, serviceName string) (float64, error)

	GetGlobalTotalMemoryLimit(ctx context.Context) (float64, error)
	GetGlobalTotalCpuLimit(ctx context.Context) (float64, error)
}
