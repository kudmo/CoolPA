package collector

import "time"

type PrometheusCollectorConfig struct {
	PrometheusURL string
	RangeStep     time.Duration
}
