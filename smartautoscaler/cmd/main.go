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
		PrometheusURL: "http://prometheus.autoscale-test.svc.cluster.local:9090",
		Interval:      15 * time.Second,
		Timeout:       10 * time.Second,
		Queries: []collector.MetricQuery{
			{
				Name:  "kube_pod_info",
				Query: `kube_pod_info`,
				Help:  "Kubernetes pod metadata",
			},
			{
				Name:  "container_cpu_usage",
				Query: `sum by (namespace,pod,container) (rate(container_cpu_usage_seconds_total[1m]))`,
				Help:  "CPU usage per container (cores)",
			},
			{
				Name:  "container_memory_usage",
				Query: `sum by (namespace,pod,container) (container_memory_usage_bytes)`,
				Help:  "Memory usage per container (bytes)",
			},
			{
				Name:  "container_cpu_quota",
				Query: `sum by (namespace,pod,container) (container_spec_cpu_quota)`,
				Help:  "CPU quota per container",
			},
			{
				Name:  "container_memory_limit",
				Query: `sum by (namespace,pod,container) (container_spec_memory_limit_bytes)`,
				Help:  "Memory limit per container (bytes)",
			},
			{
				Name:  "container_fs_usage",
				Query: `sum by (namespace,pod,container) (container_fs_usage_bytes)`,
				Help:  "Filesystem usage per container (bytes)",
			},
			{
				Name:  "container_fs_write",
				Query: `sum by (namespace,pod,container) (rate(container_fs_write_seconds_total[1m]))`,
				Help:  "Filesystem write time per container",
			},
			{
				Name:  "container_fs_read",
				Query: `sum by (namespace,pod,container) (rate(container_fs_read_seconds_total[1m]))`,
				Help:  "Filesystem read time per container",
			},
			{
				Name:  "container_network_receive",
				Query: `sum by (namespace,pod) (rate(container_network_receive_bytes_total[1m]))`,
				Help:  "Network receive bytes per pod",
			},
			{
				Name:  "container_network_transmit",
				Query: `sum by (namespace,pod) (rate(container_network_transmit_bytes_total[1m]))`,
				Help:  "Network transmit bytes per pod",
			},
			{
				Name: "istio_request_duration_p95",
				Query: `
histogram_quantile(
0.95,
sum by (le, destination_workload) (
	rate(istio_request_duration_milliseconds_bucket{
	reporter="destination",
	destination_workload!=""
	}[1m])
)
)`,
				Help: "Istio P95 request latency per workload",
			},
			{
				Name: "istio_requests",
				Query: `
sum by (destination_workload) (
  rate(istio_requests_total{
    reporter="destination",
    destination_workload!=""
  }[1m])
)`,
				Help: "Istio HTTP requests per workload (RPS)",
			},
			{
				Name: "istio_tcp_received",
				Query: `
sum by (destination_workload) (
  rate(istio_tcp_received_bytes_total{
    reporter="destination",
    destination_workload!=""
  }[1m])
)`,
				Help: "Istio TCP received bytes per workload",
			},
			{
				Name: "istio_tcp_sent",
				Query: `
sum by (destination_workload) (
  rate(istio_tcp_sent_bytes_total{
    reporter="destination",
    destination_workload!=""
  }[1m])
)`,
				Help: "Istio TCP sent bytes per workload",
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
