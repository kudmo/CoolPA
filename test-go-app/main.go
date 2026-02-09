// test-go-app/main.go
package main

import (
	"log"
	"math/rand"
	"net/http"
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
		[]string{"method", "endpoint"},
	)

	// Gauge - пример температуры CPU
	cpuTemp = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_temperature_celsius",
			Help: "Current CPU temperature",
		},
	)

	// Histogram - время обработки запросов
	httpDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)
)

func main() {
	// Инициализируем метрики
	rand.Seed(time.Now().UnixNano())

	// Имитируем изменения температуры CPU
	go func() {
		for {
			temp := 40.0 + rand.Float64()*20.0
			cpuTemp.Set(temp)
			time.Sleep(5 * time.Second)
		}
	}()

	// HTTP-обработчики
	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			duration := time.Since(start).Seconds()
			httpDuration.WithLabelValues("/").Observe(duration)
		}()

		httpRequestsTotal.WithLabelValues(r.Method, "/").Inc()
		w.Write([]byte("Test Go App is running!"))
	})

	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			duration := time.Since(start).Seconds()
			httpDuration.WithLabelValues("/api").Observe(duration)
		}()

		// Имитация работы API
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

		httpRequestsTotal.WithLabelValues(r.Method, "/api").Inc()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	})

	log.Println("Starting test Go app on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
