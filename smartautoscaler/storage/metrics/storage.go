package metrics

import (
	"sync"
	"time"
)

type Point struct {
	Timestamp time.Time
	Value     float64
}

type RawTimeWindow struct {
	points    []Point
	maxPoints int
	lock      sync.Mutex
}

func NewRawTimeWindow(maxPoints int) *RawTimeWindow {
	return &RawTimeWindow{
		points:    make([]Point, 0, maxPoints),
		maxPoints: maxPoints,
	}
}

func (w *RawTimeWindow) Add(timestamp time.Time, value float64) {
	w.lock.Lock()
	defer w.lock.Unlock()

	if len(w.points) == w.maxPoints {
		w.points = w.points[1:]
	}
	w.points = append(w.points, Point{
		Timestamp: timestamp,
		Value:     value,
	})
}

func (w *RawTimeWindow) GetAll() []Point {
	w.lock.Lock()
	defer w.lock.Unlock()

	out := make([]Point, len(w.points))
	copy(out, w.points)
	return out
}

func (w *RawTimeWindow) GetRange(from, to time.Time) []Point {
	w.lock.Lock()
	defer w.lock.Unlock()

	var result []Point
	for _, p := range w.points {
		if (p.Timestamp.Equal(from) || p.Timestamp.After(from)) && p.Timestamp.Before(to) {
			result = append(result, p)
		}
	}
	return result
}

func (w *RawTimeWindow) GetLast(n int) []Point {
	w.lock.Lock()
	defer w.lock.Unlock()

	if n <= 0 || len(w.points) == 0 {
		return nil
	}
	if n >= len(w.points) {
		out := make([]Point, len(w.points))
		copy(out, w.points)
		return out
	}
	out := make([]Point, n)
	copy(out, w.points[len(w.points)-n:])
	return out
}

func (w *RawTimeWindow) GetValues() []float64 {
	w.lock.Lock()
	defer w.lock.Unlock()

	out := make([]float64, len(w.points))
	for i, p := range w.points {
		out[i] = p.Value
	}
	return out
}

func (w *RawTimeWindow) Aggregate(interval time.Duration, aggFunc func([]float64) float64) []Point {
	w.lock.Lock()
	defer w.lock.Unlock()

	if len(w.points) == 0 {
		return nil
	}

	buckets := make(map[time.Time][]float64)
	for _, p := range w.points {
		bucketTime := p.Timestamp.Truncate(interval)
		buckets[bucketTime] = append(buckets[bucketTime], p.Value)
	}

	result := make([]Point, 0, len(buckets))
	for bucketTime, values := range buckets {
		result = append(result, Point{
			Timestamp: bucketTime,
			Value:     aggFunc(values),
		})
	}

	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Timestamp.After(result[j].Timestamp) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

type ServiceMetricsStore struct {
	lock          sync.RWMutex
	metrics       map[string]*RawTimeWindow
	windowSize    time.Duration
	metricsPeriod time.Duration
}

func NewServiceMetricsStore(windowSize, metricsPeriod time.Duration) *ServiceMetricsStore {
	return &ServiceMetricsStore{
		metrics:       make(map[string]*RawTimeWindow),
		windowSize:    windowSize,
		metricsPeriod: metricsPeriod,
	}
}

func (s *ServiceMetricsStore) Add(metricName string, t time.Time, value float64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	win, ok := s.metrics[metricName]
	if !ok {
		win = NewRawTimeWindow(int(s.windowSize / s.metricsPeriod))
		s.metrics[metricName] = win
	}
	win.Add(t, value)
}

func (s *ServiceMetricsStore) GetPoints(metricName string) []Point {
	s.lock.RLock()
	defer s.lock.RUnlock()

	win, ok := s.metrics[metricName]
	if !ok {
		return nil
	}
	return win.GetAll()
}

func (s *ServiceMetricsStore) GetRange(metricName string, from, to time.Time) []Point {
	s.lock.RLock()
	defer s.lock.RUnlock()

	win, ok := s.metrics[metricName]
	if !ok {
		return nil
	}
	return win.GetRange(from, to)
}

func (s *ServiceMetricsStore) GetLast(metricName string, n int) []Point {
	s.lock.RLock()
	defer s.lock.RUnlock()

	win, ok := s.metrics[metricName]
	if !ok {
		return nil
	}
	return win.GetLast(n)
}

func (s *ServiceMetricsStore) GetValues(metricName string) []float64 {
	s.lock.RLock()
	defer s.lock.RUnlock()

	win, ok := s.metrics[metricName]
	if !ok {
		return nil
	}
	return win.GetValues()
}

func (s *ServiceMetricsStore) Aggregate(metricName string, interval time.Duration, aggFunc func([]float64) float64) []Point {
	s.lock.RLock()
	defer s.lock.RUnlock()

	win, ok := s.metrics[metricName]
	if !ok {
		return nil
	}
	return win.Aggregate(interval, aggFunc)
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

// AppMetricsStore хранит метрики всех сервисов.
type AppMetricsStore struct {
	lock          sync.RWMutex
	services      map[string]*ServiceMetricsStore
	windowSize    time.Duration
	metricsPeriod time.Duration
}

func NewAppMetricsStore(windowSize, metricsPeriod time.Duration) *AppMetricsStore {
	return &AppMetricsStore{
		services:      make(map[string]*ServiceMetricsStore),
		windowSize:    windowSize,
		metricsPeriod: metricsPeriod,
	}
}

func (a *AppMetricsStore) Add(serviceName, metricName string, t time.Time, value float64) {
	a.lock.Lock()
	defer a.lock.Unlock()

	svc, ok := a.services[serviceName]
	if !ok {
		svc = NewServiceMetricsStore(a.windowSize, a.metricsPeriod)
		a.services[serviceName] = svc
	}
	svc.Add(metricName, t, value)
}

func (a *AppMetricsStore) GetServicePoints(serviceName, metricName string) []Point {
	a.lock.RLock()
	defer a.lock.RUnlock()

	svc, ok := a.services[serviceName]
	if !ok {
		return nil
	}
	return svc.GetPoints(metricName)
}

func (a *AppMetricsStore) GetServiceRange(serviceName, metricName string, from, to time.Time) []Point {
	a.lock.RLock()
	defer a.lock.RUnlock()

	svc, ok := a.services[serviceName]
	if !ok {
		return nil
	}
	return svc.GetRange(metricName, from, to)
}

func (a *AppMetricsStore) GetServiceLast(serviceName, metricName string, n int) []Point {
	a.lock.RLock()
	defer a.lock.RUnlock()

	svc, ok := a.services[serviceName]
	if !ok {
		return nil
	}
	return svc.GetLast(metricName, n)
}

func (a *AppMetricsStore) GetServiceValues(serviceName, metricName string) []float64 {
	a.lock.RLock()
	defer a.lock.RUnlock()

	svc, ok := a.services[serviceName]
	if !ok {
		return nil
	}
	return svc.GetValues(metricName)
}

func (a *AppMetricsStore) AggregateService(serviceName, metricName string, interval time.Duration, aggFunc func([]float64) float64) []Point {
	a.lock.RLock()
	defer a.lock.RUnlock()

	svc, ok := a.services[serviceName]
	if !ok {
		return nil
	}
	return svc.Aggregate(metricName, interval, aggFunc)
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

func (a *AppMetricsStore) MetricNamesForService(serviceName string) []string {
	a.lock.RLock()
	defer a.lock.RUnlock()

	svc, ok := a.services[serviceName]
	if !ok {
		return nil
	}
	return svc.MetricNames()
}
