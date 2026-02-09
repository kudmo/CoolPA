package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
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

func main() {
	fmt.Println("Hello METANIT.COM!")
}
