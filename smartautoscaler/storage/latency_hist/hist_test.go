package latencyhist

import (
	"math"
	"testing"
)

func TestNewHistogram(t *testing.T) {
	bounds := []float64{10, 1, 5}

	h := NewHistogram(bounds)

	if len(h.Bins) != 4 { // +Inf бин
		t.Fatalf("expected 4 bins, got %d", len(h.Bins))
	}

	// проверяем сортировку
	if h.Bins[0].UpperBound != 1 ||
		h.Bins[1].UpperBound != 5 ||
		h.Bins[2].UpperBound != 10 {
		t.Fatalf("bounds not sorted: %+v", h.Bins)
	}

	if !math.IsInf(h.Bins[3].UpperBound, 1) {
		t.Fatalf("last bin should be +Inf")
	}
}

func TestObserve(t *testing.T) {
	h := NewHistogram([]float64{10, 20})

	h.Observe(5, false)
	h.Observe(5, true)

	total, viol := h.Bins[0].Snapshot()

	if total != 2 {
		t.Fatalf("expected total=2, got %d", total)
	}
	if viol != 1 {
		t.Fatalf("expected violations=1, got %d", viol)
	}
}

func TestRebuildModel_GlobalRate(t *testing.T) {
	h := NewHistogram([]float64{10})

	h.Observe(5, false)
	h.Observe(5, true)

	h.RebuildModel()

	cache := h.cache.Load()
	if cache == nil {
		t.Fatal("cache not built")
	}

	expected := 0.5
	if math.Abs(cache.globalRate-expected) > 1e-6 {
		t.Fatalf("expected globalRate=%.2f, got %.4f", expected, cache.globalRate)
	}
}

func TestRisk_Basic(t *testing.T) {
	h := NewHistogram([]float64{10, 20})

	// бин [0-10]
	for i := 0; i < 100; i++ {
		h.Observe(5, i%2 == 0) // ~0.5
	}

	h.RebuildModel()

	r := h.Risk(5)

	if math.Abs(r-0.5) > 0.1 {
		t.Fatalf("expected ~0.5, got %.4f", r)
	}
}

func TestRisk_Interpolation(t *testing.T) {
	h := NewHistogram([]float64{10, 20})

	// бин 1: риск ~0
	for i := 0; i < 100; i++ {
		h.Observe(5, false)
	}

	// бин 2: риск ~1
	for i := 0; i < 100; i++ {
		h.Observe(15, true)
	}

	h.RebuildModel()

	r := h.Risk(12)

	if r <= 0 || r >= 1 {
		t.Fatalf("expected interpolated risk in (0,1), got %.4f", r)
	}
}

func TestRisk_ExtrapolateToZero(t *testing.T) {
	h := NewHistogram([]float64{2, 10})

	// только один бин с риском 0.8
	for i := 0; i < 100; i++ {
		h.Observe(5, true)
	}
	for i := 0; i < 25; i++ {
		h.Observe(5, false)
	}

	h.RebuildModel()

	r := h.Risk(1)

	if r >= 0.8 {
		t.Fatalf("expected risk < 0.8 due to extrapolation to zero, got %.4f", r)
	}
	if r <= 0 {
		t.Fatalf("expected risk > 0, got %.4f", r)
	}
}

func TestRisk_ExtrapolateToInf(t *testing.T) {
	h := NewHistogram([]float64{10})

	// бин с риском 0.2
	for i := 0; i < 100; i++ {
		h.Observe(5, false)
	}
	for i := 0; i < 25; i++ {
		h.Observe(5, true)
	}

	h.RebuildModel()

	r := h.Risk(1000)

	if r <= 0.2 {
		t.Fatalf("expected risk > 0.2 toward +Inf, got %.4f", r)
	}
	if r >= 1 {
		t.Fatalf("expected risk < 1, got %.4f", r)
	}
}

func TestRisk_Empty(t *testing.T) {
	h := NewHistogram([]float64{10})

	h.RebuildModel()

	r := h.Risk(5)

	if !math.IsNaN(r) && r != 0 {
		t.Fatalf("expected NaN or 0, got %.4f", r)
	}
}

func TestLogBounds(t *testing.T) {
	b := LogBounds(100)

	expectedLast := 100.0
	if b[len(b)-1] != expectedLast {
		t.Fatalf("expected last bound = %.0f, got %.0f", expectedLast, b[len(b)-1])
	}

	for i := 1; i < len(b); i++ {
		if b[i] <= b[i-1] {
			t.Fatalf("bounds not increasing: %+v", b)
		}
	}
}
