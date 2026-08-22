package storage

import (
	"fmt"
	"sync"
	"time"
)

type MetricsCache struct {
	mu       sync.RWMutex
	services map[string]*serviceMetricsStore
	config   CacheConfig
}

func (c *MetricsCache) getOrCreateService(service string) *serviceMetricsStore {
	svc, ok := c.services[service]
	if ok {
		return svc
	}

	svc = &serviceMetricsStore{}

	c.services[service] = svc
	return svc
}

func (c *MetricsCache) AddMetricValue(service, metricId string, value float64, ts time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	svc := c.getOrCreateService(service)
	svc.AddMetricValue(metricId, value, ts)
}

func (c *MetricsCache) GetMetricValue(service, metricId string) (float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	svc, ok := c.services[service]
	if !ok {
		return 0, fmt.Errorf("Getting unexisting service %s", service)
	}

	return svc.GetMetricValue(metricId)
}

func (c *MetricsCache) GetMetricRange(service, metricId string, from, to time.Time) ([]float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	svc, ok := c.services[service]
	if !ok {
		return nil, fmt.Errorf("Getting unexisting service %s", service)
	}

	return svc.GetMetricRange(metricId, from, to)
}

type serviceMetricsStore struct {
	Config  CacheConfig
	metrics sync.Map //map[string] *RingWindow
}

func (s *serviceMetricsStore) registerMetric(metricId string) *RingWindow {
	rw := NewRingWindow(s.Config.Window, s.Config.Step)
	actual, loaded := s.metrics.LoadOrStore(metricId, rw)
	if loaded {
		return actual.(*RingWindow)
	}
	return rw
}

func (s *serviceMetricsStore) AddMetricValue(metricId string, value float64, ts time.Time) {
	window := s.registerMetric(metricId)
	window.Add(ts, value)
}

func (s *serviceMetricsStore) GetMetricValue(metricId string) (float64, error) {
	window, ok := s.metrics.Load(metricId)
	if !ok {
		return 0, fmt.Errorf("Getting unexisting metric %s", metricId)
	}
	return window.(*RingWindow).HeadValue(), nil
}

func (s *serviceMetricsStore) GetMetricRange(metricId string, from, to time.Time) ([]float64, error) {
	window, ok := s.metrics.Load(metricId)
	if !ok {
		return nil, fmt.Errorf("Getting unexisting metric %s", metricId)
	}
	return window.(*RingWindow).ValuesRange(from, to), nil
}
