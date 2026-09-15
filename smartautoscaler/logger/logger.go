package logger

import (
	"log/slog"
	"os"
	"time"
)

// Init configures the global slog logger according to the provided
// configuration. It sets up the appropriate log level, output format,
// and adds source information to all log entries.
//
// The logger automatically adds an "app" attribute with the value
// "smartautoscaler" to all log entries for easy filtering in
// log aggregation systems.
//
// If an invalid log level or format is specified, the function
// falls back to "info" level and "text" format respectively.
func Init(cfg LoggerConfig) {
	var logLevel slog.Level
	switch cfg.Level {
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
	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	case "text":
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	default:
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	}

	base := slog.New(handler).With("app", "smartautoscaler")
	slog.SetDefault(base)
}

// For returns a logger with the specified component attribute attached.
// This allows for easy filtering and grouping of log messages by
// application component.
//
// Example:
//
//	logger := logger.For("autoscaler")
//	logger.Info("starting scaling operation", "target", "deployment-1")
func For(component string) *slog.Logger {
	return slog.Default().With("component", component)
}

// Debug logs a debug-level message with the specified component name.
// Debug messages are typically used for detailed troubleshooting
// information that is not needed in normal operation.
//
// Parameters:
//   - component: Name of the application component generating the log
//   - msg: The log message
//   - kv: Optional key-value pairs for additional context (e.g., "user_id", 123)
func Debug(component, msg string, kv ...any) {
	For(component).Debug(msg, kv...)
}

// Info logs an info-level message with the specified component name.
// Info messages are used for general operational information about
// normal application behavior.
//
// Parameters:
//   - component: Name of the application component generating the log
//   - msg: The log message
//   - kv: Optional key-value pairs for additional context (e.g., "duration_ms", 250)
func Info(component, msg string, kv ...any) {
	For(component).Info(msg, kv...)
}

// Warn logs a warning-level message with the specified component name.
// Warning messages indicate potential issues that don't prevent
// the application from functioning but may require attention.
//
// Parameters:
//   - component: Name of the application component generating the log
//   - msg: The log message
//   - kv: Optional key-value pairs for additional context (e.g., "retry_count", 3)
func Warn(component, msg string, kv ...any) {
	For(component).Warn(msg, kv...)
}

// Error logs an error-level message with the specified component name.
// Error messages indicate failures that prevent normal operation
// and typically require immediate attention.
//
// Parameters:
//   - component: Name of the application component generating the log
//   - msg: The log message
//   - kv: Optional key-value pairs for additional context (e.g., "error", err)
func Error(component, msg string, kv ...any) {
	For(component).Error(msg, kv...)
}
