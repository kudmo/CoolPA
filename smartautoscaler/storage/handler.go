package storage

import (
	"log/slog"
	"regexp"
	"strings"

	"github.com/kudmo/CoolPA/collector"
	"github.com/kudmo/CoolPA/storage/graph"
	"github.com/kudmo/CoolPA/storage/metrics"
)

// PodNameParser остался без изменений
type PodNameParser struct {
	patterns []*regexp.Regexp
}

func NewPodNameParser() *PodNameParser {
	return &PodNameParser{
		patterns: []*regexp.Regexp{
			// Deployment: app-name-7b9f8d4d5c-abc12
			regexp.MustCompile(`^(.*)-[a-z0-9]{9,10}-[a-z0-9]{5,10}$`),
			// StatefulSet: app-name-0, app-name-1
			regexp.MustCompile(`^(.*)-[0-9]+$`),
			// Job: app-name-abc123
			regexp.MustCompile(`^(.*)-[a-z0-9]{5,10}$`),
			// DaemonSet: app-name-abc12
			regexp.MustCompile(`^(.*)-[a-z0-9]{5}$`),
		},
	}
}

func (p *PodNameParser) ExtractServiceName(podName string) string {
	for _, pattern := range p.patterns {
		if matches := pattern.FindStringSubmatch(podName); matches != nil {
			return matches[1]
		}
	}

	return ExtractServiceNameFromPod(podName)
}

func ExtractServiceNameFromPod(podName string) string {
	parts := strings.Split(podName, "-")
	if len(parts) < 2 {
		return podName
	}

	lastPart := parts[len(parts)-1]

	if isHashOrNumber(lastPart) {
		if len(parts) > 2 && isHashOrNumber(parts[len(parts)-2]) {
			return strings.Join(parts[:len(parts)-2], "-")
		}
		return strings.Join(parts[:len(parts)-1], "-")
	}

	return podName
}

func isHashOrNumber(s string) bool {
	isNum := true
	for _, c := range s {
		if c < '0' || c > '9' {
			isNum = false
			break
		}
	}
	if isNum {
		return true
	}
	if len(s) >= 5 && len(s) <= 12 {
		for _, c := range s {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
		return true
	}

	return false
}

type StorageHandler struct {
	Store  *Storage
	parser *PodNameParser
}

func NewStorageHandler(store *Storage) *StorageHandler {
	return &StorageHandler{
		Store:  store,
		parser: NewPodNameParser(),
	}
}

func extractResourceMetricID(metricName string) (metrics.MetricID, bool) {
	switch metricName {
	case "container_cpu_usage":
		return metrics.CPUUsage, true
	case "container_memory_usage":
		return metrics.MemoryUsage, true
	case "container_fs_usage":
		return metrics.FSUsage, true
	case "container_fs_write":
		return metrics.FSWrite, true
	case "container_fs_read":
		return metrics.FSRead, true
	case "container_network_receive":
		return metrics.NetworkReceive, true
	case "container_network_transmit":
		return metrics.NetworkTransmit, true
	case "container_cpu_quota":
		return metrics.CPUQuota, true
	case "container_memory_limit":
		return metrics.MemoryLimit, true
	default:
		return 0, false
	}
}

func metricIDToName(metricID metrics.MetricID) string {
	switch metricID {
	case metrics.CPUUsage:
		return "container_cpu_usage"
	case metrics.MemoryUsage:
		return "container_memory_usage"
	case metrics.FSUsage:
		return "container_fs_usage"
	case metrics.FSWrite:
		return "container_fs_write"
	case metrics.FSRead:
		return "container_fs_read"
	case metrics.NetworkReceive:
		return "container_network_receive"
	case metrics.NetworkTransmit:
		return "container_network_transmit"
	case metrics.CPUQuota:
		return "container_cpu_quota"
	case metrics.MemoryLimit:
		return "container_memory_limit"
	default:
		return "unknown_metric"
	}
}

func (h *StorageHandler) handleResource(result collector.MetricResult) {
	pod, ok := result.Labels["pod"]
	if !ok {
		slog.Warn("Missing pod label for resource metric",
			"query_name", result.QueryName,
			"timestamp", result.Timestamp,
			"labels", result.Labels,
		)
		return
	}

	service := h.parser.ExtractServiceName(pod)
	metricID, ok := extractResourceMetricID(result.QueryName)
	if !ok {
		slog.Debug("Unknown resource metric, skipping",
			"query_name", result.QueryName,
			"pod", pod,
			"service", service,
		)
		return
	}

	h.Store.AddResourceSample(service, pod, metricID, result.Timestamp, result.Value)

	slog.Debug("Resource metric processed",
		"service", service,
		"pod", pod,
		"metric", metricIDToName(metricID),
		"value", result.Value,
		"timestamp", result.Timestamp,
	)
}

func extractIstioMetricID(metricName string) (graph.MetricID, bool) {
	switch metricName {
	case "istio_request_duration_p95":
		return graph.EdgeLatency95, true
	case "istio_request_duration_p50":
		return graph.EdgeLatency50, true
	case "istio_requests_total":
		return graph.ServiceRequestCount, true
	case "istio_tcp_sent_bytes_total":
		return graph.ServiceBytesSent, true
	case "istio_tcp_received_bytes_total":
		return graph.ServiceBytesReceived, true
	default:
		return 0, false
	}
}

func (h *StorageHandler) handleIstio(result collector.MetricResult) {
	dst, ok := result.Labels["destination_app"]
	if !ok {
		slog.Warn("Missing destination_app label for istio metric",
			"query_name", result.QueryName,
			"labels", result.Labels,
		)
		return
	}

	switch result.QueryName {
	case "istio_request_duration_p95":
		src, ok := result.Labels["source_app"]
		if !ok {
			slog.Warn("Missing source_app label for istio_request_duration_p95",
				"destination", dst,
				"labels", result.Labels,
			)
			return
		}
		h.Store.AddIstioEdgeMetric(src, dst, result.Timestamp, graph.EdgeLatency95, result.Value)
		slog.Debug("Istio edge metric processed",
			"source", src,
			"destination", dst,
			"metric", "latency_p95",
			"value", result.Value,
		)

	case "istio_request_duration_p50":
		src, ok := result.Labels["source_app"]
		if !ok {
			slog.Warn("Missing source_app label for istio_request_duration_p50",
				"destination", dst,
				"labels", result.Labels,
			)
			return
		}
		h.Store.AddIstioEdgeMetric(src, dst, result.Timestamp, graph.EdgeLatency50, result.Value)
		slog.Debug("Istio edge metric processed",
			"source", src,
			"destination", dst,
			"metric", "latency_p50",
			"value", result.Value,
		)

	case "istio_requests_total":
		h.Store.AddIstioServiceMetric(dst, result.Timestamp, graph.ServiceRequestCount, result.Value)
		slog.Debug("Istio service metric processed",
			"service", dst,
			"metric", "request_count",
			"value", result.Value,
		)

	case "istio_tcp_sent_bytes_total":
		h.Store.AddIstioServiceMetric(dst, result.Timestamp, graph.ServiceBytesSent, result.Value)
		slog.Debug("Istio service metric processed",
			"service", dst,
			"metric", "bytes_sent",
			"value", result.Value,
		)

	case "istio_tcp_received_bytes_total":
		h.Store.AddIstioServiceMetric(dst, result.Timestamp, graph.ServiceBytesReceived, result.Value)
		slog.Debug("Istio service metric processed",
			"service", dst,
			"metric", "bytes_received",
			"value", result.Value,
		)

	default:
		slog.Warn("Unknown istio metric",
			"query_name", result.QueryName,
			"destination", dst,
		)
	}
}

func (h *StorageHandler) handlePodsInfo(result collector.MetricResult) {
	pod, ok := result.Labels["pod"]
	if !ok {
		slog.Warn("Missing pod label for kube_pod_info",
			"labels", result.Labels,
		)
		return
	}

	service := h.parser.ExtractServiceName(pod)
	h.Store.servicePods[service] = append(h.Store.servicePods[service], pod)

	slog.Debug("Pod info processed",
		"service", service,
		"pod", pod,
	)
}

func (h *StorageHandler) Handle(result collector.MetricResult) {
	if _, ok := extractResourceMetricID(result.QueryName); ok {
		h.handleResource(result)
	} else if _, ok := extractIstioMetricID(result.QueryName); ok {
		h.handleIstio(result)
	} else if result.QueryName == "kube_pod_info" {
		h.handlePodsInfo(result)
	} else {
		slog.Debug("Unknown metric type, skipping",
			"query_name", result.QueryName,
			"labels", result.Labels,
		)
	}
}

func (h *StorageHandler) HandleBatch(results []collector.MetricResult) {
	for _, s := range h.Store.Graph.GetServices() {
		h.Store.servicePods[s] = make([]string, 0)
	}

	for _, result := range results {
		h.Handle(result)
	}

	h.Store.Sync(h.Store.servicePods)

	slog.Debug("=== Storage State Debug ===")

	for _, svc := range h.Store.ResourceMetrics.GetServices() {
		pods := h.Store.ResourceMetrics.GetServicePods(svc)
		slog.Debug("Service resource metrics",
			"service", svc,
			"pods_count", len(pods),
		)

		for _, pod := range pods {
			podLog := slog.With("service", svc, "pod", pod)

			for metricID := metrics.MetricID(0); metricID < metrics.MetricCount; metricID++ {
				value, found, err := h.Store.ResourceMetrics.GetPodMetricHeadValue(svc, pod, metricID)
				if err != nil {
					podLog.Error("Failed to get pod metric",
						"metric_name", metricIDToName(metricID),
						"error", err,
					)
				} else if found {
					podLog.Debug("Pod metric",
						"metric_name", metricIDToName(metricID),
						"value", value,
					)
				}
			}
		}
	}

	// Graph metrics debug
	for _, svc := range h.Store.Graph.GetServices() {
		node, ok := h.Store.Graph.GetNode(svc)
		if !ok {
			slog.Error("Failed to get service node",
				"service", svc,
			)
			continue
		}

		svcLog := slog.With("service", svc)
		svcLog.Debug("Service graph metrics",
			"request_count_avg", node.RequestCount.Avg(),
			"bytes_sent_sum", node.BytesSent.Sum(),
			"bytes_received_sum", node.BytesReceived.Sum(),
		)

		for dst, edge := range node.OutboundEdges {
			svcLog.Debug("Service edge metrics",
				"destination", dst,
				"latency_p95_avg", edge.Latency95.Avg(),
				"latency_p50_avg", edge.Latency50.Avg(),
			)
		}
	}

	slog.Debug("=== End Storage State Debug ===")
}
