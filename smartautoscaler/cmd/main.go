// cmd/smartautoscaler/main.go
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	applierprovider "github.com/kudmo/CoolPA/internal/applier/providers"
	"github.com/kudmo/CoolPA/internal/metrics/providers/cache"
	"github.com/kudmo/CoolPA/internal/metrics/providers/prometheus"
	"github.com/kudmo/CoolPA/internal/metrics/providers/prometheus/collector"
	"github.com/kudmo/CoolPA/internal/scaler"
	"github.com/kudmo/CoolPA/logger"

	"github.com/kudmo/CoolPA/config"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "/etc/config/config.yaml", "Path to configuration file")
	flag.StringVar(&configPath, "c", "/etc/config/config.yaml", "Path to configuration file (shorthand)")

	flag.Parse()

	cfg := &config.AppConfig{}
	if err := cfg.LoadFromYAML(configPath); err != nil {
		logger.Error("main", "failed to load config", "error", err, "path", configPath)
		os.Exit(1)
	}

	logger.Init(cfg.Logger)

	// Configs
	scalerConfig := scaler.ScalerConfig{
		Interval:             cfg.AnalyzerInterval,
		Cooldown:             cfg.ScalingCooldown,
		SLO:                  float64(cfg.SLO),
		Lambda:               cfg.Lambda,
		Namespace:            cfg.ScalingNamespace,
		AnomalyServicesCount: cfg.AnomalyServicesCount,
	}

	prometheusProviderConfig := prometheus.PrometheusMetricsProviderConfig{
		ScalingNamespace: cfg.ScalingNamespace,
		PrometheusConfig: collector.PrometheusCollectorConfig{
			PrometheusURL: cfg.PrometheusURL,
		},
	}

	cacheConfig := cache.CachedMetricsProviderConfig{
		TTL:          cfg.AnalyzerInterval,
		MaxCacheSize: 100,
	}

	// Components creating

	prometheusRepository, err := prometheus.NewPrometheusMetricsProvider(prometheusProviderConfig)
	if err != nil {
		logger.Error("main", "failed to create prometheus collector", "error", err)
	}

	metricsRepository := cache.NewCachedMetricsRepository(prometheusRepository, cacheConfig)

	applier, err := applierprovider.NewK8sApplier()
	if err != nil {
		logger.Error("main", "failed to create applier", "error", err)
	}

	scaler := scaler.NewScaler(
		scalerConfig,
		metricsRepository,
		applier,
	)

	// Context creating and starting components

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := scaler.Start(ctx); err != nil {
		logger.Error("main", "failed to start scaler", "error", err)
	}
	logger.Info("main", "scaler started")

	// Components destroying

	<-sigChan
	logger.Info("main", "shutdown signal received")

	if err := scaler.Stop(); err != nil {
		logger.Error("main", "error stopping scaler", "error", err)
	}
	logger.Info("main", "scaler stopped")
}
