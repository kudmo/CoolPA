package metrics

import (
	"sync"
	"time"
)

const DefaultBucketDuration = time.Minute

type Bucket struct {
	Timestamp time.Time
	Value     float64
}

type TimeWindow struct {
	buckets        []Bucket
	bucketDuration time.Duration
	maxBuckets     int
	lock           sync.Mutex
}

func NewTimeWindow(maxBuckets int, bucketDuration time.Duration) *TimeWindow {
	return &TimeWindow{
		buckets:        make([]Bucket, 0, maxBuckets),
		bucketDuration: bucketDuration,
		maxBuckets:     maxBuckets,
	}
}

func (w *TimeWindow) AddPoint(timestamp time.Time, value float64) {
	w.lock.Lock()
	defer w.lock.Unlock()

	if len(w.buckets) == 0 || timestamp.Sub(w.buckets[len(w.buckets)-1].Timestamp) >= w.bucketDuration {
		if len(w.buckets) == w.maxBuckets {
			w.buckets = w.buckets[1:]
		}
		w.buckets = append(w.buckets, Bucket{
			Timestamp: timestamp,
			Value:     value,
		})
		return
	}

	w.buckets[len(w.buckets)-1].Value += value
}

func (w *TimeWindow) Values() []float64 {
	w.lock.Lock()
	defer w.lock.Unlock()

	out := make([]float64, len(w.buckets))
	for i := range w.buckets {
		out[i] = w.buckets[i].Value / w.bucketDuration.Seconds()
	}
	return out
}

type ServiceMetricsStore struct {
	lock    sync.RWMutex
	metrics map[string]*TimeWindow
}

func NewServiceMetricsStore() *ServiceMetricsStore {
	return &ServiceMetricsStore{
		metrics: make(map[string]*TimeWindow),
	}
}

func (s *ServiceMetricsStore) Add(metricName string, t time.Time, value float64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	win, ok := s.metrics[metricName]
	if !ok {
		win = NewTimeWindow(60, DefaultBucketDuration)
		s.metrics[metricName] = win
	}
	win.AddPoint(t, value)
}

func (s *ServiceMetricsStore) GetValues(metricName string) []float64 {
	s.lock.RLock()
	defer s.lock.RUnlock()

	win, ok := s.metrics[metricName]
	if !ok {
		return nil
	}
	return win.Values()
}

func (s *ServiceMetricsStore) MetricNames() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	names := make([]string, 0, len(s.metrics))
	for name := range s.metrics {
		names = append(names, name)
	}
	return names
}

type AppMetricsStore struct {
	lock     sync.RWMutex
	services map[string]*ServiceMetricsStore
}

func NewAppMetricsStore() *AppMetricsStore {
	return &AppMetricsStore{
		services: make(map[string]*ServiceMetricsStore),
	}
}

func (a *AppMetricsStore) Add(serviceName, metricName string, t time.Time, value float64) {
	a.lock.Lock()
	defer a.lock.Unlock()

	svc, ok := a.services[serviceName]
	if !ok {
		svc = NewServiceMetricsStore()
		a.services[serviceName] = svc
	}
	svc.Add(metricName, t, value)
}

func (a *AppMetricsStore) GetValues(serviceName, metricName string) []float64 {
	a.lock.RLock()
	defer a.lock.RUnlock()

	svc, ok := a.services[serviceName]
	if !ok {
		return nil
	}
	return svc.GetValues(metricName)
}

func (a *AppMetricsStore) ServiceNames() []string {
	a.lock.RLock()
	defer a.lock.RUnlock()

	names := make([]string, 0, len(a.services))
	for name := range a.services {
		names = append(names, name)
	}
	return names
}
