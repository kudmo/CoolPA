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
	servicePods     map[string][]string
}

// NewStorage creates a new storage with given window and step for internal windows.
func NewStorage(window, step time.Duration) *Storage {
	return &Storage{
		Graph:           graph.NewCallGraph(window, step),
		ResourceMetrics: metrics.NewMetricStore(window, step),
		servicePods:     make(map[string][]string),
	}
}

// AddResourceSample forwards a resource metric sample to the resource metric store.
func (s *Storage) AddResourceSample(service, pod string, metric metrics.MetricID, ts time.Time, value float64) error {
	if s == nil || s.ResourceMetrics == nil {
		return nil
	}
	return s.ResourceMetrics.AddSample(service, pod, metric, ts, value)
}

// AddResourceMetric forwards a resource metric sample to the resource metric store.
func (s *Storage) AddResourceMetric(service, pod string, metric metrics.MetricID, ts time.Time, value float64) error {
	if s == nil || s.ResourceMetrics == nil {
		return nil
	}
	return s.ResourceMetrics.AddSample(service, pod, metric, ts, value)
}

// AddIstioServiceMetric records a service-level istio metric (by id) into the call graph.
func (s *Storage) AddIstioServiceMetric(service string, ts time.Time, id graph.MetricID, value float64) error {
	if s == nil || s.Graph == nil {
		return nil
	}
	return s.Graph.AddServiceMetric(service, ts, id, value)
}

// AddIstioEdgeMetric records an edge-level istio metric (by id) into the call graph.
func (s *Storage) AddIstioEdgeMetric(from, to string, ts time.Time, id graph.MetricID, value float64) error {
	if s == nil || s.Graph == nil {
		return nil
	}
	return s.Graph.AddEdgeMetric(from, to, ts, id, value)
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
