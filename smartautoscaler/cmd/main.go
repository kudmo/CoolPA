// cmd/smartautoscaler/main.go
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kudmo/CoolPA/analyzer"
	"github.com/kudmo/CoolPA/collector"
	"github.com/kudmo/CoolPA/config"
	"github.com/kudmo/CoolPA/storage"
)

func setupLogging(loggerConfig config.LoggerConfig) {
	var logLevel slog.Level
	switch loggerConfig.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handlerOpts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(time.Now().Format(time.RFC3339))
			}
			return a
		},
	}

	var handler slog.Handler
	switch loggerConfig.Format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	case "text":
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	default:
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "/etc/config/config.yaml", "Path to configuration file")
	flag.StringVar(&configPath, "c", "/etc/config/config.yaml", "Path to configuration file (shorthand)")

	flag.Parse()

	cfg := &config.ScalerConfig{}
	if err := cfg.LoadFromYAML(configPath); err != nil {
		panic(err)
	}

	setupLogging(cfg.Logger)

	// Configs

	collectorConfig := collector.PrometheusCollectorConfig{
		PrometheusURL: cfg.PrometheusURL,
		Interval:      cfg.PrometheusInterval,
		Timeout:       4 * time.Second,
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
				Name: "istio_request_duration_p50",
				Query: `histogram_quantile(0.50, 
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
				Query: `sum by (destination_app) (rate(istio_requests_total{destination_workload!="", reporter="destination"}[1m])) 
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
				Query: `sum by (destination_app) (rate(istio_tcp_received_bytes_total{destination_workload!=""}[1m])) 
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
				Query: `sum by (destination_app) (rate(istio_tcp_sent_bytes_total{destination_workload!=""}[1m])) 
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
	analyzerConfig := analyzer.AnalyzerConfig{
		Interval:              cfg.AnalyzerInterval,
		Confidence:            0.05,
		WelchOldIntervalBegin: time.Duration(300 * time.Second),
		WelchNowIntervalBegin: time.Duration(60 * time.Second),
	}

	// Components creating

	store := storage.NewStorage(10*time.Minute, collectorConfig.Interval)
	handler := storage.NewStorageHandler(store)

	promCollector, err := collector.NewPrometheusCollector(
		collectorConfig,
		collector.WithHandler(handler),
	)
	if err != nil {
		slog.Error("Failed to create collector", "error", err)
	}
	analyzer := analyzer.NewAnalyzer(
		analyzerConfig,
		store,
	)

	// Context creating and starting components

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := promCollector.Start(ctx); err != nil {
		slog.Error("Failed to start collector", "error", err)
	}

	slog.Info("Metric collector started. Press Ctrl+C to stop.")

	if err := analyzer.Start(ctx); err != nil {
		slog.Error("Failed to start analyzer", "error", err)
	}
	slog.Info("Analyzer started. Press Ctrl+C to stop.")

	// Components destroying

	<-sigChan
	slog.Info("Shutdown signal received")

	if err := analyzer.Stop(); err != nil {
		slog.Error("Error stopping analyzer", "error", err)
	}
	slog.Info("Analyzer stopped")

	if err := promCollector.Stop(); err != nil {
		slog.Error("Error stopping collector", "error", err)
	}

	slog.Info("Collector stopped")
}
