package welchtest

import (
	"testing"
)

func TestWelchTest_DifferentMeans(t *testing.T) {
	baseline := []float64{10, 12, 11, 13, 12}
	current := []float64{5, 6, 7, 6, 5}

	res, err := TwoSampleWelch(baseline, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TStatistic <= 0 {
		t.Fatalf("expected positive t-stat (baseline > current), got %f", res.TStatistic)
	}
}

func TestWelchTest_SimilarMeans(t *testing.T) {
	a := []float64{10, 11, 12, 10, 11}
	b := []float64{10, 11, 12, 10, 11}

	res, err := TwoSampleWelch(a, b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TStatistic > 0.5 || res.TStatistic < -0.5 {
		t.Fatalf("expected t-stat close to 0, got %f", res.TStatistic)
	}
}

func TestWelchTest_TooSmallSample(t *testing.T) {
	a := []float64{10}
	b := []float64{5}

	_, err := TwoSampleWelch(a, b)
	if err == nil {
		t.Fatal("expected error for too small samples")
	}
}

func TestWelchTest_ZeroVariance(t *testing.T) {
	a := []float64{10, 10, 10}
	b := []float64{5, 5, 5}

	_, err := TwoSampleWelch(a, b)
	if err == nil {
		t.Fatal("expected zero variance error")
	}
}
