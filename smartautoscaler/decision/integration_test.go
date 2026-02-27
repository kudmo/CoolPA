package decision

import (
	"testing"
	"time"

	"github.com/kudmo/CoolPA/decision/welchtest"
	"github.com/kudmo/CoolPA/storage"
)

func TestIntegration_RPSUnderutilization(t *testing.T) {
	store := storage.NewStorage()
	now := time.Now()

	// baseline — высокая нагрузка с небольшим разбросом
	baselineValues := []float64{600, 620, 580, 610, 590}

	// current — низкая нагрузка с небольшим разбросом
	currentValues := []float64{120, 130, 110, 125, 115}

	for i, v := range baselineValues {
		store.MetricsStore.Add("svcA", "rps", now.Add(time.Duration(i)*time.Minute), v)
	}

	for i, v := range currentValues {
		store.MetricsStore.Add("svcA", "rps", now.Add(time.Duration(i+len(baselineValues))*time.Minute), v)
	}

	values := store.MetricsStore.GetServiceValues("svcA", "rps")
	if len(values) < 10 {
		t.Fatal("insufficient window data")
	}

	mid := len(values) / 2
	baseline := values[:mid]
	current := values[mid:]

	res, err := welchtest.TwoSampleWelch(baseline, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TStatistic <= 0 {
		t.Fatalf("expected baseline > current, got t=%f", res.TStatistic)
	}
}

func TestIntegration_NoUnderutilization(t *testing.T) {
	store := storage.NewStorage()
	now := time.Now()

	// baseline и current имеют схожие значения
	baselineValues := []float64{500, 520, 510, 495, 505}
	currentValues := []float64{498, 515, 508, 490, 502}

	for i, v := range baselineValues {
		store.MetricsStore.Add("svcA", "rps", now.Add(time.Duration(i)*time.Minute), v)
	}

	for i, v := range currentValues {
		store.MetricsStore.Add("svcA", "rps", now.Add(time.Duration(i+len(baselineValues))*time.Minute), v)
	}

	values := store.MetricsStore.GetServiceValues("svcA", "rps")
	if len(values) < 10 {
		t.Fatal("insufficient window data")
	}

	mid := len(values) / 2
	baseline := values[:mid]
	current := values[mid:]

	res, err := welchtest.TwoSampleWelch(baseline, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// t-stat должен быть близок к 0
	if res.TStatistic > 2 || res.TStatistic < -2 {
		t.Fatalf("unexpected strong difference detected, t=%f", res.TStatistic)
	}
}
