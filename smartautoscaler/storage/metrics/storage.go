package metrics

import (
	"errors"
	"time"
)

/*
Package metrics provides an in-memory sliding window metric store
designed for autoscaling controllers.

Design goals:

  - Fixed metric set (no dynamic maps per metric)
  - Bucketed sliding window with constant memory usage
  - Pod-level source of truth
  - Incremental service-level aggregation
  - Deterministic pod removal after scale down
  - O(1) update complexity

Concurrency model:

  The store assumes single-writer semantics.
  External synchronization must be applied if accessed concurrently.
*/

type MetricID uint8

const (
	CPUUsage MetricID = iota
	MemoryUsage
	CPUQuota
	MemoryLimit
	FSUsage
	FSWrite
	FSRead
	NetworkReceive
	NetworkTransmit

	MetricCount
)

var ErrInvalidMetric = errors.New("invalid metric id")

// RingWindow represents a fixed-size bucketed sliding time window.
type RingWindow struct {
	buckets []Bucket
	head    int
	start   time.Time     // start time of current head bucket
	step    time.Duration // bucket size
}

type Bucket struct {
	Timestamp time.Time
	Value     float64
}

// NewRingWindow creates a new sliding window.
//
// windowSize - total window duration
// step       - bucket resolution
//
// Example: window=5m, step=10s -> 30 buckets
func NewRingWindow(windowSize, step time.Duration) *RingWindow {
	bucketCount := int(windowSize / step)
	return &RingWindow{
		buckets: make([]Bucket, bucketCount),
		head:    0,
		start:   time.Time{},
		step:    step,
	}
}

// advance moves the head forward according to timestamp.
func (r *RingWindow) advance(ts time.Time) {
	if r.start.IsZero() {
		r.start = ts.Truncate(r.step)
		return
	}

	diff := ts.Sub(r.start)
	steps := int64(diff / r.step)
	if steps <= 0 {
		return
	}

	if steps >= int64(len(r.buckets)) {
		// window fully expired
		for i := range r.buckets {
			r.buckets[i] = Bucket{}
		}
		r.head = 0
		r.start = ts.Truncate(r.step)
		return
	}

	for i := int64(0); i < steps; i++ {
		r.head = (r.head + 1) % len(r.buckets)
		r.buckets[r.head] = Bucket{}
	}
	r.start = r.start.Add(time.Duration(steps) * r.step)
}

// Add sets value in current bucket and returns previous value.
func (r *RingWindow) Add(ts time.Time, value float64) float64 {
	r.advance(ts)
	old := r.buckets[r.head].Value
	r.buckets[r.head] = Bucket{Timestamp: r.start, Value: value}
	return old
}

// AddDelta adds delta to current bucket.
func (r *RingWindow) AddDelta(ts time.Time, delta float64) {
	r.advance(ts)
	// ensure timestamp set for this bucket
	r.buckets[r.head].Timestamp = r.start
	r.buckets[r.head].Value += delta
}

// Sum returns sum across window.
func (r *RingWindow) Sum() float64 {
	var s float64
	for _, v := range r.buckets {
		s += v.Value
	}
	return s
}

// Avg returns arithmetic mean across buckets.
func (r *RingWindow) Avg() float64 {
	if len(r.buckets) == 0 {
		return 0
	}
	return r.Sum() / float64(len(r.buckets))
}

// Max returns maximum value across buckets.
func (r *RingWindow) Max() float64 {
	var m float64
	for i, v := range r.buckets {
		if i == 0 || v.Value > m {
			m = v.Value
		}
	}
	return m
}

func (r *RingWindow) SumRange(from, to time.Time) float64 {
	if r.start.IsZero() || !from.Before(to) {
		return 0
	}
	var sum float64
	n := len(r.buckets)
	if n == 0 {
		return 0
	}
	oldestIdx := (r.head + 1) % n
	for i := 0; i < n; i++ {
		idx := (oldestIdx + i) % n
		b := r.buckets[idx]
		if b.Timestamp.IsZero() {
			continue
		}
		if (!b.Timestamp.Before(from)) && b.Timestamp.Before(to) {
			sum += b.Value
		}
	}
	return sum
}

func (r *RingWindow) AvgRange(from, to time.Time) float64 {
	if r.start.IsZero() || !from.Before(to) {
		return 0
	}
	var sum float64
	var count int
	n := len(r.buckets)
	if n == 0 {
		return 0
	}
	oldestIdx := (r.head + 1) % n
	for i := 0; i < n; i++ {
		idx := (oldestIdx + i) % n
		b := r.buckets[idx]
		if b.Timestamp.IsZero() {
			continue
		}
		if (!b.Timestamp.Before(from)) && b.Timestamp.Before(to) {
			sum += b.Value
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func (r *RingWindow) MaxRange(from, to time.Time) float64 {
	if r.start.IsZero() || !from.Before(to) {
		return 0
	}
	var max float64
	first := true
	n := len(r.buckets)
	if n == 0 {
		return 0
	}
	oldestIdx := (r.head + 1) % n
	for i := 0; i < n; i++ {
		idx := (oldestIdx + i) % n
		b := r.buckets[idx]
		if b.Timestamp.IsZero() {
			continue
		}
		if (!b.Timestamp.Before(from)) && b.Timestamp.Before(to) {
			if first || b.Value > max {
				max = b.Value
				first = false
			}
		}
	}
	if first {
		return 0
	}
	return max
}

func (r *RingWindow) ValuesRange(from, to time.Time) []float64 {
	if r.start.IsZero() || !from.Before(to) {
		return nil
	}
	n := len(r.buckets)
	if n == 0 {
		return nil
	}
	// Индекс самой старой корзины (с наименьшим временем начала)
	oldestIdx := (r.head + 1) % n
	// Время начала самой старой корзины
	values := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		idx := (oldestIdx + i) % n
		b := r.buckets[idx]
		if b.Timestamp.IsZero() {
			continue
		}
		if (!b.Timestamp.Before(from)) && b.Timestamp.Before(to) {
			values = append(values, b.Value)
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

// SeriesRange returns a time-aligned series of values for the interval [from, to)
// with resolution equal to the window step. The returned slice has length
// int((to-from)/r.step) and contains zeros for buckets with no data.
func (r *RingWindow) SeriesRange(from, to time.Time) []float64 {
	if r.start.IsZero() || !from.Before(to) {
		return nil
	}
	n := len(r.buckets)
	if n == 0 {
		return nil
	}

	// number of discrete steps in requested interval
	totalSteps := int(to.Sub(from) / r.step)
	if totalSteps <= 0 {
		return nil
	}

	series := make([]float64, totalSteps)

	// iterate all buckets and place values into corresponding slot
	oldestIdx := (r.head + 1) % n
	for i := 0; i < n; i++ {
		idx := (oldestIdx + i) % n
		b := r.buckets[idx]
		if b.Timestamp.IsZero() {
			continue
		}
		if b.Timestamp.Before(from) || !b.Timestamp.Before(to) {
			continue
		}

		off := int(b.Timestamp.Sub(from) / r.step)
		if off >= 0 && off < totalSteps {
			series[off] = b.Value
		}
	}

	return series
}

type PodStore struct {
	metrics  [MetricCount]*RingWindow
	lastSeen time.Time
}

type ServiceStore struct {
	pods map[string]*PodStore
}

type MetricStore struct {
	services map[string]*ServiceStore

	window time.Duration
	step   time.Duration
}

// NewMetricStore creates a new store.
func NewMetricStore(window, step time.Duration) *MetricStore {
	return &MetricStore{
		services: make(map[string]*ServiceStore),
		window:   window,
		step:     step,
	}
}

func (s *MetricStore) getOrCreateService(name string) *ServiceStore {
	svc, ok := s.services[name]
	if ok {
		return svc
	}

	svc = &ServiceStore{
		pods: make(map[string]*PodStore),
	}

	s.services[name] = svc
	return svc
}

func (svc *ServiceStore) getOrCreatePod(name string, window, step time.Duration) *PodStore {
	p, ok := svc.pods[name]
	if ok {
		return p
	}

	p = &PodStore{}
	for i := MetricID(0); i < MetricCount; i++ {
		p.metrics[i] = NewRingWindow(window, step)
	}

	svc.pods[name] = p
	return p
}

// AddSample inserts a metric sample.
func (s *MetricStore) AddSample(
	service string,
	pod string,
	metric MetricID,
	ts time.Time,
	value float64,
) error {

	if metric >= MetricCount {
		return ErrInvalidMetric
	}

	svc := s.getOrCreateService(service)
	p := svc.getOrCreatePod(pod, s.window, s.step)

	p.metrics[metric].Add(ts, value)

	p.lastSeen = ts
	return nil
}

// RemovePod removes pod and subtracts its contribution.
func (s *MetricStore) RemovePod(service, pod string) {
	svc, ok := s.services[service]
	if !ok {
		return
	}

	_, ok = svc.pods[pod]
	if !ok {
		return
	}

	delete(svc.pods, pod)
}

// SyncPods removes pods not in active list.
func (s *MetricStore) SyncPods(service string, active []string) {
	svc, ok := s.services[service]
	if !ok {
		return
	}

	activeSet := make(map[string]struct{}, len(active))
	for _, p := range active {
		activeSet[p] = struct{}{}
	}

	for pod := range svc.pods {
		if _, ok := activeSet[pod]; !ok {
			s.RemovePod(service, pod)
		}
	}
}

// GetServices returns the list of known services.
func (s *MetricStore) GetServices() []string {
	out := make([]string, 0, len(s.services))
	for name := range s.services {
		out = append(out, name)
	}
	return out
}

// GetServicePods returns list of pods for a service, or nil if service unknown.
func (s *MetricStore) GetServicePods(service string) []string {
	svc, ok := s.services[service]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(svc.pods))
	for p := range svc.pods {
		out = append(out, p)
	}
	return out
}

// GetPodMetricHeadValue returns the current head bucket value for given pod metric.
// Returns (value, true, nil) when found, (0, false, nil) when service/pod not found,
// or (0,false,ErrInvalidMetric) when metric is invalid.
func (s *MetricStore) GetPodMetricHeadValue(service, pod string, metric MetricID) (float64, bool, error) {
	if metric >= MetricCount {
		return 0, false, ErrInvalidMetric
	}
	svc, ok := s.services[service]
	if !ok {
		return 0, false, nil
	}
	p, ok := svc.pods[pod]
	if !ok {
		return 0, false, nil
	}
	rw := p.metrics[metric]
	return rw.buckets[rw.head].Value, true, nil
}

// GetPodLastSeen returns last seen timestamp for a pod.
func (s *MetricStore) GetPodLastSeen(service, pod string) (time.Time, bool) {
	svc, ok := s.services[service]
	if !ok {
		return time.Time{}, false
	}
	p, ok := svc.pods[pod]
	if !ok {
		return time.Time{}, false
	}
	return p.lastSeen, true
}
