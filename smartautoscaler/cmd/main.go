// cmd/smartautoscaler/main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kudmo/CoolPA/collector"
)

func main() {
	config := collector.Config{
		PrometheusURL: "http://prometheus:9090",
		Interval:      15 * time.Second,
		Timeout:       10 * time.Second,
		Queries: []collector.MetricQuery{
			{
				Name:  "up",
				Query: `up{job="test-go-app"}`,
				Help:  "Test service status",
			},
			{
				Name:  "http_request_duration_seconds",
				Query: `http_request_duration_seconds`,
				Help:  "HTTP requests duration seconds",
			},
			{
				Name:  "cpu_temperature_celsius",
				Query: `cpu_temperature_celsius`,
				Help:  "CPU temperature celsius",
			},
		},
	}

	handler := collector.NewLogHandler(log.Default())

	promCollector, err := collector.NewPrometheusCollector(
		config,
		collector.WithHandler(handler),
	)
	if err != nil {
		log.Fatalf("Failed to create collector: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := promCollector.Start(ctx); err != nil {
		log.Fatalf("Failed to start collector: %v", err)
	}

	log.Println("Metric collector started. Press Ctrl+C to stop.")

	<-sigChan
	log.Println("Shutdown signal received")

	if err := promCollector.Stop(); err != nil {
		log.Printf("Error stopping collector: %v", err)
	}

	log.Println("Collector stopped")
}
