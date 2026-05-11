// cmd/smartautoscaler/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kudmo/CoolPA/logger"

	"github.com/kudmo/CoolPA/collector"
	"github.com/kudmo/CoolPA/config"
	"github.com/kudmo/CoolPA/decision"
	reactionapplier "github.com/kudmo/CoolPA/reaction_applier"
	"github.com/kudmo/CoolPA/storage"
)

// Logging is configured via the shared logger package.

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "/etc/config/config.yaml", "Path to configuration file")
	flag.StringVar(&configPath, "c", "/etc/config/config.yaml", "Path to configuration file (shorthand)")

	flag.Parse()

	cfg := &config.ScalerConfig{}
	if err := cfg.LoadFromYAML(configPath); err != nil {
		logger.Error("main", "failed to load config", "error", err, "path", configPath)
		os.Exit(1)
	}

	logger.Init(cfg.Logger)

	// Configs

	collectorConfig := collector.PrometheusCollectorConfig{
		PrometheusURL: cfg.PrometheusURL,
		Interval:      cfg.PrometheusInterval,
		Timeout:       4 * time.Second,
		Queries: []collector.MetricQuery{
			{
				Name:  "kube_pod_info",
				Query: fmt.Sprintf(`kube_pod_info{namespace="%s"}`, cfg.ScalingNamespace),
				Help:  "Kubernetes pod metadata",
			},
			{
				Name:  "kube_resourcequota",
				Query: fmt.Sprintf(`kube_resourcequota{namespace="%s",resource=~"limits.*|pods", type="hard"}`, cfg.ScalingNamespace),
				Help:  "Kubernetes namespace limits",
			},
			{
				Name:  "kube_limitrange",
				Query: fmt.Sprintf(`kube_limitrange{namespace="%s", constraint=~"min|max"}`, cfg.ScalingNamespace),
				Help:  "Kubernetes service limits",
			},
			{
				Name:  "container_cpu_usage",
				Query: fmt.Sprintf(`sum by (pod) (rate(container_cpu_usage_seconds_total{container!="",container!="istio-proxy",container!="POD",namespace="%s"}[1m]))`, cfg.ScalingNamespace),
				Help:  "CPU usage per container (cores)",
			},
			{
				Name:  "container_memory_usage",
				Query: fmt.Sprintf(`sum by (pod) (container_memory_usage_bytes{container!="",container!="istio-proxy",container!="POD",namespace="%s"}) / (1024*1024)`, cfg.ScalingNamespace),
				Help:  "Memory usage per container (Mbytes)",
			},
			{
				Name:  "container_cpu_quota",
				Query: fmt.Sprintf(`sum by (pod) (container_spec_cpu_quota{container!="",container!="istio-proxy",container!="POD",namespace="%s"}) / 100`, cfg.ScalingNamespace),
				Help:  "CPU quota per container",
			},
			{
				Name:  "container_memory_limit",
				Query: fmt.Sprintf(`sum by (pod) (container_spec_memory_limit_bytes{container!="",container!="istio-proxy",container!="POD",namespace="%s"})  / (1024*1024)`, cfg.ScalingNamespace),
				Help:  "Memory limit per container (Mbytes)",
			},
			{
				Name:  "pod_cpu_quota",
				Query: fmt.Sprintf(`sum by (pod) (container_spec_cpu_quota{namespace="%s"}) / 100`, cfg.ScalingNamespace),
				Help:  "CPU quota per pod",
			},
			{
				Name:  "pod_memory_limit",
				Query: fmt.Sprintf(`sum by (pod) (container_spec_memory_limit_bytes{namespace="%s"})  / (1024*1024)`, cfg.ScalingNamespace),
				Help:  "Memory limit per pod",
			},
			{
				Name:  "container_fs_usage",
				Query: fmt.Sprintf(`sum by (pod) (container_fs_usage_bytes{container!="",container!="istio-proxy",container!="POD",namespace="%s"})`, cfg.ScalingNamespace),
				Help:  "Filesystem usage per container (bytes)",
			},
			{
				Name:  "container_fs_write",
				Query: fmt.Sprintf(`sum by (pod) (rate(container_fs_writes_bytes_total{container!="",container!="istio-proxy",container!="POD",namespace="%s"}[1m]))`, cfg.ScalingNamespace),
				Help:  "Filesystem write bytes per container",
			},
			{
				Name:  "container_fs_read",
				Query: fmt.Sprintf(`sum by (pod) (rate(container_fs_reads_bytes_total{container!="",container!="istio-proxy",container!="POD",namespace="%s"}[1m]))`, cfg.ScalingNamespace),
				Help:  "Filesystem read bytes per container",
			},
			{
				Name:  "container_network_receive",
				Query: fmt.Sprintf(`sum by (pod) (rate(container_network_receive_bytes_total{namespace="%s"}[1m]))`, cfg.ScalingNamespace),
				Help:  "Network receive bytes per pod",
			},
			{
				Name:  "container_network_transmit",
				Query: fmt.Sprintf(`sum by (pod) (rate(container_network_transmit_bytes_total{namespace="%s"}[1m]))`, cfg.ScalingNamespace),
				Help:  "Network transmit bytes per pod",
			},
			{
				Name: "istio_request_duration_p95",
				Query: fmt.Sprintf(`histogram_quantile(0.95, 
					sum by (le, destination_workload, source_workload) (rate(istio_request_duration_milliseconds_bucket{destination_workload!="",destination_workload_namespace="%s"}[1m])) 
				)`, cfg.ScalingNamespace),
				Help: "Istio P95 request latency per app (both source and destination must have autoscaling-enabled)",
			},
			{
				Name: "istio_request_duration_p50",
				Query: fmt.Sprintf(`histogram_quantile(0.50, 
					sum by (le, destination_workload, source_workload) (rate(istio_request_duration_milliseconds_bucket{destination_workload!="",destination_workload_namespace="%s"}[1m])) 
				)`, cfg.ScalingNamespace),
				Help: "Istio P95 request latency per app (both source and destination must have autoscaling-enabled)",
			},
			{
				Name:  "istio_requests_total",
				Query: fmt.Sprintf(`sum by (destination_workload) (rate(istio_requests_total{destination_workload!="", reporter="destination",destination_workload_namespace="%s"}[1m]))`, cfg.ScalingNamespace),
				Help:  "Istio HTTP requests per app (RPS) - both source and destination must have autoscaling-enabled",
			},
			{
				Name:  "istio_tcp_received_bytes_total",
				Query: fmt.Sprintf(`sum by (destination_workload) (rate(istio_tcp_received_bytes_total{destination_workload!="",destination_workload_namespace="%s"}[1m]))`, cfg.ScalingNamespace),
				Help:  "Istio TCP received bytes per app - both source and destination must have autoscaling-enabled",
			},
			{
				Name:  "istio_tcp_sent_bytes_total",
				Query: fmt.Sprintf(`sum by (destination_workload) (rate(istio_tcp_sent_bytes_total{destination_workload!="",destination_workload_namespace="%s"}[1m]))`, cfg.ScalingNamespace),
				Help:  "Istio TCP sent bytes per app - both source and destination must have autoscaling-enabled",
			},
		},
	}
	analyzerConfig := decision.DecisionMakerConfig{
		Interval: cfg.AnalyzerInterval,
		Cooldown: cfg.ScalingCooldown,
		SLO:      float64(cfg.SLO),
		Lambda:   cfg.Lambda,
	}

	// Components creating

	store := storage.NewStorage(10*time.Minute, collectorConfig.Interval, cfg)
	handler := storage.NewStorageHandler(store)

	promCollector, err := collector.NewPrometheusCollector(
		collectorConfig,
		collector.WithHandler(handler),
	)
	if err != nil {
		logger.Error("main", "failed to create collector", "error", err)
	}

	applier, err := reactionapplier.BuildApplier()
	if err != nil {
		logger.Error("main", "failed to create applier", "error", err)
	}

	analyzer := decision.NewDecisionMaker(
		analyzerConfig,
		store,
		applier,
	)

	// Context creating and starting components

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := promCollector.Start(ctx); err != nil {
		logger.Error("main", "failed to start collector", "error", err)
	}

	logger.Info("main", "metric collector started")

	if err := analyzer.Start(ctx); err != nil {
		logger.Error("main", "failed to start analyzer", "error", err)
	}
	logger.Info("main", "analyzer started")

	// Components destroying

	<-sigChan
	logger.Info("main", "shutdown signal received")

	if err := analyzer.Stop(); err != nil {
		logger.Error("main", "error stopping analyzer", "error", err)
	}
	logger.Info("main", "analyzer stopped")

	if err := promCollector.Stop(); err != nil {
		logger.Error("main", "error stopping collector", "error", err)
	}

	logger.Info("main", "collector stopped")
}
