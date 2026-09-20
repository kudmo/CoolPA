package prometheus

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/kudmo/CoolPA/internal/core"
	"github.com/kudmo/CoolPA/internal/metrics"
	"github.com/kudmo/CoolPA/internal/metrics/providers/prometheus/collector"
)

type PrometheusMetricsProvider struct {
	config              PrometheusMetricsProviderConfig
	prometheusCollector *collector.PrometheusCollector
}

func NewPrometheusMetricsProvider(config PrometheusMetricsProviderConfig) (*PrometheusMetricsProvider, error) {
	c, err := collector.NewPrometheusCollector(config.PrometheusConfig)
	if err != nil {
		return nil, err
	}
	return &PrometheusMetricsProvider{
		config:              config,
		prometheusCollector: c,
	}, nil
}

func (p *PrometheusMetricsProvider) ListServices(ctx context.Context, namespace string) ([]string, error) {
	result_raw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name:  "services",
		Query: fmt.Sprintf(`kube_service_info{namespace="%s"}`, namespace),
	})
	if err != nil {
		return nil, err
	}
	var services []string
	for _, s := range result_raw {
		services = append(services, s.Labels["service"])
	}
	return services, nil
}

func (p *PrometheusMetricsProvider) GetService(ctx context.Context, svc core.ServiceID) (metrics.ServiceInfo, error) {
	serviceInfo := metrics.ServiceInfo{
		ID:            svc,
		InboundCalls:  []core.ServiceID{},
		OutboundCalls: []core.ServiceID{},
	}

	inboundRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "inbound_calls",
		Query: fmt.Sprintf(`
			group by (source_workload, source_workload_namespace) (
				istio_requests_total{
					destination_workload_namespace="%s",
					destination_workload="%s",
					source_workload!="unknown",
					source_workload!="",
					source_workload_namespace!=""
				}
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return serviceInfo, fmt.Errorf("failed to get inbound calls: %w", err)
	}

	for _, call := range inboundRaw {
		sourceName := call.Labels["source_workload"]
		sourceNamespace := call.Labels["source_workload_namespace"]
		if sourceName == "" || sourceNamespace == "" {
			continue
		}
		serviceInfo.InboundCalls = append(serviceInfo.InboundCalls, core.ServiceID{
			Namespace: sourceNamespace,
			Name:      sourceName,
		})
	}

	outboundRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "outbound_calls",
		Query: fmt.Sprintf(`
			group by (destination_workload, destination_workload_namespace) (
				istio_requests_total{
					source_workload_namespace="%s",
					source_workload="%s",
					destination_workload!="unknown",
					destination_workload!="",
					destination_workload_namespace!=""
				}
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return serviceInfo, fmt.Errorf("failed to get outbound calls: %w", err)
	}

	for _, call := range outboundRaw {
		destName := call.Labels["destination_workload"]
		destNamespace := call.Labels["destination_workload_namespace"]
		if destName == "" || destNamespace == "" {
			continue
		}
		serviceInfo.OutboundCalls = append(serviceInfo.OutboundCalls, core.ServiceID{
			Namespace: destNamespace,
			Name:      destName,
		})
	}

	serviceInfo.InboundCalls = dedupeAndSortServiceIDs(serviceInfo.InboundCalls)
	serviceInfo.OutboundCalls = dedupeAndSortServiceIDs(serviceInfo.OutboundCalls)

	return serviceInfo, nil
}

func dedupeAndSortServiceIDs(ids []core.ServiceID) []core.ServiceID {
	if len(ids) == 0 {
		return ids
	}
	seen := make(map[core.ServiceID]struct{}, len(ids))
	out := make([]core.ServiceID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (p *PrometheusMetricsProvider) GetServiceReplicasCountValue(ctx context.Context, svc core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_replicas_count",
		Query: fmt.Sprintf(`
			count(
				kube_pod_info{
					namespace="%s",
					created_by_name=~"%s.*"
				}
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetServiceCpuUsageValue(ctx context.Context, svc core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_cpu_usage",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					rate(container_cpu_usage_seconds_total{
						container!="",
						container!="istio-proxy",
						container!="POD",
						namespace="%s",
						pod=~"%s.*"
					}[1m])
				)
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetServiceMemoryUsageValue(ctx context.Context, svc core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_memory_usage",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					container_memory_usage_bytes{
						container!="",
						container!="istio-proxy",
						container!="POD",
						namespace="%s",
						pod=~"%s.*"
					}
				) / (1024*1024)
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get memory usage: %w", err)
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetServiceCpuQuota(ctx context.Context, svc core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_cpu_quota",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					container_spec_cpu_quota{
						container!="",
						container!="istio-proxy",
						container!="POD",
						namespace="%s",
						pod=~"%s.*"
					}
				) / 100
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetServiceMemoryQuota(ctx context.Context, svc core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_memory_quota",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					container_spec_memory_limit_bytes{
						container!="",
						container!="istio-proxy",
						container!="POD",
						namespace="%s",
						pod=~"%s.*"
					}
				) / (1024*1024)
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetServiceFSUsageValue(ctx context.Context, svc core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_fs_usage",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					container_fs_usage_bytes{
						container!="",
						container!="istio-proxy",
						container!="POD",
						namespace="%s",
						pod=~"%s.*"
					}
				)
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetServiceFSWriteValue(ctx context.Context, svc core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_fs_write",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					rate(container_fs_writes_bytes_total{
						container!="",
						container!="istio-proxy",
						container!="POD",
						namespace="%s",
						pod=~"%s.*"
					}[1m])
				)
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetServiceFSReadValue(ctx context.Context, svc core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_fs_read",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					rate(container_fs_reads_bytes_total{
						container!="",
						container!="istio-proxy",
						container!="POD",
						namespace="%s",
						pod=~"%s.*"
					}[1m])
				)
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetServiceNetworkReceiveValue(ctx context.Context, svc core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_network_receive",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					rate(container_network_receive_bytes_total{
						namespace="%s",
						pod=~"%s.*"
					}[1m])
				)
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetServiceNetworkTransmitValue(ctx context.Context, svc core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_network_transmit",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					rate(container_network_transmit_bytes_total{
						namespace="%s",
						pod=~"%s.*"
					}[1m])
				)
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetServicePodCpuQuota(ctx context.Context, svc core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_pod_cpu_quota",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					container_spec_cpu_quota{
						namespace="%s",
						pod=~"%s.*"
					}
				) / 100
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetServicePodMemoryQuota(ctx context.Context, svc core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_pod_memory_quota",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					container_spec_memory_limit_bytes{
						namespace="%s",
						pod=~"%s.*"
					}
				) / (1024*1024)
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetServiceCpuUsageRange(ctx context.Context, svc core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	from, to, ok := core.GetTimeRange(ctx, rangeWindow)
	if !ok {
		return nil, fmt.Errorf("analysis time not set in context")
	}

	resultRaw, err := p.prometheusCollector.CollectQueryRange(ctx, collector.MetricQueryRange{
		Name: "service_cpu_usage_range",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					rate(container_cpu_usage_seconds_total{
						container!="",
						container!="istio-proxy",
						container!="POD",
						namespace="%s",
						pod=~"%s.*"
					}[1m])
				)
			)`, svc.Namespace, svc.Name),
		Start: from,
		End:   to,
	})
	if err != nil {
		return nil, err
	}

	return extractRangeValues(resultRaw), nil
}

func (p *PrometheusMetricsProvider) GetServiceMemoryUsageRange(ctx context.Context, svc core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	from, to, ok := core.GetTimeRange(ctx, rangeWindow)
	if !ok {
		return nil, fmt.Errorf("analysis time not set in context")
	}

	resultRaw, err := p.prometheusCollector.CollectQueryRange(ctx, collector.MetricQueryRange{
		Name: "service_memory_usage_range",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					container_memory_usage_bytes{
						container!="",
						container!="istio-proxy",
						container!="POD",
						namespace="%s",
						pod=~"%s.*"
					}
				) / (1024*1024)
			)`, svc.Namespace, svc.Name),
		Start: from,
		End:   to,
	})
	if err != nil {
		return nil, err
	}

	return extractRangeValues(resultRaw), nil
}

func (p *PrometheusMetricsProvider) GetServiceFSUsageRange(ctx context.Context, svc core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	from, to, ok := core.GetTimeRange(ctx, rangeWindow)
	if !ok {
		return nil, fmt.Errorf("analysis time not set in context")
	}

	resultRaw, err := p.prometheusCollector.CollectQueryRange(ctx, collector.MetricQueryRange{
		Name: "service_fs_usage_range",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					container_fs_usage_bytes{
						container!="",
						container!="istio-proxy",
						container!="POD",
						namespace="%s",
						pod=~"%s.*"
					}
				)
			)`, svc.Namespace, svc.Name),
		Start: from,
		End:   to,
	})
	if err != nil {
		return nil, err
	}

	return extractRangeValues(resultRaw), nil
}

func (p *PrometheusMetricsProvider) GetServiceFSWriteRange(ctx context.Context, svc core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	from, to, ok := core.GetTimeRange(ctx, rangeWindow)
	if !ok {
		return nil, fmt.Errorf("analysis time not set in context")
	}

	resultRaw, err := p.prometheusCollector.CollectQueryRange(ctx, collector.MetricQueryRange{
		Name: "service_fs_write_range",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					rate(container_fs_writes_bytes_total{
						container!="",
						container!="istio-proxy",
						container!="POD",
						namespace="%s",
						pod=~"%s.*"
					}[1m])
				)
			)`, svc.Namespace, svc.Name),
		Start: from,
		End:   to,
	})
	if err != nil {
		return nil, err
	}

	return extractRangeValues(resultRaw), nil
}

func (p *PrometheusMetricsProvider) GetServiceFSReadRange(ctx context.Context, svc core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	from, to, ok := core.GetTimeRange(ctx, rangeWindow)
	if !ok {
		return nil, fmt.Errorf("analysis time not set in context")
	}

	resultRaw, err := p.prometheusCollector.CollectQueryRange(ctx, collector.MetricQueryRange{
		Name: "service_fs_read_range",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					rate(container_fs_reads_bytes_total{
						container!="",
						container!="istio-proxy",
						container!="POD",
						namespace="%s",
						pod=~"%s.*"
					}[1m])
				)
			)`, svc.Namespace, svc.Name),
		Start: from,
		End:   to,
	})
	if err != nil {
		return nil, err
	}

	return extractRangeValues(resultRaw), nil
}

func (p *PrometheusMetricsProvider) GetServiceNetworkReceiveRange(ctx context.Context, svc core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	from, to, ok := core.GetTimeRange(ctx, rangeWindow)
	if !ok {
		return nil, fmt.Errorf("analysis time not set in context")
	}

	resultRaw, err := p.prometheusCollector.CollectQueryRange(ctx, collector.MetricQueryRange{
		Name: "service_network_receive_range",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					rate(container_network_receive_bytes_total{
						namespace="%s",
						pod=~"%s.*"
					}[1m])
				)
			)`, svc.Namespace, svc.Name),
		Start: from,
		End:   to,
	})
	if err != nil {
		return nil, err
	}

	return extractRangeValues(resultRaw), nil
}

func (p *PrometheusMetricsProvider) GetServiceNetworkTransmitRange(ctx context.Context, svc core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	from, to, ok := core.GetTimeRange(ctx, rangeWindow)
	if !ok {
		return nil, fmt.Errorf("analysis time not set in context")
	}

	resultRaw, err := p.prometheusCollector.CollectQueryRange(ctx, collector.MetricQueryRange{
		Name: "service_network_transmit_range",
		Query: fmt.Sprintf(`
			avg(
				sum by (pod) (
					rate(container_network_transmit_bytes_total{
						namespace="%s",
						pod=~"%s.*"
					}[1m])
				)
			)`, svc.Namespace, svc.Name),
		Start: from,
		End:   to,
	})
	if err != nil {
		return nil, err
	}

	return extractRangeValues(resultRaw), nil
}

func (p *PrometheusMetricsProvider) GetServiceRequestsCountRange(ctx context.Context, svc core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	from, to, ok := core.GetTimeRange(ctx, rangeWindow)
	if !ok {
		return nil, fmt.Errorf("analysis time not set in context")
	}

	resultRaw, err := p.prometheusCollector.CollectQueryRange(ctx, collector.MetricQueryRange{
		Name: "service_requests_count_range",
		Query: fmt.Sprintf(`
			sum(
				rate(istio_requests_total{
					destination_workload_namespace="%s",
					destination_workload="%s",
					destination_workload!="",
					reporter="destination"
				}[1m])
			)`, svc.Namespace, svc.Name),
		Start: from,
		End:   to,
	})
	if err != nil {
		return nil, err
	}

	return extractRangeValues(resultRaw), nil
}

func (p *PrometheusMetricsProvider) GetGraphRequestsCountValue(ctx context.Context, svcFrom, svcTo core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "graph_requests_count",
		Query: fmt.Sprintf(`
			sum(
				rate(istio_requests_total{
					source_workload_namespace="%s",
					source_workload="%s",
					destination_workload_namespace="%s",
					destination_workload="%s",
					reporter="destination"
				}[1m])
			)`, svcFrom.Namespace, svcFrom.Name, svcTo.Namespace, svcTo.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetGraphLatencyP95Value(ctx context.Context, svcFrom, svcTo core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "graph_latency_p95",
		Query: fmt.Sprintf(`
			histogram_quantile(0.95, 
				sum by (le) (
					rate(istio_request_duration_milliseconds_bucket{
						source_workload_namespace="%s",
						source_workload="%s",
						destination_workload="%s",
						destination_workload_namespace="%s"
					}[1m])
				)
			)`, svcFrom.Namespace, svcFrom.Name, svcTo.Namespace, svcTo.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetGraphLatencyP50Value(ctx context.Context, svcFrom, svcTo core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "graph_latency_p50",
		Query: fmt.Sprintf(`
			histogram_quantile(0.50, 
				sum by (le) (
					rate(istio_request_duration_milliseconds_bucket{
						source_workload_namespace="%s",
						source_workload="%s",
						destination_workload="%s",
						destination_workload_namespace="%s"
					}[1m])
				)
			)`, svcFrom.Namespace, svcFrom.Name, svcTo.Namespace, svcTo.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetServiceAverageLatency95Value(ctx context.Context, svc core.ServiceID) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_average_latency_p95",
		Query: fmt.Sprintf(`
			histogram_quantile(0.95, 
				sum by (le) (
					rate(istio_request_duration_milliseconds_bucket{
						destination_workload_namespace="%s",
						destination_workload="%s",
						destination_workload!=""
					}[1m])
				)
			)`, svc.Namespace, svc.Name),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetGraphRequestsCountRange(ctx context.Context, svcFrom, svcTo core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	from, to, ok := core.GetTimeRange(ctx, rangeWindow)
	if !ok {
		return nil, fmt.Errorf("analysis time not set in context")
	}

	resultRaw, err := p.prometheusCollector.CollectQueryRange(ctx, collector.MetricQueryRange{
		Name: "graph_requests_count_range",
		Query: fmt.Sprintf(`
			sum(
				rate(istio_requests_total{
					source_workload_namespace="%s",
					source_workload="%s",
					destination_workload_namespace="%s",
					destination_workload="%s",
					reporter="destination"
				}[1m])
			)`, svcFrom.Namespace, svcFrom.Name, svcTo.Namespace, svcTo.Name),
		Start: from,
		End:   to,
	})
	if err != nil {
		return nil, err
	}

	return extractRangeValues(resultRaw), nil
}

func (p *PrometheusMetricsProvider) GetGraphLatencyP95Range(ctx context.Context, svcFrom, svcTo core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	from, to, ok := core.GetTimeRange(ctx, rangeWindow)
	if !ok {
		return nil, fmt.Errorf("analysis time not set in context")
	}

	resultRaw, err := p.prometheusCollector.CollectQueryRange(ctx, collector.MetricQueryRange{
		Name: "graph_latency_p95_range",
		Query: fmt.Sprintf(`
			histogram_quantile(0.95, 
				sum by (le) (
					rate(istio_request_duration_milliseconds_bucket{
						source_workload_namespace="%s",
						source_workload="%s",
						destination_workload="%s",
						destination_workload_namespace="%s"
					}[1m])
				)
			)`, svcFrom.Namespace, svcFrom.Name, svcTo.Namespace, svcTo.Name),
		Start: from,
		End:   to,
	})
	if err != nil {
		return nil, err
	}

	return extractRangeValues(resultRaw), nil
}

func (p *PrometheusMetricsProvider) GetGraphLatencyP50Range(ctx context.Context, svcFrom, svcTo core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	from, to, ok := core.GetTimeRange(ctx, rangeWindow)
	if !ok {
		return nil, fmt.Errorf("analysis time not set in context")
	}

	resultRaw, err := p.prometheusCollector.CollectQueryRange(ctx, collector.MetricQueryRange{
		Name: "graph_latency_p50_range",
		Query: fmt.Sprintf(`
			histogram_quantile(0.50, 
				sum by (le) (
					rate(istio_request_duration_milliseconds_bucket{
						source_workload_namespace="%s",
						source_workload="%s",
						destination_workload="%s",
						destination_workload_namespace="%s"
					}[1m])
				)
			)`, svcFrom.Namespace, svcFrom.Name, svcTo.Namespace, svcTo.Name),
		Start: from,
		End:   to,
	})
	if err != nil {
		return nil, err
	}

	return extractRangeValues(resultRaw), nil
}

func (p *PrometheusMetricsProvider) GetServiceAverageLatency95Range(ctx context.Context, svc core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	from, to, ok := core.GetTimeRange(ctx, rangeWindow)
	if !ok {
		return nil, fmt.Errorf("analysis time not set in context")
	}

	resultRaw, err := p.prometheusCollector.CollectQueryRange(ctx, collector.MetricQueryRange{
		Name: "service_average_latency_p95_range",
		Query: fmt.Sprintf(`
			histogram_quantile(0.95, 
				sum by (le) (
					rate(istio_request_duration_milliseconds_bucket{
						destination_workload_namespace="%s",
						destination_workload="%s",
						destination_workload!=""
					}[1m])
				)
			)`, svc.Namespace, svc.Name),
		Start: from,
		End:   to,
	})
	if err != nil {
		return nil, err
	}

	return extractRangeValues(resultRaw), nil
}

func (p *PrometheusMetricsProvider) GetGlobalTotalMemoryLimit(ctx context.Context, namespace string) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "global_total_memory_limit",
		Query: fmt.Sprintf(`
			kube_resourcequota{
				namespace="%s",
				resource="limits.memory",
				type="hard"
			}`, namespace),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value / (1024 * 1024), nil
}

func (p *PrometheusMetricsProvider) GetGlobalTotalCpuLimit(ctx context.Context, namespace string) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "global_total_cpu_limit",
		Query: fmt.Sprintf(`
			kube_resourcequota{
				namespace="%s",
				resource="limits.cpu",
				type="hard"
			}`, namespace),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetGlobalTotalPodsLimit(ctx context.Context, namespace string) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "global_total_pods_limit",
		Query: fmt.Sprintf(`
			kube_resourcequota{
				namespace="%s",
				resource="pods",
				type="hard"
			}`, namespace),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetGlobalServiceMinCpu(ctx context.Context, namespace string) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_min_cpu",
		Query: fmt.Sprintf(`
			kube_limitrange{
				namespace="%s",
				constraint="min",
				resource="cpu"
			}`, namespace),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetGlobalServiceMaxCpu(ctx context.Context, namespace string) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_max_cpu",
		Query: fmt.Sprintf(`
			kube_limitrange{
				namespace="%s",
				constraint="max",
				resource="cpu"
			}`, namespace),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value, nil
}

func (p *PrometheusMetricsProvider) GetGlobalServiceMinMemory(ctx context.Context, namespace string) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_min_memory",
		Query: fmt.Sprintf(`
			kube_limitrange{
				namespace="%s",
				constraint="min",
				resource="memory"
			}`, namespace),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value / (1024 * 1024), nil
}

func (p *PrometheusMetricsProvider) GetGlobalServiceMaxMemory(ctx context.Context, namespace string) (float64, error) {
	resultRaw, err := p.prometheusCollector.CollectQuery(ctx, collector.MetricQuery{
		Name: "service_max_memory",
		Query: fmt.Sprintf(`
			kube_limitrange{
				namespace="%s",
				constraint="max",
				resource="memory"
			}`, namespace),
	})
	if err != nil {
		return 0, err
	}

	if len(resultRaw) == 0 {
		return 0, nil
	}

	return resultRaw[0].Value / (1024 * 1024), nil
}

func extractRangeValues(resultRaw []collector.MetricRangeResult) []float64 {
	if len(resultRaw) == 0 {
		return []float64{}
	}

	return resultRaw[0].Value
}
