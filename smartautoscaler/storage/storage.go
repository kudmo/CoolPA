package storage

import (
	"time"

	"github.com/kudmo/CoolPA/storage/graph"
	"github.com/kudmo/CoolPA/storage/metrics"
)

// Storage is a container for resource-level and call-graph (istio) metrics.
type Storage struct {
	Graph           *graph.CallGraph
	ResourceMetrics *metrics.MetricStore
}

// NewStorage creates a new storage with given window and step for internal windows.
func NewStorage(window, step time.Duration) *Storage {
	return &Storage{
		Graph:           graph.NewCallGraph(window, step),
		ResourceMetrics: metrics.NewMetricStore(window, step),
	}
}

// AddResourceSample forwards a resource metric sample to the resource metric store.
func (s *Storage) AddResourceSample(service, pod string, metric metrics.MetricID, ts time.Time, value float64) error {
	if s == nil || s.ResourceMetrics == nil {
		return nil
	}
	return s.ResourceMetrics.AddSample(service, pod, metric, ts, value)
}

// AddIstioServiceSample records service-level istio metrics into the call graph.
func (s *Storage) AddIstioServiceSample(service string, ts time.Time, requests, duration, bytesSent, bytesReceived float64) {
	if s == nil || s.Graph == nil {
		return
	}
	s.Graph.AddServiceSample(service, ts, requests, duration, bytesSent, bytesReceived)
}

// AddIstioEdgeSample records edge (from->to) istio latency metrics into the call graph.
func (s *Storage) AddIstioEdgeSample(from, to string, ts time.Time, p95, p50 float64) {
	if s == nil || s.Graph == nil {
		return
	}
	s.Graph.AddEdgeSample(from, to, ts, p95, p50)
}

// Sync synchronizes active services and their pods. The input map maps service->list of active pods.
// It will call the resource store to sync pods per-service and will sync services in the call graph.
func (s *Storage) Sync(active map[string][]string) {
	if s == nil {
		return
	}

	// Sync pods per service in resource metrics
	if s.ResourceMetrics != nil {
		for svc, pods := range active {
			s.ResourceMetrics.SyncPods(svc, pods)
		}
	}

	// Sync services in call graph
	if s.Graph != nil {
		services := make([]string, 0, len(active))
		for svc := range active {
			services = append(services, svc)
		}
		s.Graph.SyncServices(services)
	}
}
