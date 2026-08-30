package config

import (
	"fmt"
	"os"
	"time"

	"github.com/kudmo/CoolPA/logger"
	"gopkg.in/yaml.v3"
)

// AppConfig holds configuration parameters for the autoscaler.
type AppConfig struct {
	// The namespace where the autoscaler will monitor and scale deployments.
	// This field is required and cannot be empty.
	ScalingNamespace string `yaml:"scaling_namespace"`

	// The base URL of the Prometheus server
	// used for collecting metrics that drive scaling decisions.
	// This field is required and cannot be empty.
	PrometheusURL string `yaml:"prometheus_url"`

	// How frequently the autoscaler
	// analyzes metrics and evaluates scaling decisions.
	// Must be greater than zero.
	AnalyzerInterval time.Duration `yaml:"analyzer_interval"`

	// The target service level
	// that the autoscaler aims to maintain, expressed
	// as a 95th percentile of latency, in milliseconds.
	SLO int `yaml:"slo"`

	// The minimum time between
	// consecutive scaling operations to prevent rapid
	// oscillations. Must be greater than 1 minute.
	ScalingCooldown time.Duration `yaml:"scaling_cooldown"`

	// The value balances SLO compliance against resource savings.
	// Range: 0..1.
	//   0 — ignore SLO risk entirely (maximize savings)
	//   1 — prioritize SLO compliance (use more resources)
	Lambda float64 `yaml:"lambda"`

	// The number of services
	// that will be considered anomalous,
	// i.e., the maximum number of services that will be scaled up at once.
	AnomalyServicesCount int `yaml:"anomaly_services_count"`

	// Configuration settings for the application logger.
	Logger logger.LoggerConfig `yaml:"logger"`
}

// Validate checks that all configuration parameters meet
// the required constraints. It returns an error describing
// the first invalid parameter encountered, or nil if the
// configuration is valid.
//
// Validation rules:
//   - ScalingNamespace and PrometheusURL must not be empty
//   - AnalyzerInterval must be greater than zero
//   - ScalingCooldown must be greater than 1 minute
//   - Lambda must be between 0 and 1 (exclusive)
func (c *AppConfig) Validate() error {
	if c.ScalingNamespace == "" {
		return fmt.Errorf("scaling_namespace cannot be empty")
	}

	if c.PrometheusURL == "" {
		return fmt.Errorf("prometheus_url cannot be empty")
	}

	if c.AnalyzerInterval <= 0 {
		return fmt.Errorf("analyzer_interval must be greater than 0")
	}

	minCooldown := 1 * time.Minute
	if c.ScalingCooldown <= minCooldown {
		return fmt.Errorf("scaling_cooldown must be greater than 1 minute (got: %v)", c.ScalingCooldown)
	}

	if c.Lambda <= 0 || c.Lambda >= 1 {
		return fmt.Errorf("lambda must be between 0 and 1")
	}

	return nil
}

// LoadFromYAML reads configuration from a YAML file and
// populates the AppConfig struct. The method performs
// validation after parsing to ensure the configuration
// is valid.
//
// It returns an error if the file cannot be read, the
// YAML cannot be parsed, or the configuration fails
// validation.
func (c *AppConfig) LoadFromYAML(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := c.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

// LoadYAMLConfig is a convenience function that creates a new
// AppConfig instance and loads it from the specified YAML file.
//
// It returns the populated configuration or an error if loading
// or validation fails.
func LoadYAMLConfig(filename string) (*AppConfig, error) {
	config := &AppConfig{}
	err := config.LoadFromYAML(filename)
	return config, err
}
