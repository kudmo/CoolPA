package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/kudmo/CoolPA/config"
)

// Init configures the global slog logger according to cfg.
func Init(cfg config.LoggerConfig) {
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

// For returns a logger with the `component` attribute attached.
func For(component string) *slog.Logger {
	return slog.Default().With("component", component)
}

// Convenience helpers to log with component name.
func Debug(component, msg string, kv ...any) {
	For(component).Debug(msg, kv...)
}

func Info(component, msg string, kv ...any) {
	For(component).Info(msg, kv...)
}

func Warn(component, msg string, kv ...any) {
	For(component).Warn(msg, kv...)
}

func Error(component, msg string, kv ...any) {
	For(component).Error(msg, kv...)
}
