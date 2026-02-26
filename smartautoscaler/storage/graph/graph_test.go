package graph

import (
	"math"
	"testing"
	"time"

	"github.com/kudmo/CoolPA/storage/metrics"
)

func TestAppMetricsStore_AddGetValues(t *testing.T) {
	app := metrics.NewAppMetricsStore()
	now := time.Now()

	// Добавляем метрики для двух сервисов
	app.Add("svcA", "rps", now, 10)
	app.Add("svcA", "rps", now.Add(time.Second*30), 20)
	app.Add("svcB", "rps", now, 5)

	valuesA := app.GetValues("svcA", "rps")
	if len(valuesA) != 1 {
		t.Errorf("expected 1 bucket for svcA, got %d", len(valuesA))
	}
	if valuesA[0] <= 0 {
		t.Errorf("expected positive value for svcA, got %f", valuesA[0])
	}

	valuesB := app.GetValues("svcB", "rps")
	if len(valuesB) != 1 {
		t.Errorf("expected 1 bucket for svcB, got %d", len(valuesB))
	}
}

func TestPearson(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{2, 4, 6, 8, 10}
	r := Pearson(x, y)
	if math.Abs(r-1.0) > 1e-6 {
		t.Errorf("expected 1.0, got %f", r)
	}

	y2 := []float64{5, 3, 4, 2, 1}
	r2 := Pearson(x, y2)
	if r2 > 0 {
		t.Errorf("expected negative correlation, got %f", r2)
	}
}

func TestBuildCorrelationGraph(t *testing.T) {
	app := metrics.NewAppMetricsStore()
	now := time.Now()

	for i := 0; i < 5; i++ {
		t := now.Add(time.Minute * time.Duration(i))
		app.Add("svcA", "rps", t, float64(i+1))
		app.Add("svcB", "rps", t, float64((i+1)*2))
	}

	structGraph := map[string][]string{"svcA": {"svcB"}}
	cg := BuildCorrelationGraph(app, structGraph, "rps", 0.8)

	if len(cg.edges) == 0 {
		t.Fatal("expected at least one edge in correlation graph")
	}
	weight := cg.edges["svcA"]["svcB"]
	if weight < 0.99 {
		t.Errorf("expected strong positive correlation, got %f", weight)
	}
}
