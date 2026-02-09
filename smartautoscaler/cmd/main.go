package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Config содержит конфигурацию сборщика
type Config struct {
	PrometheusURL string
	Interval      time.Duration
	Queries       []MetricQuery
}

// MetricQuery описывает один запрос к Prometheus
type MetricQuery struct {
	Name  string
	Query string
	Help  string
}

// MetricCollector основной сборщик метрик
type MetricCollector struct {
	api    v1.API
	config Config
	logger *log.Logger
}

// NewMetricCollector создает новый сборщик метрик
func NewMetricCollector(config Config) (*MetricCollector, error) {
	client, err := api.NewClient(api.Config{
		Address: config.PrometheusURL,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания клиента: %w", err)
	}

	return &MetricCollector{
		api:    v1.NewAPI(client),
		config: config,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}, nil
}

func getDefaultQueries() []MetricQuery {
	return []MetricQuery{
		{
			Name:  "up",
			Query: "up{job=\"test-go-app\"}",
			Help:  "Статус тестового Go сервиса (1 = up, 0 = down)",
		},
		{
			Name:  "http_requests_total",
			Query: "rate(http_requests_total[5m])",
			Help:  "Количество HTTP запросов в секунду за 5 минут",
		},
		{
			Name:  "cpu_temperature_celsius",
			Query: "cpu_temperature_celsius",
			Help:  "Текущая температура CPU (имитация)",
		},
		{
			Name:  "http_request_duration_seconds_bucket",
			Query: "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
			Help:  "95-й перцентиль времени ответа HTTP (секунды)",
		},
	}
}

// fetchMetric выполняет один запрос к Prometheus и логирует результат
func (c *MetricCollector) fetchMetric(ctx context.Context, query MetricQuery) {
	result, warnings, err := c.api.Query(ctx, query.Query, time.Now())
	if err != nil {
		c.logger.Printf("ОШИБКА запроса %s (%s): %v", query.Name, query.Query, err)
		return
	}

	if len(warnings) > 0 {
		c.logger.Printf("Предупреждения для %s: %v", query.Name, warnings)
	}

	c.logMetricResult(query, result)
}

// logMetricResult парсит и логирует результат Prometheus запроса
func (c *MetricCollector) logMetricResult(query MetricQuery, val model.Value) {
	switch val.Type() {
	case model.ValScalar:
		scalar := val.(*model.Scalar)
		c.logger.Printf("✅ %s: %s = %.4f", query.Name, query.Help, scalar.Value)

	case model.ValVector:
		vector := val.(model.Vector)
		if len(vector) == 0 {
			c.logger.Printf("⚠️  %s: нет данных", query.Name)
			return
		}
		for i, sample := range vector {
			labelStr := ""
			if len(sample.Metric) > 0 {
				labelStr = fmt.Sprintf(" %v", sample.Metric)
			}
			c.logger.Printf("📊 %s%s: %s = %.4f", query.Name, labelStr, query.Help, sample.Value)
			if i >= 2 && len(vector) > 3 {
				c.logger.Printf("📊 %s: ... и еще %d значений", query.Name, len(vector)-3)
				break
			}
		}

	case model.ValMatrix:
		matrix := val.(model.Matrix)
		if len(matrix) == 0 {
			c.logger.Printf("⚠️  %s: нет данных (матрица пуста)", query.Name)
			return
		}
		// Выводим информацию о временном ряде
		for i, series := range matrix {
			if len(series.Values) > 0 {
				lastValue := series.Values[len(series.Values)-1]
				c.logger.Printf("📈 %s %v: %s = %.4f (последнее из %d точек)",
					query.Name, series.Metric, query.Help, lastValue.Value, len(series.Values))
			}
			if i >= 1 {
				break // Ограничиваем вывод
			}
		}

	default:
		c.logger.Printf("❌ %s: неподдерживаемый тип данных %s", query.Name, val.Type())
	}
}

// collectMetrics выполняет сбор всех метрик из конфигурации
func (c *MetricCollector) collectMetrics(ctx context.Context) {
	c.logger.Println("=== Начало сбора метрик ===")
	defer c.logger.Println("=== Завершение сбора метрик ===")

	for _, query := range c.config.Queries {
		queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		c.fetchMetric(queryCtx, query)
		cancel()
		time.Sleep(500 * time.Millisecond)
	}
}

// Run запускает сборщик метрик с заданным интервалом
func (c *MetricCollector) Run() {
	c.logger.Printf("Запуск сборщика метрик")
	c.logger.Printf("Prometheus URL: %s", c.config.PrometheusURL)
	c.logger.Printf("Интервал сбора: %v", c.config.Interval)
	c.logger.Printf("Количество метрик: %d", len(c.config.Queries))

	ctx := context.Background()
	c.collectMetrics(ctx)

	ticker := time.NewTicker(c.config.Interval)
	defer ticker.Stop()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			c.collectMetrics(ctx)

		case sig := <-sigChan:
			c.logger.Printf("⚠️  Получен сигнал %v, завершение работы", sig)
			return
		}
	}
}

func main() {
	config := Config{
		PrometheusURL: "http://prometheus:9090",
		Interval:      15 * time.Second,
		Queries:       getDefaultQueries(),
	}

	collector, err := NewMetricCollector(config)
	if err != nil {
		log.Fatalf("Ошибка создания сборщика: %v", err)
	}

	collector.Run()
	log.Println("Сборщик метрик завершил работу")
}
