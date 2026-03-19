package config

type LoggerConfig struct {
	Level  string `yaml:"level" validate:"required,oneof=debug info warn error"`
	Format string `yaml:"format" validate:"required,oneof=json text"`
}
