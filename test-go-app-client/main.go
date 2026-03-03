// test-go-app-client/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Метрики для отслеживания отправленных запросов
	requestsSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "client_requests_sent_total",
			Help: "Total number of requests sent to target service",
		},
		[]string{"target_service", "endpoint", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "client_request_duration_seconds",
			Help:    "Duration of requests to target service",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"target_service"},
	)

	activeRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "client_active_requests",
			Help: "Number of active requests being sent",
		},
	)

	// Информация о целевом сервисе
	targetServiceInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "client_target_service_info",
			Help: "Information about target service",
		},
		[]string{"target_service", "target_port"},
	)
)

type ClientConfig struct {
	TargetService     string
	TargetPort        string
	Interval          time.Duration
	RequestsPerMinute int
	Endpoints         []string
}

func main() {
	// Конфигурация - можно вынести в переменные окружения
	config := ClientConfig{
		TargetService:     "test-go-app",
		TargetPort:        "8080",
		Interval:          5 * time.Second,
		RequestsPerMinute: 60,
		Endpoints:         []string{"/", "/api"},
	}

	// Устанавливаем информацию о целевом сервисе
	targetServiceInfo.WithLabelValues(
		config.TargetService,
		config.TargetPort,
	).Set(1)

	// Запускаем HTTP сервер для метрик
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK")
		})

		log.Println("Starting metrics server on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	// Запускаем отправку запросов
	startTrafficGenerator(config)

	// Блокируем основной поток
	select {}
}

func startTrafficGenerator(config ClientConfig) {
	ticker := time.NewTicker(config.Interval)

	// Создаем HTTP клиент с таймаутами
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	baseURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%s",
		config.TargetService,
		"autoscale-test",
		config.TargetPort,
	)

	log.Printf("Starting traffic generator to %s\n", baseURL)
	log.Printf("Endpoints: %v\n", config.Endpoints)

	go func() {
		for range ticker.C {
			for i := 0; i < config.RequestsPerMinute/12; i++ {
				go sendRequest(client, baseURL, config.Endpoints)
			}
		}
	}()
}

func sendRequest(client *http.Client, baseURL string, endpoints []string) {
	activeRequests.Inc()
	defer activeRequests.Dec()

	endpoint := endpoints[rand.Intn(len(endpoints))]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+endpoint, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		requestsSent.WithLabelValues(baseURL, endpoint, "error").Inc()
		return
	}

	req.Header.Set("User-Agent", "Test-Client/1.0")
	req.Header.Set("X-Request-ID", fmt.Sprintf("%d", time.Now().UnixNano()))

	startTime := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(startTime).Seconds()

	if err != nil {
		log.Printf("Error sending request to %s: %v", endpoint, err)
		requestsSent.WithLabelValues(baseURL, endpoint, "error").Inc()
		return
	}
	defer resp.Body.Close()

	status := fmt.Sprintf("%d", resp.StatusCode)
	requestsSent.WithLabelValues(baseURL, endpoint, status).Inc()
	requestDuration.WithLabelValues(baseURL).Observe(duration)

	log.Printf("Request to %s completed with status %d in %.3fs",
		endpoint, resp.StatusCode, duration)
}
