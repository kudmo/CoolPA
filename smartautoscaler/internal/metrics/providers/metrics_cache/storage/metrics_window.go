package storage

import (
	"time"
)

type RingWindow struct {
	buckets []Bucket
	head    int
	start   time.Time
	step    time.Duration
}

type Bucket struct {
	Timestamp time.Time
	Value     float64
}

func NewRingWindow(windowSize, step time.Duration) *RingWindow {
	bucketCount := int(windowSize / step)
	return &RingWindow{
		buckets: make([]Bucket, bucketCount),
		head:    0,
		start:   time.Time{},
		step:    step,
	}
}

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

func (r *RingWindow) HeadValue() float64 {
	return r.buckets[r.head].Value
}
