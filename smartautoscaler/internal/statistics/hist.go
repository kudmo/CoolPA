// Package statistics provides histogram-based tracking of latency
// observations and SLO violations. It supports efficient cumulative
// risk calculation via a cached model, and a thread-safe store for
// per-service histograms.
package statistics

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
)

// Bin represents a single latency bucket with an upper bound.
// It tracks the total number of observations and the number of
// SLO violations within that bucket. Counters are atomic for
// concurrent access.
type Bin struct {
	UpperBound float64

	total      atomic.Uint64
	violations atomic.Uint64
}

// Snapshot returns the current total and violation counts for the bin.
func (b *Bin) Snapshot() (total, violations uint64) {
	return b.total.Load(), b.violations.Load()
}

// Risk returns the violation ratio for this bin, or NaN if no
// observations have been recorded.
func (b *Bin) Risk() float64 {
	total := b.total.Load()
	if total == 0 {
		return math.NaN()
	}
	return float64(b.violations.Load()) / float64(total)
}

// modelCache holds precomputed cumulative sums for fast risk lookups.
type modelCache struct {
	cumTotal      []float64
	cumViolations []float64
	bounds        []float64

	globalRate float64
}

// Histogram represents a distribution of latencies divided into bins.
// It maintains a cached model for O(log n) risk interpolation.
type Histogram struct {
	Bins  []Bin
	cache atomic.Pointer[modelCache]
}

// NewHistogram creates a histogram from the given bin boundaries.
// Boundaries are sorted, and an overflow bin with +Inf upper bound
// is appended automatically.
func NewHistogram(bounds []float64) *Histogram {
	sort.Float64s(bounds)

	bins := make([]Bin, len(bounds)+1)
	for i, b := range bounds {
		bins[i] = Bin{UpperBound: b}
	}
	bins[len(bounds)] = Bin{UpperBound: math.Inf(1)}

	return &Histogram{Bins: bins}
}

// RebuildModel recomputes cumulative totals and the global violation
// rate, then atomically stores them in the cache for subsequent
// Risk queries.
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

// findBin returns the index of the bin whose upper bound is the
// smallest value >= latency.
func (h *Histogram) findBin(latency float64) int {
	return sort.Search(len(h.Bins), func(i int) bool {
		return latency <= h.Bins[i].UpperBound
	})
}

// Observe records a latency value and whether it violated the SLO.
// The observation is added to the appropriate bin's counters.
func (h *Histogram) Observe(latency float64, violation bool) {
	idx := h.findBin(latency)
	b := &h.Bins[idx]

	b.total.Add(1)
	if violation {
		b.violations.Add(1)
	}
}

// Risk returns an interpolated violation probability at the given
// latency x. It uses the cached model for efficiency. If no model
// has been built, it returns NaN.
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

	// Find nearest non-empty bins to the left and right for interpolation.
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

// safeBinRisk returns the violation ratio for a single bin, or the
// global rate if the bin is empty.
func safeBinRisk(c *modelCache, i int) float64 {
	total := windowTotal(c, i, i)
	if total == 0 {
		return c.globalRate
	}
	return windowViolations(c, i, i) / total
}

// windowTotal returns the total observations in the inclusive range [l,r].
func windowTotal(c *modelCache, l, r int) float64 {
	if l == 0 {
		return c.cumTotal[r]
	}
	return c.cumTotal[r] - c.cumTotal[l-1]
}

// windowViolations returns the total violations in the inclusive range [l,r].
func windowViolations(c *modelCache, l, r int) float64 {
	if l == 0 {
		return c.cumViolations[r]
	}
	return c.cumViolations[r] - c.cumViolations[l-1]
}

// HistStore provides a thread-safe mapping of service names to their
// Histogram instances.
type HistStore struct {
	services sync.Map // map[string]*Histogram
}

// Register adds a new histogram for the given service if it does not
// already exist, and returns the histogram. If the service already
// has a histogram, the existing one is returned.
func (s *HistStore) Register(service string, bounds []float64) *Histogram {
	h := NewHistogram(bounds)

	actual, loaded := s.services.LoadOrStore(service, h)
	if loaded {
		return actual.(*Histogram)
	}
	return h
}

// GetHistogram returns the histogram for the given service, or nil if
// no histogram has been registered.
func (s *HistStore) GetHistogram(service string) *Histogram {
	h, ok := s.services.Load(service)
	if ok {
		return h.(*Histogram)
	}
	return nil
}

// LogBounds generates a slice of logarithmically spaced bin boundaries
// starting at 1 and doubling until max is reached. The max value is
// always included as the last boundary. Returns nil if max <= 0.
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
