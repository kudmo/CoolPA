package metrics

import (
	"testing"
	"time"
)

func TestWindowStore_AddAndRotate(t *testing.T) {
	ws := NewTimeWindow(3, time.Minute)

	now := time.Now()

	ws.AddPoint(now, 60) // 1 RPS
	ws.AddPoint(now.Add(30*time.Second), 60)
	ws.AddPoint(now.Add(1*time.Minute), 120)
	ws.AddPoint(now.Add(2*time.Minute), 180)
	ws.AddPoint(now.Add(3*time.Minute), 240)

	values := ws.Values()

	if len(values) != 3 {
		t.Fatalf("expected 3 buckets, got %d", len(values))
	}

	if values[0] <= 0 {
		t.Fatalf("unexpected bucket value: %v", values)
	}
}

func TestWindowStore_RPSCalculation(t *testing.T) {
	ws := NewTimeWindow(2, time.Minute)
	now := time.Now()

	ws.AddPoint(now, 120)

	values := ws.Values()
	if len(values) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(values))
	}

	expected := 2.0
	if values[0] != expected {
		t.Fatalf("expected RPS %.2f, got %.2f", expected, values[0])
	}
}

func TestServiceStore_AddAndGet(t *testing.T) {
	store := NewServiceMetricsStore()
	now := time.Now()

	store.Add("svcA", now, 100)
	store.Add("svcA", now.Add(time.Minute), 200)

	values := store.GetValues("svcA")

	if len(values) == 0 {
		t.Fatal("expected non-empty values")
	}

	if store.GetValues("unknown") != nil {
		t.Fatal("expected nil for unknown service")
	}
}
