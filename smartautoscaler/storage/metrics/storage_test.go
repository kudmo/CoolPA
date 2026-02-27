package metrics

import (
	"testing"
	"time"
)

func TestRawTimeWindow_AddAndRotate(t *testing.T) {
	// Храним максимум 3 точки
	window := NewRawTimeWindow(3)
	now := time.Now()

	// Добавляем 5 точек (должны остаться только последние 3)
	window.Add(now, 60.0)
	window.Add(now.Add(30*time.Second), 60.0)
	window.Add(now.Add(1*time.Minute), 120.0)
	window.Add(now.Add(2*time.Minute), 180.0)
	window.Add(now.Add(3*time.Minute), 240.0)

	points := window.GetAll()
	if len(points) != 3 {
		t.Fatalf("expected 3 points after rotation, got %d", len(points))
	}

	// Проверяем, что остались последние 3 (с минутами 1,2,3)
	expectedMinutes := []int{1, 2, 3}
	for i, p := range points {
		gotMin := int(p.Timestamp.Sub(now).Minutes())
		if gotMin != expectedMinutes[i] {
			t.Errorf("expected point at minute %d, got minute %d", expectedMinutes[i], gotMin)
		}
	}

	// Проверяем значения
	expectedValues := []float64{120.0, 180.0, 240.0}
	for i, v := range expectedValues {
		if points[i].Value != v {
			t.Errorf("expected value %.2f at position %d, got %.2f", v, i, points[i].Value)
		}
	}
}

func TestRawTimeWindow_GetRange(t *testing.T) {
	window := NewRawTimeWindow(10)
	now := time.Now()

	// Добавляем точки каждые 15 секунд в течение 2 минут
	for i := 0; i < 8; i++ {
		window.Add(now.Add(time.Duration(i)*15*time.Second), float64(i*10))
	}

	// Запрашиваем интервал с 30 секунд до 90 секунд
	from := now.Add(30 * time.Second)
	to := now.Add(90 * time.Second)
	rangePoints := window.GetRange(from, to)

	// Ожидаем точки на 30, 45, 60, 75 секундах (4 точки)
	if len(rangePoints) != 4 {
		t.Fatalf("expected 4 points in range, got %d", len(rangePoints))
	}

	// Проверяем, что все точки внутри интервала
	for _, p := range rangePoints {
		if p.Timestamp.Before(from) || !p.Timestamp.Before(to) {
			t.Errorf("point at %v outside range [%v, %v)", p.Timestamp, from, to)
		}
	}
}

func TestRawTimeWindow_GetLast(t *testing.T) {
	window := NewRawTimeWindow(10)
	now := time.Now()

	// Добавляем 5 точек
	for i := 0; i < 5; i++ {
		window.Add(now.Add(time.Duration(i)*time.Minute), float64(i*100))
	}

	// Получаем последние 3 точки
	last3 := window.GetLast(3)
	if len(last3) != 3 {
		t.Fatalf("expected 3 last points, got %d", len(last3))
	}

	// Проверяем, что это действительно последние
	expectedValues := []float64{200.0, 300.0, 400.0} // индексы 2,3,4
	for i, v := range expectedValues {
		if last3[i].Value != v {
			t.Errorf("expected last point value %.2f, got %.2f", v, last3[i].Value)
		}
	}

	// Запрашиваем больше, чем есть
	last10 := window.GetLast(10)
	if len(last10) != 5 {
		t.Fatalf("expected all 5 points when requesting more, got %d", len(last10))
	}
}

func TestRawTimeWindow_GetValues(t *testing.T) {
	window := NewRawTimeWindow(5)
	now := time.Now()

	// Добавляем точки
	for i := 0; i < 3; i++ {
		window.Add(now.Add(time.Duration(i)*time.Minute), float64(i*50))
	}

	values := window.GetValues()
	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(values))
	}

	expected := []float64{0.0, 50.0, 100.0}
	for i, v := range expected {
		if values[i] != v {
			t.Errorf("expected value %.2f at position %d, got %.2f", v, i, values[i])
		}
	}
}

func TestRawTimeWindow_Aggregate(t *testing.T) {
	window := NewRawTimeWindow(20)
	now := time.Now().Truncate(time.Hour) // фиксируем начало часа для предсказуемости

	// Добавляем точки каждые 15 секунд в течение 3 минут (12 точек)
	for i := 0; i < 12; i++ {
		window.Add(now.Add(time.Duration(i)*15*time.Second), float64(i*5))
	}

	// Агрегируем по минутам с функцией среднего
	minuteAvg := window.Aggregate(time.Minute, func(vals []float64) float64 {
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		return sum / float64(len(vals))
	})

	// Ожидаем 3 бакета (минуты 0,1,2)
	if len(minuteAvg) != 3 {
		t.Fatalf("expected 3 aggregated buckets, got %d", len(minuteAvg))
	}

	// Проверяем средние значения
	// Минута 0: точки 0-3 (значения 0,5,10,15) среднее = 7.5
	// Минута 1: точки 4-7 (значения 20,25,30,35) среднее = 27.5
	// Минута 2: точки 8-11 (значения 40,45,50,55) среднее = 47.5
	expectedAvg := []float64{7.5, 27.5, 47.5}
	for i, v := range expectedAvg {
		if minuteAvg[i].Value != v {
			t.Errorf("minute %d: expected avg %.2f, got %.2f", i, v, minuteAvg[i].Value)
		}
	}

	// Агрегируем с функцией максимума
	minuteMax := window.Aggregate(time.Minute, func(vals []float64) float64 {
		max := vals[0]
		for _, v := range vals[1:] {
			if v > max {
				max = v
			}
		}
		return max
	})

	expectedMax := []float64{15.0, 35.0, 55.0}
	for i, v := range expectedMax {
		if minuteMax[i].Value != v {
			t.Errorf("minute %d: expected max %.2f, got %.2f", i, v, minuteMax[i].Value)
		}
	}
}

func TestServiceMetricsStore_AddAndGet(t *testing.T) {
	windowSize := 5 * time.Minute
	period := 15 * time.Second
	store := NewServiceMetricsStore(windowSize, period)
	now := time.Now()

	// Добавляем несколько точек для разных метрик
	store.Add("rps", now, 100.0)
	store.Add("rps", now.Add(15*time.Second), 120.0)
	store.Add("rps", now.Add(30*time.Second), 110.0)

	store.Add("latency_p99", now, 50.0)
	store.Add("latency_p99", now.Add(15*time.Second), 55.0)

	// Проверяем GetValues
	rpsValues := store.GetValues("rps")
	if len(rpsValues) != 3 {
		t.Fatalf("expected 3 rps values, got %d", len(rpsValues))
	}
	expectedRPS := []float64{100.0, 120.0, 110.0}
	for i, v := range expectedRPS {
		if rpsValues[i] != v {
			t.Errorf("rps[%d]: expected %.2f, got %.2f", i, v, rpsValues[i])
		}
	}

	// Проверяем GetPoints
	latPoints := store.GetPoints("latency_p99")
	if len(latPoints) != 2 {
		t.Fatalf("expected 2 latency points, got %d", len(latPoints))
	}

	// Проверяем неизвестную метрику
	if store.GetValues("unknown") != nil {
		t.Error("expected nil for unknown metric")
	}

	// Проверяем MetricNames
	names := store.MetricNames()
	expectedNames := map[string]bool{"rps": true, "latency_p99": true}
	if len(names) != 2 {
		t.Fatalf("expected 2 metric names, got %d", len(names))
	}
	for _, name := range names {
		if !expectedNames[name] {
			t.Errorf("unexpected metric name: %s", name)
		}
	}
}

func TestServiceMetricsStore_GetRange(t *testing.T) {
	windowSize := 5 * time.Minute
	period := 15 * time.Second
	store := NewServiceMetricsStore(windowSize, period)
	now := time.Now()

	// Добавляем точки каждые 15 секунд в течение 2 минут
	for i := 0; i < 8; i++ {
		store.Add("test_metric", now.Add(time.Duration(i)*15*time.Second), float64(i*10))
	}

	// Запрашиваем интервал с 30 секунд до 90 секунд
	from := now.Add(30 * time.Second)
	to := now.Add(90 * time.Second)
	rangePoints := store.GetRange("test_metric", from, to)

	if len(rangePoints) != 4 {
		t.Fatalf("expected 4 points in range, got %d", len(rangePoints))
	}
}

func TestServiceMetricsStore_GetLast(t *testing.T) {
	windowSize := 5 * time.Minute
	period := 15 * time.Second
	store := NewServiceMetricsStore(windowSize, period)
	now := time.Now()

	// Добавляем 5 точек
	for i := 0; i < 5; i++ {
		store.Add("test_metric", now.Add(time.Duration(i)*time.Minute), float64(i*100))
	}

	// Получаем последние 3 точки
	last3 := store.GetLast("test_metric", 3)
	if len(last3) != 3 {
		t.Fatalf("expected 3 last points, got %d", len(last3))
	}

	expectedValues := []float64{200.0, 300.0, 400.0}
	for i, v := range expectedValues {
		if last3[i].Value != v {
			t.Errorf("expected last point value %.2f, got %.2f", v, last3[i].Value)
		}
	}
}

func TestServiceMetricsStore_Aggregate(t *testing.T) {
	windowSize := 5 * time.Minute
	period := 15 * time.Second
	store := NewServiceMetricsStore(windowSize, period)
	now := time.Now().Truncate(time.Hour)

	// Добавляем точки каждые 15 секунд
	for i := 0; i < 12; i++ {
		store.Add("test_metric", now.Add(time.Duration(i)*15*time.Second), float64(i*5))
	}

	// Агрегируем по минутам со средним
	minuteAvg := store.Aggregate("test_metric", time.Minute, func(vals []float64) float64 {
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		return sum / float64(len(vals))
	})

	if len(minuteAvg) != 3 {
		t.Fatalf("expected 3 aggregated buckets, got %d", len(minuteAvg))
	}
}

func TestAppMetricsStore_Integration(t *testing.T) {
	windowSize := 5 * time.Minute
	period := 15 * time.Second
	app := NewAppMetricsStore(windowSize, period)
	now := time.Now()

	// Добавляем метрики для разных сервисов
	app.Add("frontend", "rps", now, 100.0)
	app.Add("frontend", "rps", now.Add(15*time.Second), 120.0)
	app.Add("frontend", "latency", now, 50.0)

	app.Add("cart", "rps", now, 200.0)
	app.Add("cart", "cpu", now, 0.5)

	// Проверяем список сервисов
	services := app.ServiceNames()
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}

	// Проверяем метрики для frontend
	frontendMetrics := app.MetricNamesForService("frontend")
	if len(frontendMetrics) != 2 {
		t.Fatalf("expected 2 metrics for frontend, got %d", len(frontendMetrics))
	}

	// Проверяем значения
	rpsValues := app.GetServiceValues("frontend", "rps")
	if len(rpsValues) != 2 {
		t.Fatalf("expected 2 rps values for frontend, got %d", len(rpsValues))
	}

	// Проверяем GetServicePoints
	points := app.GetServicePoints("cart", "rps")
	if len(points) != 1 {
		t.Fatalf("expected 1 point for cart rps, got %d", len(points))
	}

	// Проверяем неизвестный сервис
	if app.GetServiceValues("unknown", "metric") != nil {
		t.Error("expected nil for unknown service")
	}

	// Проверяем неизвестную метрику
	if app.GetServiceValues("frontend", "unknown") != nil {
		t.Error("expected nil for unknown metric")
	}
}

func TestAppMetricsStore_ConcurrentAccess(t *testing.T) {
	windowSize := 5 * time.Minute
	period := 15 * time.Second
	app := NewAppMetricsStore(windowSize, period)
	done := make(chan bool)

	// Пишущие горутины
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				now := time.Now()
				app.Add("service", "metric", now, float64(j))
			}
			done <- true
		}(i)
	}

	// Читающие горутины
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				app.GetServiceValues("service", "metric")
				app.ServiceNames()
				app.MetricNamesForService("service")
			}
			done <- true
		}()
	}

	// Ждём завершения
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestAppMetricsStore_AggregateService(t *testing.T) {
	windowSize := 5 * time.Minute
	period := 15 * time.Second
	app := NewAppMetricsStore(windowSize, period)
	now := time.Now().Truncate(time.Hour)

	// Добавляем точки для сервиса
	for i := 0; i < 12; i++ {
		app.Add("frontend", "rps", now.Add(time.Duration(i)*15*time.Second), float64(i*10))
	}

	// Агрегируем по минутам
	minuteAvg := app.AggregateService("frontend", "rps", time.Minute, func(vals []float64) float64 {
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		return sum / float64(len(vals))
	})

	if len(minuteAvg) != 3 {
		t.Fatalf("expected 3 aggregated buckets, got %d", len(minuteAvg))
	}
}
