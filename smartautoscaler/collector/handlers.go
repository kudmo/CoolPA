package collector

import (
	"fmt"
	"log"
	"sync"
)

// LogHandler обработчик, который логирует метрики
type LogHandler struct {
	logger *log.Logger
	mu     sync.Mutex
}

// NewLogHandler создает новый логгирующий обработчик
func NewLogHandler(logger *log.Logger) *LogHandler {
	return &LogHandler{logger: logger}
}

// Handle обрабатывает одну метрику
func (h *LogHandler) Handle(result MetricResult) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if result.Error != nil {
		h.logger.Printf("Metric %s error: %v", result.QueryName, result.Error)
		return
	}

	labelStr := ""
	if len(result.Labels) > 0 {
		labelStr = fmt.Sprintf(" %v", result.Labels)
	}

	h.logger.Printf("📊 %s%s: %s = %.4f",
		result.QueryName, labelStr, result.Help, result.Value)
}

// HandleBatch обрабатывает пакет метрик
func (h *LogHandler) HandleBatch(results []MetricResult) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(results) == 0 {
		h.logger.Println("No metrics collected")
		return
	}

	h.logger.Printf("📦 Collected %d metrics", len(results))

	seen := make(map[string]bool)

	for _, result := range results {
		labelStr := ""
		if len(result.Labels) > 0 {
			labelStr = fmt.Sprintf("%v", result.Labels)
		}
		key := fmt.Sprintf("%s|%s", result.QueryName, labelStr)

		if seen[key] {
			continue
		}
		seen[key] = true

		h.logger.Printf("📊 %s%s: %s = %.4f", result.QueryName, labelStr, result.Help, result.Value)
	}
}
