package metrics

import (
	"testing"
	"time"
)

func getLogicalBuckets(r *RingWindow) []float64 {
	count := len(r.buckets)
	out := make([]float64, count)
	for i := 0; i < count; i++ {
		idx := (r.head + 1 + i) % count
		out[i] = r.buckets[idx].Value
	}
	return out
}

func TestRingWindowBasicInsert(t *testing.T) {
	step := 10 * time.Second
	window := 30 * time.Second // 3 bucket
	r := NewRingWindow(window, step)

	base := time.Now().Truncate(step)

	// Добавляем 3 значения
	r.Add(base, 1)                     // bucket 0
	r.Add(base.Add(10*time.Second), 2) // bucket 1
	r.Add(base.Add(20*time.Second), 3) // bucket 2

	expected := []float64{1, 2, 3}
	for i := range r.buckets {
		v := r.buckets[i].Value
		if v != expected[i] {
			t.Fatalf("bucket %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestRingWindowWraparound(t *testing.T) {
	step := 10 * time.Second
	window := 30 * time.Second // 3 bucket
	r := NewRingWindow(window, step)

	base := time.Now().Truncate(step)

	// Заполняем первые 3 bucket
	r.Add(base, 1)
	r.Add(base.Add(10*time.Second), 2)
	r.Add(base.Add(20*time.Second), 3)

	// Добавляем 4-й — должен перезаписать первый bucket
	r.Add(base.Add(30*time.Second), 4)

	expected := []float64{4, 2, 3} // head=0 после advance
	for i := range r.buckets {
		v := r.buckets[i].Value
		if v != expected[i] {
			t.Fatalf("wraparound bucket %d: expected %v, got %v", i, expected[i], v)
		}
	}

	// Добавляем ещё один — перезаписывается второй
	r.Add(base.Add(40*time.Second), 5)

	expected = []float64{4, 5, 3}
	for i := range r.buckets {
		v := r.buckets[i].Value
		if v != expected[i] {
			t.Fatalf("second wraparound bucket %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestRingWindowFullCycle(t *testing.T) {
	step := 5 * time.Second
	window := 20 * time.Second // 4 buckets
	r := NewRingWindow(window, step)

	base := time.Now().Truncate(step)

	// Добавляем 6 значений — должны храниться последние 4
	values := []float64{1, 2, 3, 4, 5, 6}
	for i, v := range values {
		r.Add(base.Add(time.Duration(i)*step), v)
	}

	expected := []float64{3, 4, 5, 6} // последние 4 логически
	actual := getLogicalBuckets(r)

	for i, v := range actual {
		if v != expected[i] {
			t.Fatalf("full cycle bucket %d: expected %v, got %v", i, expected[i], v)
		}
	}
}

func TestMetricStoreAddAndRetrievePod(t *testing.T) {
	window := 30 * time.Second
	step := 10 * time.Second
	store := NewMetricStore(window, step)

	serviceName := "svc1"
	podName := "pod1"
	metric := CPUUsage
	ts := time.Now()
	value := 42.0

	err := store.AddSample(serviceName, podName, metric, ts, value)
	if err != nil {
		t.Fatalf("AddSample returned error: %v", err)
	}

	// Проверяем, что сервис создан
	svc, ok := store.services[serviceName]
	if !ok {
		t.Fatal("service not created")
	}

	// Проверяем, что под создан
	pod, ok := svc.pods[podName]
	if !ok {
		t.Fatal("pod not created")
	}

	// Проверяем, что метрика записана в bucket
	bucket := pod.metrics[metric].buckets[pod.metrics[metric].head].Value
	if bucket != value {
		t.Fatalf("expected bucket value %v, got %v", value, bucket)
	}

	// Проверяем lastSeen
	if !pod.lastSeen.Equal(ts) {
		t.Fatalf("expected lastSeen %v, got %v", ts, pod.lastSeen)
	}
}

func TestMetricStoreRemovePod(t *testing.T) {
	window := 30 * time.Second
	step := 10 * time.Second
	store := NewMetricStore(window, step)

	serviceName := "svc1"
	podName1 := "pod1"
	podName2 := "pod2"
	metric := CPUUsage
	ts := time.Now()

	store.AddSample(serviceName, podName1, metric, ts, 10)
	store.AddSample(serviceName, podName2, metric, ts, 20)

	svc := store.services[serviceName]

	// Под1 должен существовать
	if _, ok := svc.pods[podName1]; !ok {
		t.Fatal("pod1 not created")
	}

	store.RemovePod(serviceName, podName1)

	if _, ok := svc.pods[podName1]; ok {
		t.Fatal("pod1 was not removed")
	}

	// Под2 всё ещё существует
	if _, ok := svc.pods[podName2]; !ok {
		t.Fatal("pod2 should still exist")
	}
}

func TestMetricStoreSyncPods(t *testing.T) {
	window := 30 * time.Second
	step := 10 * time.Second
	store := NewMetricStore(window, step)

	serviceName := "svc1"
	pods := []string{"pod1", "pod2", "pod3"}
	ts := time.Now()

	for _, pod := range pods {
		store.AddSample(serviceName, pod, CPUUsage, ts, float64(len(pod)*10))
	}

	svc := store.services[serviceName]

	// Проверяем, что все 3 пода существуют
	if len(svc.pods) != 3 {
		t.Fatalf("expected 3 pods, got %d", len(svc.pods))
	}

	// Синхронизируем, оставляем только pod2
	store.SyncPods(serviceName, []string{"pod2"})

	if len(svc.pods) != 1 {
		t.Fatalf("expected 1 pod after sync, got %d", len(svc.pods))
	}

	if _, ok := svc.pods["pod2"]; !ok {
		t.Fatal("pod2 should exist after sync")
	}
}

func TestMetricStoreMultipleMetrics(t *testing.T) {
	window := 30 * time.Second
	step := 10 * time.Second
	store := NewMetricStore(window, step)

	serviceName := "svc1"
	podName := "pod1"
	ts := time.Now()

	values := map[MetricID]float64{
		CPUUsage:        10,
		MemoryUsage:     50,
		NetworkReceive:  100,
		NetworkTransmit: 200,
	}

	for m, v := range values {
		err := store.AddSample(serviceName, podName, m, ts, v)
		if err != nil {
			t.Fatalf("AddSample error: %v", err)
		}
	}

	pod := store.services[serviceName].pods[podName]

	for m, v := range values {
		bucket := pod.metrics[m].buckets[pod.metrics[m].head].Value
		if bucket != v {
			t.Fatalf("metric %v: expected %v, got %v", m, v, bucket)
		}
	}
}

func TestMetricStoreAddSampleInvalidMetric(t *testing.T) {
	window := 30 * time.Second
	step := 10 * time.Second
	store := NewMetricStore(window, step)

	err := store.AddSample("svc1", "pod1", MetricID(100), time.Now(), 1)
	if err == nil {
		t.Fatal("expected error for invalid metric, got nil")
	}
	if err != ErrInvalidMetric {
		t.Fatalf("expected ErrInvalidMetric, got %v", err)
	}
}

func TestRingWindow_SumRange(t *testing.T) {
	// Создаём окно с 4 корзинами по 10 секунд (всего 40 секунд)
	w := NewRingWindow(40*time.Second, 10*time.Second)

	// Начальное состояние: startTs = 0
	if got := w.SumRange(time.Unix(100, 0), time.Unix(200, 0)); got != 0 {
		t.Errorf("SumRange on empty window = %v, want 0", got)
	}

	// Добавляем данные в корзины с известными значениями
	// Время: 100, 110, 120, 130 секунд (начала корзин)
	// Корзина 0 (время 100) -> 10
	// Корзина 1 (время 110) -> 20
	// Корзина 2 (время 120) -> 30
	// Корзина 3 (время 130) -> 40
	w.Add(time.Unix(100, 0), 10) // создаст start = 100
	w.Add(time.Unix(110, 0), 20)
	w.Add(time.Unix(120, 0), 30)
	w.Add(time.Unix(130, 0), 40)

	// Полная сумма всех корзин
	if got := w.SumRange(time.Unix(100, 0), time.Unix(140, 0)); got != 100 {
		t.Errorf("SumRange(100,140) = %v, want 100", got)
	}

	// Частичный диапазон, включающий две корзины
	if got := w.SumRange(time.Unix(110, 0), time.Unix(130, 0)); got != 20+30 { // корзины 110 и 120
		t.Errorf("SumRange(110,130) = %v, want 50", got)
	}

	// Диапазон, начинающийся между корзинами
	// 105-115: должна попасть корзина 110 (20), корзина 100 (10) не попадает, т.к. её начало 100 < 105
	if got := w.SumRange(time.Unix(105, 0), time.Unix(115, 0)); got != 20 {
		t.Errorf("SumRange(105,115) = %v, want 20", got)
	}

	// Диапазон, полностью лежащий внутри одной корзины (начало корзины вне интервала)
	if got := w.SumRange(time.Unix(111, 0), time.Unix(119, 0)); got != 0 {
		t.Errorf("SumRange(111,119) = %v, want 0", got)
	}

	// Диапазон, не пересекающий окно (позже)
	if got := w.SumRange(time.Unix(200, 0), time.Unix(210, 0)); got != 0 {
		t.Errorf("SumRange(200,210) = %v, want 0", got)
	}

	// Диапазон, не пересекающий окно (раньше)
	if got := w.SumRange(time.Unix(0, 0), time.Unix(10, 0)); got != 0 {
		t.Errorf("SumRange(0,10) = %v, want 0", got)
	}

	// from >= to
	if got := w.SumRange(time.Unix(150, 0), time.Unix(140, 0)); got != 0 {
		t.Errorf("SumRange(150,140) = %v, want 0", got)
	}
}

func TestRingWindow_AvgRange(t *testing.T) {
	w := NewRingWindow(40*time.Second, 10*time.Second)

	// Пустое окно
	if got := w.AvgRange(time.Unix(100, 0), time.Unix(200, 0)); got != 0 {
		t.Errorf("AvgRange on empty window = %v, want 0", got)
	}

	w.Add(time.Unix(100, 0), 10)
	w.Add(time.Unix(110, 0), 20)
	w.Add(time.Unix(120, 0), 30)
	w.Add(time.Unix(130, 0), 40)

	// Полный диапазон
	if got := w.AvgRange(time.Unix(100, 0), time.Unix(140, 0)); got != (10+20+30+40)/4 {
		t.Errorf("AvgRange(100,140) = %v, want %v", got, (10+20+30+40)/4)
	}

	// Две корзины
	if got := w.AvgRange(time.Unix(110, 0), time.Unix(130, 0)); got != (20+30)/2 {
		t.Errorf("AvgRange(110,130) = %v, want %v", got, (20+30)/2)
	}

	// Одна корзина
	if got := w.AvgRange(time.Unix(120, 0), time.Unix(130, 0)); got != 30 {
		t.Errorf("AvgRange(120,130) = %v, want 30", got)
	}

	// Нет корзин
	if got := w.AvgRange(time.Unix(200, 0), time.Unix(210, 0)); got != 0 {
		t.Errorf("AvgRange(200,210) = %v, want 0", got)
	}

	// from >= to
	if got := w.AvgRange(time.Unix(150, 0), time.Unix(140, 0)); got != 0 {
		t.Errorf("AvgRange(150,140) = %v, want 0", got)
	}
}

func TestRingWindow_MaxRange(t *testing.T) {
	w := NewRingWindow(40*time.Second, 10*time.Second)

	// Пустое окно
	if got := w.MaxRange(time.Unix(100, 0), time.Unix(200, 0)); got != 0 {
		t.Errorf("MaxRange on empty window = %v, want 0", got)
	}

	w.Add(time.Unix(100, 0), 10)
	w.Add(time.Unix(110, 0), 20)
	w.Add(time.Unix(120, 0), 30)
	w.Add(time.Unix(130, 0), 40)

	// Полный диапазон
	if got := w.MaxRange(time.Unix(100, 0), time.Unix(140, 0)); got != 40 {
		t.Errorf("MaxRange(100,140) = %v, want 40", got)
	}

	// Две корзины
	if got := w.MaxRange(time.Unix(110, 0), time.Unix(130, 0)); got != 30 {
		t.Errorf("MaxRange(110,130) = %v, want 30", got)
	}

	// Одна корзина
	if got := w.MaxRange(time.Unix(120, 0), time.Unix(130, 0)); got != 30 {
		t.Errorf("MaxRange(120,130) = %v, want 30", got)
	}

	// Нет корзин
	if got := w.MaxRange(time.Unix(200, 0), time.Unix(210, 0)); got != 0 {
		t.Errorf("MaxRange(200,210) = %v, want 0", got)
	}

	// from >= to
	if got := w.MaxRange(time.Unix(150, 0), time.Unix(140, 0)); got != 0 {
		t.Errorf("MaxRange(150,140) = %v, want 0", got)
	}
}

func TestRingWindow_RangeAfterAdvance(t *testing.T) {
	// Проверяем, что после продвижения окна старые данные не учитываются
	w := NewRingWindow(30*time.Second, 10*time.Second) // 3 корзины

	// Добавляем в корзины с временем 100, 110, 120
	w.Add(time.Unix(100, 0), 10)
	w.Add(time.Unix(110, 0), 20)
	w.Add(time.Unix(120, 0), 30)

	// Сумма всего окна сейчас: 60
	if got := w.SumRange(time.Unix(100, 0), time.Unix(130, 0)); got != 60 {
		t.Errorf("before advance: SumRange(100,130)=%v, want 60", got)
	}

	// Добавляем новое значение с временем 200, что сдвинет окно вперёд и обнулит все корзины,
	// так как шаг > размера окна (70 секунд разницы > 30 секунд)
	w.Add(time.Unix(200, 0), 100)

	// Теперь окно должно содержать только корзину с временем 200 (значение 100),
	// остальные обнулены. startTs должен стать 200 (200-200%10=200)
	// Проверим сумму за период 100-130: старых данных нет
	if got := w.SumRange(time.Unix(100, 0), time.Unix(130, 0)); got != 0 {
		t.Errorf("after advance: SumRange(100,130)=%v, want 0", got)
	}

	// Сумма за период, включающий новую корзину
	if got := w.SumRange(time.Unix(200, 0), time.Unix(210, 0)); got != 100 {
		t.Errorf("after advance: SumRange(200,210)=%v, want 100", got)
	}

	// Среднее за новое окно
	if got := w.AvgRange(time.Unix(200, 0), time.Unix(210, 0)); got != 100 {
		t.Errorf("after advance: AvgRange(200,210)=%v, want 100", got)
	}
}

func TestRingWindow_EdgeCases(t *testing.T) {
	// Проверяем граничные случаи с нулевыми значениями и частичным перекрытием
	w := NewRingWindow(20*time.Second, 10*time.Second) // 2 корзины

	// Заполняем корзины со временем 100 и 110
	w.Add(time.Unix(100, 0), 5)  // корзина 100
	w.Add(time.Unix(110, 0), 15) // корзина 110

	// Диапазон [100,110): должна войти только корзина 100 (5)
	if got := w.SumRange(time.Unix(100, 0), time.Unix(110, 0)); got != 5 {
		t.Errorf("SumRange(100,110) = %v, want 5", got)
	}

	// Диапазон [100,100) пустой
	if got := w.SumRange(time.Unix(100, 0), time.Unix(100, 0)); got != 0 {
		t.Errorf("SumRange(100,100) = %v, want 0", got)
	}

	// Диапазон [110,120): корзина 110 (15)
	if got := w.SumRange(time.Unix(110, 0), time.Unix(120, 0)); got != 15 {
		t.Errorf("SumRange(110,120) = %v, want 15", got)
	}

	// Диапазон [105,115): попадают обе корзины? корзина 100 начинается в 100, заканчивается в 110, её начало 100 < 105? Нет, она не попадает, т.к. её начало < from. Корзина 110 начинается в 110, что >=105 и <115 => попадает только 110. Итог 15.
	if got := w.SumRange(time.Unix(105, 0), time.Unix(115, 0)); got != 15 {
		t.Errorf("SumRange(105,115) = %v, want 15", got)
	}

	// Диапазон [90, 120): обе корзины попадают (100 и 110)
	if got := w.SumRange(time.Unix(90, 0), time.Unix(120, 0)); got != 20 {
		t.Errorf("SumRange(90,120) = %v, want 20", got)
	}
}

func TestGetServicesAndPodsAndLastSeen(t *testing.T) {
	w := 30 * time.Second
	s := NewMetricStore(w, 10*time.Second)

	svcA := "svcA"
	svcB := "svcB"
	pod1 := "p1"
	pod2 := "p2"

	ts1 := time.Now()
	ts2 := ts1.Add(1 * time.Minute)

	if err := s.AddSample(svcA, pod1, CPUUsage, ts1, 1); err != nil {
		t.Fatalf("AddSample failed: %v", err)
	}
	if err := s.AddSample(svcA, pod2, MemoryUsage, ts2, 2); err != nil {
		t.Fatalf("AddSample failed: %v", err)
	}
	if err := s.AddSample(svcB, pod1, CPUUsage, ts1, 3); err != nil {
		t.Fatalf("AddSample failed: %v", err)
	}

	services := s.GetServices()
	foundA := false
	foundB := false
	for _, v := range services {
		if v == svcA {
			foundA = true
		}
		if v == svcB {
			foundB = true
		}
	}
	if !foundA || !foundB {
		t.Fatalf("GetServices missing entries: got=%v", services)
	}

	pods := s.GetServicePods(svcA)
	if pods == nil || len(pods) != 2 {
		t.Fatalf("GetServicePods(svcA) expected 2 pods, got %v", pods)
	}

	// last seen checks
	if ls, ok := s.GetPodLastSeen(svcA, pod1); !ok || !ls.Equal(ts1) {
		t.Fatalf("GetPodLastSeen mismatch for %s/%s: got=%v ok=%v want=%v", svcA, pod1, ls, ok, ts1)
	}
	if ls, ok := s.GetPodLastSeen(svcA, pod2); !ok || !ls.Equal(ts2) {
		t.Fatalf("GetPodLastSeen mismatch for %s/%s: got=%v ok=%v want=%v", svcA, pod2, ls, ok, ts2)
	}
}

func TestGetPodMetricHeadValueInvalidMetric(t *testing.T) {
	s := NewMetricStore(30*time.Second, 10*time.Second)
	_, ok, err := s.GetPodMetricHeadValue("no", "no", MetricID(255))
	if err == nil {
		t.Fatal("expected error for invalid metric id")
	}
	if err != ErrInvalidMetric {
		t.Fatalf("expected ErrInvalidMetric, got %v", err)
	}
	if ok {
		t.Fatal("ok should be false for missing service/pod")
	}
}

func TestGetPodMetricHeadValueNotFound(t *testing.T) {
	s := NewMetricStore(30*time.Second, 10*time.Second)
	v, ok, err := s.GetPodMetricHeadValue("svcX", "podX", CPUUsage)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false for non-existing pod, got value=%v", v)
	}
}
