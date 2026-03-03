// test-go-app/main.go
package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Counter - счетчик HTTP запросов
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// Gauge - пример температуры CPU и использования памяти
	cpuTemp = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_temperature_celsius",
			Help: "Current CPU temperature",
		},
	)

	memoryUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Current memory usage",
		},
	)

	// Histogram - время обработки запросов
	httpDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"endpoint"},
	)

	// Счетчик активных горутин
	activeGoroutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_goroutines_total",
			Help: "Number of active goroutines",
		},
	)
)

// DataCache - структура для имитации кэша с мьютексами
type DataCache struct {
	mu    sync.RWMutex
	data  map[string]*CacheItem
	items int
}

type CacheItem struct {
	Value       string
	ExpiresAt   time.Time
	AccessCount int
}

var (
	cache = &DataCache{
		data:  make(map[string]*CacheItem),
		items: 0,
	}

	// Глобальная блокировка для имитации проблем
	globalMu sync.Mutex

	// Очередь задач для имитации backlog
	taskQueue = make(chan func(), 1000)

	// Пул соединений с блокировками
	connectionPool = make([]*Connection, 10)
	poolMu         sync.Mutex

	// Счетчик активных запросов
	activeRequests int32
	requestMu      sync.Mutex
)

type Connection struct {
	ID       int
	InUse    bool
	LastUsed time.Time
	mu       sync.Mutex
}

func init() {
	// Инициализация пула соединений
	for i := 0; i < 10; i++ {
		connectionPool[i] = &Connection{
			ID:       i,
			InUse:    false,
			LastUsed: time.Now(),
		}
	}

	// Запуск воркеров для обработки очереди
	for i := 0; i < 5; i++ {
		go taskWorker(i)
	}
}

func taskWorker(id int) {
	for task := range taskQueue {
		time.Sleep(10 * time.Millisecond) // Имитация обработки
		task()
	}
}

func main() {
	// Инициализируем метрики
	rand.Seed(time.Now().UnixNano())

	// Имитируем изменения температуры CPU и памяти
	go func() {
		for {
			temp := 40.0 + rand.Float64()*20.0
			cpuTemp.Set(temp)

			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			memoryUsage.Set(float64(m.Alloc))

			activeGoroutines.Set(float64(runtime.NumGoroutine()))

			time.Sleep(5 * time.Second)
		}
	}()

	// Периодическая очистка кэша с блокировкой
	go func() {
		for {
			time.Sleep(30 * time.Second)
			cleanCache()
		}
	}()

	// HTTP-обработчики
	http.Handle("/metrics", promhttp.Handler())

	// Простой эндпоинт с минимальной нагрузкой
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			duration := time.Since(start).Seconds()
			httpDuration.WithLabelValues("/").Observe(duration)
		}()

		httpRequestsTotal.WithLabelValues(r.Method, "/", "200").Inc()
		w.Write([]byte("Test Go App is running!"))
	})

	// API эндпоинт с тяжелой логикой
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		requestMu.Lock()
		activeRequests++
		requestCount := activeRequests
		requestMu.Unlock()

		defer func() {
			duration := time.Since(start).Seconds()
			httpDuration.WithLabelValues("/api").Observe(duration)

			requestMu.Lock()
			activeRequests--
			requestMu.Unlock()
		}()

		if rand.Float32() < 0.3 {
			globalMu.Lock()
			time.Sleep(50 * time.Millisecond)
			globalMu.Unlock()
		}

		conn := getConnection()
		if conn != nil {
			defer releaseConnection(conn)

			conn.mu.Lock()
			time.Sleep(10 * time.Millisecond)
			conn.LastUsed = time.Now()
			conn.mu.Unlock()
		}

		cacheKey := r.URL.Query().Get("key")
		if cacheKey != "" {
			if item := cacheGet(cacheKey); item != nil {
				time.Sleep(5 * time.Millisecond)
				jsonResponse(w, map[string]interface{}{
					"status": "cached",
					"data":   item.Value,
				})
				httpRequestsTotal.WithLabelValues(r.Method, "/api", "200").Inc()
				return
			}

			result := heavyComputation(cacheKey)
			cacheSet(cacheKey, result, 60*time.Second)
		}

		processCount := 10000
		if requestCount > 30 {
			processCount = 50000
		}

		var syncMap sync.Map
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < processCount/10; j++ {
					syncMap.Store(id*10000+j, j*j)

					_ = float64(j) * rand.Float64()

					if j%1000 == 0 {
						globalMu.Lock()
						time.Sleep(time.Microsecond)
						globalMu.Unlock()
					}
				}
			}(i)
		}
		wg.Wait()

		taskQueue <- func() {
			time.Sleep(50 * time.Millisecond)
		}

		httpRequestsTotal.WithLabelValues(r.Method, "/api", "200").Inc()
		jsonResponse(w, map[string]interface{}{
			"status":          "ok",
			"active_requests": requestCount,
		})
	})

	log.Println("Starting test Go app on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func cacheGet(key string) *CacheItem {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	item, exists := cache.data[key]
	if !exists {
		return nil
	}

	if time.Now().After(item.ExpiresAt) {
		return nil
	}

	item.AccessCount++
	return item
}

func cacheSet(key, value string, ttl time.Duration) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	cache.data[key] = &CacheItem{
		Value:       value,
		ExpiresAt:   time.Now().Add(ttl),
		AccessCount: 0,
	}
	cache.items++
}

func cleanCache() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	now := time.Now()
	for key, item := range cache.data {
		if now.After(item.ExpiresAt) {
			delete(cache.data, key)
			cache.items--
		}
	}
}

func getConnection() *Connection {
	poolMu.Lock()
	defer poolMu.Unlock()

	for _, conn := range connectionPool {
		conn.mu.Lock()
		if !conn.InUse {
			conn.InUse = true
			conn.mu.Unlock()
			return conn
		}
		conn.mu.Unlock()
	}

	time.Sleep(50 * time.Millisecond)
	return &Connection{
		ID:       len(connectionPool),
		InUse:    true,
		LastUsed: time.Now(),
	}
}

func releaseConnection(conn *Connection) {
	poolMu.Lock()
	defer poolMu.Unlock()

	conn.mu.Lock()
	conn.InUse = false
	conn.LastUsed = time.Now()
	conn.mu.Unlock()
}

func heavyComputation(input string) string {
	result := input
	for i := 0; i < 1000; i++ {
		hash := 0
		for _, ch := range result {
			hash = (hash + int(ch)) * 31
			if i%100 == 0 {
				time.Sleep(time.Microsecond)
			}
		}
		result = string(rune(hash))
	}
	return result
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
