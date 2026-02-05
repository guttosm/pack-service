package cache

import "github.com/guttosm/pack-service/internal/domain/model"

// Cache defines the interface for cache operations.
type Cache interface {
	Get(key int) (model.PackResult, bool)
	Set(key int, value model.PackResult)
	Invalidate(key int)
	Clear()
	Stop()
}

// Metrics provides cache performance metrics.
type Metrics struct {
	Hits       int64
	Misses     int64
	Evictions  int64
	Size       int
	Capacity   int
}

// CacheWithMetrics extends Cache with metrics reporting.
type CacheWithMetrics interface {
	Cache
	Metrics() Metrics
}
