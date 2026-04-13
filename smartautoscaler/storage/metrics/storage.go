package metrics

import (
	"errors"
	"math"
	"sync"
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

  The store supports concurrent access with RWMutex for thread safety.
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
	PodCPUQuota
	PodMemoryLimit
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

func (r *RingWindow) Values() []float64 {
	if len(r.buckets) == 0 {
		return nil
	}

	values := make([]float64, 0, len(r.buckets))

	n := len(r.buckets)
	oldestIdx := (r.head + 1) % n

	for i := 0; i < n; i++ {
		idx := (oldestIdx + i) % n
		b := r.buckets[idx]
		if !b.Timestamp.IsZero() {
			values = append(values, b.Value)
		}
	}

	return values
}

func (r *RingWindow) ValuesRange(from, to time.Time) []float64 {
	if r.start.IsZero() || !from.Before(to) {
		return nil
	}
	n := len(r.buckets)
	if n == 0 {
		return nil
	}
	oldestIdx := (r.head + 1) % n
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

func (r *RingWindow) StdDev() float64 {
	values := r.Values()
	if len(values) < 2 {
		return 0
	}

	// Вычисляем среднее
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Вычисляем дисперсию
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return math.Sqrt(variance)
}

func (r *RingWindow) StdDevRange(from, to time.Time) float64 {
	values := r.ValuesRange(from, to)
	if len(values) < 2 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return math.Sqrt(variance)
}

func (r *RingWindow) Min() float64 {
	values := r.Values()
	if len(values) == 0 {
		return 0
	}

	minVal := values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func (r *RingWindow) Median() float64 {
	values := r.Values()
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)

	for i := 1; i < len(sorted); i++ {
		j := i
		for j > 0 && sorted[j-1] > sorted[j] {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
			j--
		}
	}

	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func (r *RingWindow) Quantile(p float64) float64 {
	values := r.Values()
	if len(values) == 0 {
		return 0
	}

	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	for i := 1; i < len(sorted); i++ {
		j := i
		for j > 0 && sorted[j-1] > sorted[j] {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
			j--
		}
	}

	pos := p * float64(len(sorted)-1)
	idx := int(pos)
	frac := pos - float64(idx)

	if idx >= len(sorted)-1 {
		return sorted[len(sorted)-1]
	}

	return sorted[idx]*(1-frac) + sorted[idx+1]*frac
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
		return make([]float64, 0)
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
	mu     sync.RWMutex
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
	s.mu.Lock()
	defer s.mu.Unlock()

	if metric >= MetricCount {
		return ErrInvalidMetric
	}

	svc := s.getOrCreateService(service)
	p := svc.getOrCreatePod(pod, s.window, s.step)

	p.metrics[metric].Add(ts, value)

	p.lastSeen = ts
	return nil
}

// SyncPods removes pods not in active list.
func (s *MetricStore) SyncPods(service string, active []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
			delete(svc.pods, pod)
		}
	}
}

// GetServices returns the list of known services.
func (s *MetricStore) GetServices() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]string, 0, len(s.services))
	for name := range s.services {
		out = append(out, name)
	}
	return out
}

// GetServicePods returns list of pods for a service, or nil if service unknown.
func (s *MetricStore) GetServicePods(service string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

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

func (s *MetricStore) GetPodMetricWindow(service, pod string, metric MetricID) (*RingWindow, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if metric >= MetricCount {
		return nil, false, ErrInvalidMetric
	}
	svc, ok := s.services[service]
	if !ok {
		return nil, false, nil
	}
	p, ok := svc.pods[pod]
	if !ok {
		return nil, false, nil
	}
	rw := p.metrics[metric]
	return rw, true, nil
}

// GetPodLastSeen returns last seen timestamp for a pod.
func (s *MetricStore) GetPodLastSeen(service, pod string) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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

// GetServiceMetricAvgSeries returns a time series of average metric values across all pods
// in a service. Each point in the series is the average of all pods' values at that
// timestamp bucket.
// Returns (series, true, nil) when service exists, (nil, false, nil) when service not found,
// or (nil, false, ErrInvalidMetric) when metric is invalid.
func (s *MetricStore) GetServiceMetricAvgSeries(service string, metric MetricID, from, to time.Time) ([]float64, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if metric >= MetricCount {
		return nil, false, ErrInvalidMetric
	}

	if !from.Before(to) {
		return nil, false, nil
	}

	svc, ok := s.services[service]
	if !ok {
		return nil, false, nil
	}

	if len(svc.pods) == 0 {
		return make([]float64, 0), true, nil
	}

	// Calculate the number of time points
	totalSteps := int(to.Sub(from) / s.step)
	if totalSteps <= 0 {
		return make([]float64, 0), true, nil
	}

	// Initialize result series
	series := make([]float64, totalSteps)
	podCount := make([]int, totalSteps)

	// Accumulate values from all pods
	for _, pod := range svc.pods {
		if pod.metrics[metric] != nil {
			podSeries := pod.metrics[metric].SeriesRange(from, to)
			if len(podSeries) == totalSteps {
				for i, val := range podSeries {
					if val > 0 || !math.IsNaN(val) {
						series[i] += val
						podCount[i]++
					}
				}
			}
		}
	}

	// Calculate averages
	for i := range series {
		if podCount[i] > 0 {
			series[i] = series[i] / float64(podCount[i])
		}
	}

	return series, true, nil
}

// GetServiceMetricAvgHead returns the average of head bucket values across all pods
// in a service. This represents the most recent sample point.
// Returns (value, true, nil) when service exists, (0, false, nil) when service not found,
// or (0, false, ErrInvalidMetric) when metric is invalid.
func (s *MetricStore) GetServiceMetricAvgHead(service string, metric MetricID) (float64, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if metric >= MetricCount {
		return 0, false, ErrInvalidMetric
	}

	svc, ok := s.services[service]
	if !ok {
		return 0, false, nil
	}

	if len(svc.pods) == 0 {
		return 0, true, nil
	}

	var sum float64
	var count int
	for _, pod := range svc.pods {
		if pod.metrics[metric] != nil {
			headValue := pod.metrics[metric].buckets[pod.metrics[metric].head].Value
			if headValue > 0 || !math.IsNaN(headValue) {
				sum += headValue
				count++
			}
		}
	}

	if count == 0 {
		return 0, true, nil
	}

	return sum / float64(count), true, nil
}

// GetServiceMetricAvgAllTime returns the average metric value across all pods and all time buckets
// for a given service. This represents the overall average across the entire history.
// Returns (value, true, nil) when service exists and has data, (0, false, nil) when service not found,
// or (0, false, ErrInvalidMetric) when metric is invalid.
func (s *MetricStore) GetServiceMetricAvg(service string, metric MetricID) (float64, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if metric >= MetricCount {
		return 0, false, ErrInvalidMetric
	}

	svc, ok := s.services[service]
	if !ok {
		return 0, false, nil
	}

	if len(svc.pods) == 0 {
		return 0, true, nil
	}

	var totalSum float64
	var totalCount int

	for _, pod := range svc.pods {
		if pod.metrics[metric] != nil {
			// Get all values from all buckets
			buckets := pod.metrics[metric].buckets
			for _, bucket := range buckets {
				val := bucket.Value
				if val > 0 || !math.IsNaN(val) {
					totalSum += val
					totalCount++
				}
			}
		}
	}

	if totalCount == 0 {
		return 0, true, nil
	}

	return totalSum / float64(totalCount), true, nil
}

// GetServiceMetricAvgValue returns the average value of a metric across all pods in a service
// over the specified time range. It calculates the mean of all individual metric points
// collected from all pods during the interval (from, to).
// Returns (average, true, nil) when service exists, (0, false, nil) when service not found,
// or (0, false, ErrInvalidMetric) when metric is invalid.
func (s *MetricStore) GetServiceMetricAvgRange(service string, metric MetricID, from, to time.Time) (float64, bool, error) {
	// Reuse existing method to get the time series of averages
	series, exists, err := s.GetServiceMetricAvgSeries(service, metric, from, to)
	if err != nil {
		return 0, false, err
	}
	if !exists {
		return 0, false, nil
	}

	if len(series) == 0 {
		return 0, true, nil
	}

	// Calculate the average of the time series averages
	var sum float64
	var count int
	for _, val := range series {
		if val > 0 || !math.IsNaN(val) {
			sum += val
			count++
		}
	}

	if count == 0 {
		return 0, true, nil
	}

	return sum / float64(count), true, nil
}
