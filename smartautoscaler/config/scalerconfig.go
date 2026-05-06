package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ScalerConfig holds configuration parameters for the autoscaler.
type ScalerConfig struct {
	ScalingNamespace   string        `yaml:"scaling_namespace"`
	PrometheusURL      string        `yaml:"prometheus_url"`
	PrometheusInterval time.Duration `yaml:"prometheus_interval"`
	AnalyzerInterval   time.Duration `yaml:"analyzer_interval"`
	SLO                int           `yaml:"slo"`
	ScalingCooldown    time.Duration `yaml:"scaling_cooldown"`
	Lambda             float64       `yaml:"lambda"`
	Logger             LoggerConfig  `yaml:"logger"`
}

func (c *ScalerConfig) Validate() error {
	if c.ScalingNamespace == "" {
		return fmt.Errorf("scaling_namespace cannot be empty")
	}

	if c.PrometheusURL == "" {
		return fmt.Errorf("prometheus_url cannot be empty")
	}

	if c.PrometheusInterval <= 0 {
		return fmt.Errorf("prometheus_interval must be greater than 0")
	}

	if c.AnalyzerInterval <= 0 {
		return fmt.Errorf("analyzer_interval must be greater than 0")
	}

	minCooldown := 1 * time.Minute
	if c.ScalingCooldown <= minCooldown {
		return fmt.Errorf("scaling_cooldown must be greater than 1 minute (got: %v)", c.ScalingCooldown)
	}

	if c.AnalyzerInterval <= c.PrometheusInterval {
		return fmt.Errorf("analyzer_interval (%v) must be greater than prometheus_interval (%v)",
			c.AnalyzerInterval, c.PrometheusInterval)
	}

	if c.Lambda <= 0 || c.Lambda >= 1 {
		return fmt.Errorf("lambda must be between 0 and 1")
	}

	return nil
}

// LoadFromYAML reads configuration from a YAML file.
func (c *ScalerConfig) LoadFromYAML(filename string) error {
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

// LoadYAMLConfig is a convenience function that creates a new ScalerConfig
// and loads it from a YAML file.
func LoadYAMLConfig(filename string) (*ScalerConfig, error) {
	config := &ScalerConfig{}
	err := config.LoadFromYAML(filename)
	return config, err
}
