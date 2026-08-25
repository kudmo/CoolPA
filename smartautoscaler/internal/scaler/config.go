package scaler

import "time"

type ScalerConfig struct {
	Interval             time.Duration
	Cooldown             time.Duration
	SLO                  float64
	Lambda               float64
	AnomalyServicesCount int
	Namespace            string
}
