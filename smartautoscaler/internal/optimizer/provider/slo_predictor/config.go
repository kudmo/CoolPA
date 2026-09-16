package slopredictor

import "time"

type FitnessConfig struct {
	Window time.Duration
	Lambda float64
}
