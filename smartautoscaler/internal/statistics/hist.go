package statistics

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
)

type Bin struct {
	UpperBound float64

	total      atomic.Uint64
	violations atomic.Uint64
}

func (b *Bin) Snapshot() (total, violations uint64) {
	return b.total.Load(), b.violations.Load()
}

func (b *Bin) Risk() float64 {
	total := b.total.Load()
	if total == 0 {
		return math.NaN()
	}
	return float64(b.violations.Load()) / float64(total)
}

type Histogram struct {
	Bins []Bin
}

func NewHistogram(bounds []float64) *Histogram {
	sort.Float64s(bounds)

	bins := make([]Bin, len(bounds)+1)

	for i, b := range bounds {
		bins[i] = Bin{UpperBound: b}
	}

	bins[len(bounds)] = Bin{
		UpperBound: math.Inf(1),
	}

	return &Histogram{Bins: bins}
}

func (h *Histogram) findBin(latency float64) int {
	return sort.Search(len(h.Bins), func(i int) bool {
		return latency <= h.Bins[i].UpperBound
	})
}

func (h *Histogram) Observe(latency float64, violation bool) {
	idx := h.findBin(latency)

	b := &h.Bins[idx]

	b.total.Add(1)
	if violation {
		b.violations.Add(1)
	}
}

func (h *Histogram) Risk(x float64) float64 {
	if x <= 0 {
		return 0
	}

	n := len(h.Bins)
	i := sort.Search(n, func(i int) bool {
		return x <= h.Bins[i].UpperBound
	})

	// Find nearest non-empty bins to the left and right.
	leftIdx := -1
	for j := i; j >= 0; j-- {
		if h.Bins[j].total.Load() > 0 {
			leftIdx = j
			break
		}
	}

	rightIdx := -1
	for j := i; j < n; j++ {
		if h.Bins[j].total.Load() > 0 {
			rightIdx = j
			break
		}
	}

	var li, ri float64
	var ly, ry float64

	if leftIdx == -1 {
		li = -1
		ly = 0
	} else {
		li = float64(leftIdx)
		ly = h.Bins[leftIdx].Risk() // total > 0, so no NaN
	}

	if rightIdx == -1 {
		ri = float64(n)
		ry = 1
	} else {
		ri = float64(rightIdx)
		ry = h.Bins[rightIdx].Risk()
	}

	xi := float64(i)

	if leftIdx == rightIdx && leftIdx != -1 {
		return ly
	}

	if ri == li {
		return ly
	}

	t := (xi - li) / (ri - li)

	return ly + t*(ry-ly)
}

type HistStore struct {
	services sync.Map // map[string]*Histogram
}

func (s *HistStore) Register(service string, bounds []float64) *Histogram {
	h := NewHistogram(bounds)

	actual, loaded := s.services.LoadOrStore(service, h)
	if loaded {
		return actual.(*Histogram)
	}

	return h
}

func (s *HistStore) GetHistogram(service string) *Histogram {
	h, ok := s.services.Load(service)
	if ok {
		return h.(*Histogram)
	}
	return nil
}

func LogBounds(max float64) []float64 {
	if max <= 0 {
		return nil
	}

	var bounds []float64

	v := 1.0
	for v < max {
		bounds = append(bounds, v)
		v *= 2
	}

	if len(bounds) == 0 || bounds[len(bounds)-1] != max {
		bounds = append(bounds, max)
	}

	return bounds
}
