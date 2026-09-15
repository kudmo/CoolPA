// Command smartautoscaler is the entry point for the Smart AutoScaler
// application.
//
// The application monitors Kubernetes deployments in a specified
// namespace, collects metrics from Prometheus, and automatically
// scales deployments based on analyzed performance patterns.
//
// The application is configured via a YAML file (default:
// /etc/config/config.yaml) and can be gracefully shut down using
// SIGINT or SIGTERM signals.
//
// Usage:
//
//	smartautoscaler -config /path/to/config.yaml
//	smartautoscaler -c /path/to/config.yaml
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	applierprovider "github.com/kudmo/CoolPA/internal/applier/providers"
	"github.com/kudmo/CoolPA/internal/metrics/providers/cache"
	"github.com/kudmo/CoolPA/internal/metrics/providers/prometheus"
	"github.com/kudmo/CoolPA/internal/metrics/providers/prometheus/collector"
	"github.com/kudmo/CoolPA/internal/scaler"
	"github.com/kudmo/CoolPA/logger"

	"github.com/kudmo/CoolPA/config"
)

func main() {
	// Define command-line flags for configuration file path
	var configPath string
	flag.StringVar(&configPath, "config", "/etc/config/config.yaml", "Path to configuration file")
	flag.StringVar(&configPath, "c", "/etc/config/config.yaml", "Path to configuration file (shorthand)")

	// Parse command-line arguments
	flag.Parse()

	// Load and validate application configuration
	cfg := &config.AppConfig{}
	if err := cfg.LoadFromYAML(configPath); err != nil {
		logger.Error("main", "failed to load config", "error", err, "path", configPath)
		os.Exit(1)
	}

	// Initialize the global logger with configured settings
	logger.Init(cfg.Logger)

	// Configure the scaler component
	scalerConfig := scaler.ScalerConfig{
		Interval:             cfg.AnalyzerInterval,
		Cooldown:             cfg.ScalingCooldown,
		SLO:                  float64(cfg.SLO),
		Lambda:               cfg.Lambda,
		Namespace:            cfg.ScalingNamespace,
		AnomalyServicesCount: cfg.AnomalyServicesCount,
	}

	// Configure the Prometheus metrics provider
	prometheusProviderConfig := prometheus.PrometheusMetricsProviderConfig{
		ScalingNamespace: cfg.ScalingNamespace,
		PrometheusConfig: collector.PrometheusCollectorConfig{
			PrometheusURL: cfg.PrometheusURL,
			RangeStep:     5 * time.Second,
		},
	}

	// Configure the metrics cache
	cacheConfig := cache.CachedMetricsProviderConfig{
		TTL:          cfg.AnalyzerInterval / 2,
		MaxCacheSize: 100,
	}

	// Initialize application components

	// Create Prometheus metrics provider for data collection
	prometheusRepository, err := prometheus.NewPrometheusMetricsProvider(prometheusProviderConfig)
	if err != nil {
		logger.Error("main", "failed to create prometheus collector", "error", err)
		os.Exit(1)
	}

	// Wrap the Prometheus provider with caching for improved performance
	metricsRepository := cache.NewCachedMetricsRepository(prometheusRepository, cacheConfig)

	// Create Kubernetes applier for executing scaling operations
	applier, err := applierprovider.NewK8sApplier()
	if err != nil {
		logger.Error("main", "failed to create applier", "error", err)
		os.Exit(1)
	}

	// Create the main scaler instance
	scaler := scaler.NewScaler(
		scalerConfig,
		metricsRepository,
		applier,
	)

	// Set up context and signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Register signal handlers for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the scaler
	if err := scaler.Start(ctx); err != nil {
		logger.Error("main", "failed to start scaler", "error", err)
		os.Exit(1)
	}
	logger.Info("main", "scaler started")

	// Wait for shutdown signal
	<-sigChan
	logger.Info("main", "shutdown signal received")

	// Gracefully stop the scaler
	if err := scaler.Stop(); err != nil {
		logger.Error("main", "error stopping scaler", "error", err)
		os.Exit(1)
	}
	logger.Info("main", "scaler stopped")
}
