// All methods accept the context.Context parameter to enable cancellation and
// setting timeouts. Implementations must adhere to the deadlines set in the context, and
// if the operation cannot be completed, an error must be returned.

// The context also passes the "analysis_time_begin" parameter,
// which indicates the time at which the data is requested.
package metrics

import (
	"context"
	"time"
)

// MetricsRepository is the contract for providing metrics to the
// autoscaler. It abstracts the underlying data source (e.g.,
// Prometheus, Kubernetes Metrics API) and offers methods to retrieve
// both instantaneous values and time series.
//
// Implementations must be safe for concurrent use.
type MetricsRepository interface {
	// ListServices returns the names of all services that are being
	// monitored and are candidates for autoscaling.
	ListServices(ctx context.Context) ([]string, error)

	// GetService returns detailed information about a specific service.
	GetService(ctx context.Context, serviceName string) (ServiceInfo, error)

	// GetServiceReplicasCountValue returns the current number of replicas
	// for the given service.
	GetServiceReplicasCountValue(ctx context.Context, serviceName string) (float64, error)

	// GetServiceCpuUsageValue returns the current CPU usage of the
	// service (in CPU cores or millicores depending on implementation).
	GetServiceCpuUsageValue(ctx context.Context, serviceName string) (float64, error)

	// GetServiceMemoryUsageValue returns the current memory usage of
	// the service in bytes.
	GetServiceMemoryUsageValue(ctx context.Context, serviceName string) (float64, error)

	// The following methods return quotas (limits) for the service's
	// container(s).

	// GetServiceCpuQuota returns the CPU limit allocated to the service's
	// container.
	GetServiceCpuQuota(ctx context.Context, serviceName string) (float64, error)

	// GetServiceMemoryQuota returns the memory limit allocated to the
	// service's container.
	GetServiceMemoryQuota(ctx context.Context, serviceName string) (float64, error)

	// GetServiceFSUsageValue returns the current filesystem usage of
	// the service.
	GetServiceFSUsageValue(ctx context.Context, serviceName string) (float64, error)

	// GetServiceFSWriteValue returns the current filesystem write rate
	// of the service.
	GetServiceFSWriteValue(ctx context.Context, serviceName string) (float64, error)

	// GetServiceFSReadValue returns the current filesystem read rate
	// of the service.
	GetServiceFSReadValue(ctx context.Context, serviceName string) (float64, error)

	// GetServiceNetworkReceiveValue returns the current network receive
	// rate of the service.
	GetServiceNetworkReceiveValue(ctx context.Context, serviceName string) (float64, error)

	// GetServiceNetworkTransmitValue returns the current network
	// transmit rate of the service.
	GetServiceNetworkTransmitValue(ctx context.Context, serviceName string) (float64, error)

	// Pod-level quotas (limits for the entire pod, not just container).

	// GetServicePodCpuQuota returns the total CPU limit for the pod(s)
	// of the service.
	GetServicePodCpuQuota(ctx context.Context, serviceName string) (float64, error)

	// GetServicePodMemoryQuota returns the total memory limit for the
	// pod(s) of the service.
	GetServicePodMemoryQuota(ctx context.Context, serviceName string) (float64, error)

	// GetServiceAverageLatency95Value returns the current 95th percentile
	// latency for the service.
	GetServiceAverageLatency95Value(ctx context.Context, serviceName string) (float64, error)

	// The following methods return a time series of values over the
	// specified rangeWindow ending at the current time. The slice is
	// ordered chronologically. If no data is available, an empty slice
	// and nil error are returned.

	GetServiceCpuUsageRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceMemoryUsageRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceFSUsageRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceFSWriteRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceFSReadRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceNetworkReceiveRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceNetworkTransmitRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)
	GetServiceRequestsCountRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)

	// Graph metrics represent interactions between two services.
	// serviceFrom is the source, serviceTo is the destination.

	GetGraphRequestsCountValue(ctx context.Context, serviceFrom, serviceTo string) (float64, error)
	GetGraphLatencyP95Value(ctx context.Context, serviceFrom, serviceTo string) (float64, error)
	GetGraphLatencyP50Value(ctx context.Context, serviceFrom, serviceTo string) (float64, error)

	GetGraphRequestsCountRange(ctx context.Context, serviceFrom, serviceTo string, rangeWindow time.Duration) ([]float64, error)
	GetGraphLatencyP95Range(ctx context.Context, serviceFrom, serviceTo string, rangeWindow time.Duration) ([]float64, error)
	GetGraphLatencyP50Range(ctx context.Context, serviceFrom, serviceTo string, rangeWindow time.Duration) ([]float64, error)

	// GetServiceAverageLatency95Range returns a time series of the
	// service's 95th percentile latency.
	GetServiceAverageLatency95Range(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error)

	// Global cluster-wide limits and per-service min/max constraints.

	// GetGlobalTotalMemoryLimit returns the total memory limit for the
	// entire cluster or namespace (implementation-specific).
	GetGlobalTotalMemoryLimit(ctx context.Context) (float64, error)

	// GetGlobalTotalCpuLimit returns the total CPU limit for the entire
	// cluster or namespace.
	GetGlobalTotalCpuLimit(ctx context.Context) (float64, error)

	// GetGlobalTotalPodsLimit returns the maximum number of pods allowed
	// in the cluster or namespace.
	GetGlobalTotalPodsLimit(ctx context.Context) (float64, error)

	// GetGlobalServiceMinCpu returns the minimum CPU allowed for a
	// single service.
	GetGlobalServiceMinCpu(ctx context.Context) (float64, error)

	// GetGlobalServiceMaxCpu returns the maximum CPU allowed for a
	// single service.
	GetGlobalServiceMaxCpu(ctx context.Context) (float64, error)

	// GetGlobalServiceMinMemory returns the minimum memory allowed for
	// a single service.
	GetGlobalServiceMinMemory(ctx context.Context) (float64, error)

	// GetGlobalServiceMaxMemory returns the maximum memory allowed for
	// a single service.
	GetGlobalServiceMaxMemory(ctx context.Context) (float64, error)
}
