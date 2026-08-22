package storage

import "time"

type CacheConfig struct {
	Window time.Duration
	Step   time.Duration
}
