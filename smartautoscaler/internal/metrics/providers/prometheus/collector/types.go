package collector

import "time"

type MetricQuery struct {
	Name  string
	Query string
}

type MetricResult struct {
	QueryName string
	Help      string
	Value     float64
	Labels    map[string]string
	Timestamp time.Time
	Error     error
}
