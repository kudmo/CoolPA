package storage

import "github.com/kudmo/CoolPA/storage/metrics"

type Storage struct {
	MetricsStore *metrics.AppMetricsStore
	CallGraph    map[string]string
}

func NewStorage() *Storage {
	return &Storage{
		MetricsStore: metrics.NewAppMetricsStore(),
		CallGraph:    make(map[string]string),
	}
}
