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

type modelCache struct {
	cumTotal      []float64
	cumViolations []float64
	bounds        []float64

	globalRate float64
}

type Histogram struct {
	Bins  []Bin
	cache atomic.Pointer[modelCache]
}

func NewHistogram(bounds []float64) *Histogram {
	sort.Float64s(bounds)

	// добавляем overflow бин
	bins := make([]Bin, len(bounds)+1)

	for i, b := range bounds {
		bins[i] = Bin{UpperBound: b}
	}

	bins[len(bounds)] = Bin{
		UpperBound: math.Inf(1),
	}

	return &Histogram{Bins: bins}
}

func (h *Histogram) RebuildModel() {
	n := len(h.Bins)

	cumT := make([]float64, n)
	cumV := make([]float64, n)
	bounds := make([]float64, n)

	var sumT, sumV float64

	for i := 0; i < n; i++ {
		t := float64(h.Bins[i].total.Load())
		v := float64(h.Bins[i].violations.Load())

		sumT += t
		sumV += v

		cumT[i] = sumT
		cumV[i] = sumV
		bounds[i] = h.Bins[i].UpperBound
	}

	globalRate := 0.0
	if sumT > 0 {
		globalRate = sumV / sumT
	}

	cache := &modelCache{
		cumTotal:      cumT,
		cumViolations: cumV,
		bounds:        bounds,
		globalRate:    globalRate,
	}

	h.cache.Store(cache)
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
	cache := h.cache.Load()
	if cache == nil {
		return math.NaN()
	}

	if x <= 0 {
		return 0
	}

	i := sort.Search(len(cache.bounds), func(i int) bool {
		return x <= cache.bounds[i]
	})

	n := len(cache.bounds)

	leftIdx := -1
	for j := i; j >= 0; j-- {
		if windowTotal(cache, j, j) > 0 {
			leftIdx = j
			break
		}
	}

	rightIdx := -1
	for j := i; j < n; j++ {
		if windowTotal(cache, j, j) > 0 {
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
		ly = safeBinRisk(cache, leftIdx)
	}

	if rightIdx == -1 {
		ri = float64(n)
		ry = 1
	} else {
		ri = float64(rightIdx)
		ry = safeBinRisk(cache, rightIdx)
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

func safeBinRisk(c *modelCache, i int) float64 {
	total := windowTotal(c, i, i)
	if total == 0 {
		return c.globalRate
	}
	return windowViolations(c, i, i) / total
}

func windowTotal(c *modelCache, l, r int) float64 {
	if l == 0 {
		return c.cumTotal[r]
	}
	return c.cumTotal[r] - c.cumTotal[l-1]
}

func windowViolations(c *modelCache, l, r int) float64 {
	if l == 0 {
		return c.cumViolations[r]
	}
	return c.cumViolations[r] - c.cumViolations[l-1]
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
