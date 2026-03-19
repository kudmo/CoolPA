package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ScalerConfig holds configuration parameters for the autoscaler.
type ScalerConfig struct {
	PrometheusURL      string        `yaml:"prometheus_url"`
	PrometheusInterval time.Duration `yaml:"prometheus_interval"`
	AnalyzerInterval   time.Duration `yaml:"analyzer_interval"`
	ScalingCooldown    time.Duration `yaml:"scaling_cooldown"`
	Logger             LoggerConfig  `yaml:"logger"`
}

// LoadFromYAML reads configuration from a YAML file.
func (c *ScalerConfig) LoadFromYAML(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, c)
}

// LoadYAMLConfig is a convenience function that creates a new ScalerConfig
// and loads it from a YAML file.
func LoadYAMLConfig(filename string) (*ScalerConfig, error) {
	config := &ScalerConfig{}
	err := config.LoadFromYAML(filename)
	return config, err
}
