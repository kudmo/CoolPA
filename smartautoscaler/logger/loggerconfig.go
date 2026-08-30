package logger

// LoggerConfig holds configuration parameters for the application logger.
//
// The configuration supports two output formats (JSON and plain text)
// and four logging levels (debug, info, warn, error).
type LoggerConfig struct {
	// Level specifies the minimum logging level. Valid values are:
	//   - "debug": Logs all messages including debug information
	//   - "info":  Logs informational messages and above (default if invalid)
	//   - "warn":  Logs warning and error messages only
	//   - "error": Logs error messages only
	Level string `yaml:"level" validate:"required,oneof=debug info warn error"`

	// Format specifies the output format for log messages. Valid values are:
	//   - "json": Structured JSON output (recommended for production)
	//   - "text": Human-readable text output (default if invalid)
	Format string `yaml:"format" validate:"required,oneof=json text"`
}
