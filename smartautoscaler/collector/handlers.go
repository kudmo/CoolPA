package collector

type MetricHandler interface {
	Handle(result MetricResult)
	HandleBatch(results []MetricResult)
}
