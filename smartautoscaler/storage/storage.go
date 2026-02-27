package storage

import (
	"time"

	"github.com/kudmo/CoolPA/storage/metrics"
)

type Storage struct {
	MetricsStore *metrics.AppMetricsStore
	CallGraph    map[string][]string
}

func NewStorage() *Storage {
	return &Storage{
		MetricsStore: metrics.NewAppMetricsStore(time.Duration(time.Second*300), time.Duration(time.Second*15)),
		CallGraph:    make(map[string][]string),
	}
}
