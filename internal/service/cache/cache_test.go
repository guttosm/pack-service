//go:build !integration

package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/guttosm/pack-service/internal/domain/model"
)

// TestCacheInterface tests that the Cache interface is properly defined.
// This is a compile-time test to ensure the interface contract is correct.
func TestCacheInterface(t *testing.T) {
	// This test ensures the Cache interface methods are properly defined
	// by attempting to use them with a mock implementation
	
	var cache Cache = &mockCache{}
	
	result, found := cache.Get(100)
	assert.False(t, found)
	assert.Equal(t, model.PackResult{}, result)
	
	cache.Set(100, model.PackResult{OrderedItems: 100})
	cache.Stop()
}

// TestCacheWithMetricsInterface tests that the CacheWithMetrics interface is properly defined.
func TestCacheWithMetricsInterface(t *testing.T) {
	var cache CacheWithMetrics = &mockCacheWithMetrics{}
	
	result, found := cache.Get(100)
	assert.False(t, found)
	assert.Equal(t, model.PackResult{}, result)
	
	cache.Set(100, model.PackResult{OrderedItems: 100})
	
	metrics := cache.Metrics()
	assert.Equal(t, Metrics{}, metrics)
	
	cache.Stop()
}

// TestMetricsStructure tests the Metrics struct.
func TestMetricsStructure(t *testing.T) {
	metrics := Metrics{
		Hits:      10,
		Misses:    5,
		Evictions: 2,
		Size:      8,
		Capacity:  10,
	}
	
	assert.Equal(t, int64(10), metrics.Hits)
	assert.Equal(t, int64(5), metrics.Misses)
	assert.Equal(t, int64(2), metrics.Evictions)
	assert.Equal(t, 8, metrics.Size)
	assert.Equal(t, 10, metrics.Capacity)
}

// mockCache is a minimal implementation of Cache for testing.
type mockCache struct{}

func (m *mockCache) Get(key int) (model.PackResult, bool) {
	return model.PackResult{}, false
}

func (m *mockCache) Set(key int, value model.PackResult) {}

func (m *mockCache) Invalidate(key int) {}

func (m *mockCache) Clear() {}

func (m *mockCache) Stop() {}

// mockCacheWithMetrics is a minimal implementation of CacheWithMetrics for testing.
type mockCacheWithMetrics struct {
	mockCache
}

func (m *mockCacheWithMetrics) Metrics() Metrics {
	return Metrics{}
}
