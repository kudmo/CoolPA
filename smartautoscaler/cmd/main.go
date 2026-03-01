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
	"github.com/kudmo/CoolPA/storage"
)

func main() {
	config := collector.Config{
		PrometheusURL: "http://prometheus.autoscale-test.svc.cluster.local:9090",
		Interval:      15 * time.Second,
		Timeout:       10 * time.Second,
		Queries: []collector.MetricQuery{
			{
				Name:  "kube_pod_info",
				Query: `kube_pod_info * on (namespace, pod) group_left() (kube_pod_labels{label_autoscaling_enabled="true"})`,
				Help:  "Kubernetes pod metadata",
			},
			{
				Name: "container_cpu_usage",
				Query: `sum by (pod) (
					rate(container_cpu_usage_seconds_total{container!="",container!="istio-proxy",container!="POD"}[1m])
					* on (namespace, pod) group_left()
					(kube_pod_labels{label_autoscaling_enabled="true"})
				)`,
				Help: "CPU usage per container (cores)",
			},
			{
				Name: "container_memory_usage",
				Query: `sum by (pod) (
					container_memory_usage_bytes{container!="",container!="istio-proxy",container!="POD"}
					* on (namespace, pod) group_left()
					(kube_pod_labels{label_autoscaling_enabled="true"})
				)`,
				Help: "Memory usage per container (bytes)",
			},
			{
				Name: "container_cpu_quota",
				Query: `sum by (pod) (
					container_spec_cpu_quota{container!="",container!="istio-proxy",container!="POD"}
					* on (namespace, pod) group_left()
					(kube_pod_labels{label_autoscaling_enabled="true"})
				)`,
				Help: "CPU quota per container",
			},
			{
				Name: "container_memory_limit",
				Query: `sum by (pod) (
					container_spec_memory_limit_bytes{container!="",container!="istio-proxy",container!="POD"}
					* on (namespace, pod) group_left()
					(kube_pod_labels{label_autoscaling_enabled="true"})
				)`,
				Help: "Memory limit per container (bytes)",
			},
			{
				Name: "container_fs_usage",
				Query: `sum by (pod) (
					container_fs_usage_bytes{container!="",container!="istio-proxy",container!="POD"}
					* on (namespace, pod) group_left()
					(kube_pod_labels{label_autoscaling_enabled="true"})
				)`,
				Help: "Filesystem usage per container (bytes)",
			},
			{
				Name: "container_fs_write",
				Query: `sum by (pod) (
					rate(container_fs_writes_bytes_total{container!="",container!="istio-proxy",container!="POD"}[1m])
					* on (namespace, pod) group_left()
					(kube_pod_labels{label_autoscaling_enabled="true"})
				)`,
				Help: "Filesystem write bytes per container",
			},
			{
				Name: "container_fs_read",
				Query: `sum by (pod) (
					rate(container_fs_reads_bytes_total{container!="",container!="istio-proxy",container!="POD"}[1m])
					* on (namespace, pod) group_left()
					(kube_pod_labels{label_autoscaling_enabled="true"})
				)`,
				Help: "Filesystem read bytes per container",
			},
			{
				Name: "container_network_receive",
				Query: `sum by (pod) (
					rate(container_network_receive_bytes_total{}[1m])
					* on (namespace, pod) group_left()
					(kube_pod_labels{label_autoscaling_enabled="true"})
				)`,
				Help: "Network receive bytes per pod",
			},
			{
				Name: "container_network_transmit",
				Query: `sum by (pod) (
					rate(container_network_transmit_bytes_total{}[1m])
					* on (namespace, pod) group_left()
					(kube_pod_labels{label_autoscaling_enabled="true"})
				)`,
				Help: "Network transmit bytes per pod",
			},
			{
				Name: "istio_request_duration_p95",
				Query: `histogram_quantile(0.95, 
					sum by (le, destination_app, source_app) (rate(istio_request_duration_milliseconds_bucket{destination_workload!=""}[1m])) 
				) 
				* on (source_app) group_left()
				max by (source_app) (
					label_replace(
					kube_pod_labels{label_autoscaling_enabled="true"},
					"source_app", "$1", "label_app", "(.*)"
					)
				)
				* on (destination_app) group_left()
				max by (destination_app) (
					label_replace(
					kube_pod_labels{label_autoscaling_enabled="true"},
					"destination_app", "$1", "label_app", "(.*)"
					)
				)`,
				Help: "Istio P95 request latency per app (both source and destination must have autoscaling-enabled)",
			},
			{
				Name: "istio_requests_total",
				Query: `sum by (destination_app, source_app) (rate(istio_requests_total{destination_workload!="", reporter="destination"}[1m])) 
				* on (source_app) group_left()
				max by (source_app) (
					label_replace(
					kube_pod_labels{label_autoscaling_enabled="true"},
					"source_app", "$1", "label_app", "(.*)"
					)
				)
				* on (destination_app) group_left()
				max by (destination_app) (
					label_replace(
					kube_pod_labels{label_autoscaling_enabled="true"},
					"destination_app", "$1", "label_app", "(.*)"
					)
				)`,
				Help: "Istio HTTP requests per app (RPS) - both source and destination must have autoscaling-enabled",
			},
			{
				Name: "istio_tcp_received_bytes_total",
				Query: `sum by (source_app, destination_app) (rate(istio_tcp_received_bytes_total{destination_workload!=""}[1m])) 
				* on (source_app) group_left()
				max by (source_app) (
					label_replace(
					kube_pod_labels{label_autoscaling_enabled="true"},
					"source_app", "$1", "label_app", "(.*)"
					)
				)
				* on (destination_app) group_left()
				max by (destination_app) (
					label_replace(
					kube_pod_labels{label_autoscaling_enabled="true"},
					"destination_app", "$1", "label_app", "(.*)"
					)
				)`,
				Help: "Istio TCP received bytes per app - both source and destination must have autoscaling-enabled",
			},
			{
				Name: "istio_tcp_sent_bytes_total",
				Query: `sum by (source_app, destination_app) (rate(istio_tcp_sent_bytes_total{destination_workload!=""}[1m])) 
				* on (source_app) group_left()
				max by (source_app) (
					label_replace(
					kube_pod_labels{label_autoscaling_enabled="true"},
					"source_app", "$1", "label_app", "(.*)"
					)
				)
				* on (destination_app) group_left()
				max by (destination_app) (
					label_replace(
					kube_pod_labels{label_autoscaling_enabled="true"},
					"destination_app", "$1", "label_app", "(.*)"
					)
				)`,
				Help: "Istio TCP sent bytes per app - both source and destination must have autoscaling-enabled",
			},
		},
	}
	handler := storage.NewStorageHandler(config.Interval*10, config.Interval)

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
