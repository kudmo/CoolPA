package storage

import (
	"time"

	"github.com/kudmo/CoolPA/config"
	"github.com/kudmo/CoolPA/logger"
	"github.com/kudmo/CoolPA/storage/graph"
	latencyhist "github.com/kudmo/CoolPA/storage/latency_hist"
	"github.com/kudmo/CoolPA/storage/metrics"
	"github.com/kudmo/CoolPA/storage/quotas"
)

// Storage is a container for resource-level and call-graph (istio) metrics.
type Storage struct {
	GlobalConfig    *config.AppConfig
	Graph           *graph.CallGraph
	ResourceMetrics *metrics.MetricStore
	Limits          quotas.QuotasStorage
	Hist            latencyhist.HistStore
	servicePods     map[string][]string
}

// NewStorage creates a new storage with given window and step for internal windows.
func NewStorage(window, step time.Duration, config *config.AppConfig) *Storage {
	s := &Storage{
		GlobalConfig:    config,
		Graph:           graph.NewCallGraph(window, step),
		ResourceMetrics: metrics.NewMetricStore(window, step),
		Limits:          quotas.QuotasStorage{},
		Hist:            latencyhist.HistStore{},
		servicePods:     make(map[string][]string),
	}

	logger.Info("storage", "new storage created", "window", window.String(), "step", step.String())
	return s
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

// func (s *Storage) SetServiceQuota(service string, quotaID quotas.ServiceQuotaID, value int64) error {
// 	serviceQuotas, ok := s.Limits.ServiceQuotas[service]
// 	if !ok {
// 		serviceQuotas = quotas.ServiceQuotas{}
// 	}
// 	serviceQuotas.Quotas[quotaID] = value
// 	s.Limits.ServiceQuotas[service] = serviceQuotas

// 	return nil
// }

func (s *Storage) SetNamespaceLimit(limitID quotas.NamespaceLimitID, value int64) error {
	s.Limits.NamespaceLimits[limitID] = value
	return nil
}

func (s *Storage) SetServiceLimit(limitID quotas.ServiceLimitRangeID, value int64) error {
	s.Limits.ServiceLimits[limitID] = value
	return nil
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

	// Sync services in call graph (handled internally by graph package when needed)

	for svc := range active {
		if s.Hist.GetHistogram(svc) == nil {
			bounds := latencyhist.LogBounds(float64(s.GlobalConfig.SLO))
			s.Hist.Register(svc, bounds)
			logger.Debug("storage", "registered histogram for service", "service", svc)
		}
	}
	logger.Debug("storage", "sync completed", "services_count", len(active))
}
