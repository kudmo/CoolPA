package scaler

import "time"

// ScalerConfig holds parameters that control the autoscaling loop.
type ScalerConfig struct {
	// Interval is how often the scaler runs an analysis cycle.
	Interval time.Duration

	// Cooldown is the minimum time between scaling actions to avoid
	// rapid oscillations.
	Cooldown time.Duration

	// SLO is the target 95th percentile latency in milliseconds.
	SLO float64

	// Lambda is the smoothing factor for metric analysis (0..1).
	Lambda float64

	// AnomalyServicesCount limits the number of services scaled
	// per cycle.
	AnomalyServicesCount int

	// Namespace is the Kubernetes namespace the scaler operates on.
	Namespace string
}
