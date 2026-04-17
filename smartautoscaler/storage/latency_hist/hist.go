package latencyhist

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
	bins  []Bin
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

	return &Histogram{bins: bins}
}

func (h *Histogram) RebuildModel() {
	n := len(h.bins)

	cumT := make([]float64, n)
	cumV := make([]float64, n)
	bounds := make([]float64, n)

	var sumT, sumV float64

	for i := 0; i < n; i++ {
		t := float64(h.bins[i].total.Load())
		v := float64(h.bins[i].violations.Load())

		sumT += t
		sumV += v

		cumT[i] = sumT
		cumV[i] = sumV
		bounds[i] = h.bins[i].UpperBound
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
	return sort.Search(len(h.bins), func(i int) bool {
		return latency <= h.bins[i].UpperBound
	})
}

func (h *Histogram) Observe(latency float64, violation bool) {
	idx := h.findBin(latency)

	b := &h.bins[idx]

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

	i := sort.Search(len(cache.bounds), func(i int) bool {
		return x <= cache.bounds[i]
	})

	left := max(0, i-1)
	right := min(len(cache.bounds)-1, i+1)

	// расширяем окно пока мало данных
	const minCount = 50

	for {
		t := windowTotal(cache, left, right)
		if t >= minCount || (left == 0 && right == len(cache.bounds)-1) {
			break
		}

		if left > 0 {
			left--
		}
		if right < len(cache.bounds)-1 {
			right++
		}
	}

	total := windowTotal(cache, left, right)
	viol := windowViolations(cache, left, right)

	if total == 0 {
		return cache.globalRate
	}

	localRate := viol / total

	const alpha = 100.0
	w := total / (total + alpha)
	// w := 1 - math.Exp(-total/alpha)

	return w*localRate + (1-w)*cache.globalRate
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
