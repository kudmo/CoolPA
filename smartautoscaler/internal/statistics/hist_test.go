package statistics_test

import (
	"math"
	"testing"

	"github.com/kudmo/CoolPA/internal/statistics"
)

func TestNewHistogram(t *testing.T) {
	bounds := []float64{10, 1, 5}

	h := statistics.NewHistogram(bounds)

	if len(h.Bins) != 4 {
		t.Fatalf("expected 4 bins, got %d", len(h.Bins))
	}

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
	h := statistics.NewHistogram([]float64{10, 20})

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

func TestRisk_Basic(t *testing.T) {
	h := statistics.NewHistogram([]float64{10, 20})

	for i := range 100 {
		h.Observe(5, i%2 == 0)
	}

	r := h.Risk(5)

	if math.Abs(r-0.5) > 0.1 {
		t.Fatalf("expected ~0.5, got %.4f", r)
	}
}

func TestRisk_Interpolation(t *testing.T) {
	h := statistics.NewHistogram([]float64{10, 20})

	for range 100 {
		h.Observe(5, false)
	}

	for range 100 {
		h.Observe(15, true)
	}

	r := h.Risk(12)

	if r < 0 || r > 1 {
		t.Fatalf("expected interpolated risk in (0,1), got %.4f", r)
	}
}

func TestRisk_ExtrapolateToZero(t *testing.T) {
	h := statistics.NewHistogram([]float64{2, 10})

	for range 100 {
		h.Observe(5, true)
	}
	for range 25 {
		h.Observe(5, false)
	}

	r := h.Risk(1)

	if r >= 0.8 {
		t.Fatalf("expected risk < 0.8 due to extrapolation to zero, got %.4f", r)
	}
	if r <= 0 {
		t.Fatalf("expected risk > 0, got %.4f", r)
	}
}

func TestRisk_ExtrapolateToInf(t *testing.T) {
	h := statistics.NewHistogram([]float64{10})

	for range 100 {
		h.Observe(5, false)
	}
	for range 25 {
		h.Observe(5, true)
	}

	r := h.Risk(1000)

	if r <= 0.2 {
		t.Fatalf("expected risk > 0.2 toward +Inf, got %.4f", r)
	}
	if r >= 1 {
		t.Fatalf("expected risk < 1, got %.4f", r)
	}
}

func TestLogBounds(t *testing.T) {
	b := statistics.LogBounds(100)

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
