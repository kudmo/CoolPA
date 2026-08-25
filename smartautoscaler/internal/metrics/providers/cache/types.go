package cache

import "time"

type cacheEntry struct {
	value     any
	expiresAt time.Time
}
