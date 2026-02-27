package storage

import (
	"fmt"
	"time"

	"github.com/kudmo/CoolPA/collector"
)

type StorageHandler struct {
	Store *Storage
}

func (h *StorageHandler) handleResourceMetrics(result collector.MetricResult) {
	h.Store.MetricsStore.Add(
		result.Labels["pod"],
		result.QueryName,
		result.Timestamp,
		result.Value,
	)
}

func (h *StorageHandler) Handle(result collector.MetricResult) {
	if _, ok := result.Labels["pod"]; ok {
		h.handleResourceMetrics(result)
	}
}

func (h *StorageHandler) HandleBatch(results []collector.MetricResult) {
	for _, r := range results {
		h.Handle(r)
	}

	for _, s := range h.Store.MetricsStore.ServiceNames() {
		fmt.Printf("[DEBUG] %s:\n", s)
		for _, m := range h.Store.MetricsStore.MetricNamesForService(s) {
			fmt.Printf("[DEBUG]\t%s(agg value): %f\n", m,
				h.Store.MetricsStore.AggregateService(
					s,
					m,
					time.Duration(time.Second*30),
					func(vals []float64) float64 {
						var sum float64 = 0
						for _, v := range vals {
							sum += v
						}
						return sum / float64(len(vals))
					})[0].Value,
			)
		}
		// h.Store.MetricsStore.AggregateService(s)
	}
}
