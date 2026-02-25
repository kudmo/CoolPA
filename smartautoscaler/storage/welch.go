package storage

import (
	"sync"
	"time"
)

const DefaultBucketDuration = time.Minute

type Bucket struct {
	Timestamp time.Time
	Metrics   float64
}

type WindowStore struct {
	buckets        []Bucket
	bucketDuration time.Duration
	maxBuckets     int
	lock           sync.Mutex
}

func NewWindowStore(maxBuckets int, bucketDuration time.Duration) *WindowStore {
	return &WindowStore{
		buckets:        make([]Bucket, 0, maxBuckets),
		bucketDuration: bucketDuration,
		maxBuckets:     maxBuckets,
	}
}

func (w *WindowStore) AddPoint(timestamp time.Time, metrics float64) {
	w.lock.Lock()
	defer w.lock.Unlock()

	if len(w.buckets) == 0 ||
		timestamp.Sub(w.buckets[len(w.buckets)-1].Timestamp) >= w.bucketDuration {

		if len(w.buckets) == w.maxBuckets {
			w.buckets = w.buckets[1:]
		}

		w.buckets = append(w.buckets, Bucket{
			Timestamp: timestamp,
			Metrics:   metrics,
		})
		return
	}

	w.buckets[len(w.buckets)-1].Metrics += metrics
}

func (w *WindowStore) Values() []float64 {
	w.lock.Lock()
	defer w.lock.Unlock()

	out := make([]float64, len(w.buckets))
	for i := range w.buckets {
		out[i] = w.buckets[i].Metrics / w.bucketDuration.Seconds()
	}
	return out
}

type ServiceStore struct {
	lock    sync.RWMutex
	windows map[string]*WindowStore
}

func NewServiceStore() *ServiceStore {
	return &ServiceStore{
		windows: make(map[string]*WindowStore),
	}
}

func (s *ServiceStore) Add(service string, t time.Time, reqs float64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	ws, ok := s.windows[service]
	if !ok {
		ws = NewWindowStore(60, DefaultBucketDuration)
		s.windows[service] = ws
	}
	ws.AddPoint(t, reqs)
}

func (s *ServiceStore) GetValues(service string) []float64 {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ws, ok := s.windows[service]
	if !ok {
		return nil
	}
	return ws.Values()
}
