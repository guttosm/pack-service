package service

import (
	"testing"
	"time"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/service/cache"
	"github.com/stretchr/testify/assert"
)

func TestTTLCache_Get(t *testing.T) {
	tests := []struct {
		name          string
		setupCache    func() *ttlCache
		key           int
		expectedValue model.PackResult
		expectedFound bool
	}{
		{
			name: "returns value when exists and not expired",
			setupCache: func() *ttlCache {
				c := newTTLCache(10, time.Minute)
				c.Set(100, model.PackResult{OrderedItems: 100, TotalItems: 250})
				return c
			},
			key:           100,
			expectedValue: model.PackResult{OrderedItems: 100, TotalItems: 250},
			expectedFound: true,
		},
		{
			name: "returns false when key not found",
			setupCache: func() *ttlCache {
				return newTTLCache(10, time.Minute)
			},
			key:           999,
			expectedFound: false,
		},
		{
			name: "returns false when expired",
			setupCache: func() *ttlCache {
				c := newTTLCache(10, 50*time.Millisecond)
				c.Set(100, model.PackResult{OrderedItems: 100})
				time.Sleep(100 * time.Millisecond)
				return c
			},
			key:           100,
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := tt.setupCache()
			value, found := cache.Get(tt.key)

			assert.Equal(t, tt.expectedFound, found)
			if tt.expectedFound {
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}

func TestTTLCache_Set(t *testing.T) {
	tests := []struct {
		name       string
		capacity   int
		operations []struct {
			op   string
			key  int
			value model.PackResult
		}
		validate func(*testing.T, *ttlCache)
	}{
		{
			name:     "evicts LRU when at capacity",
			capacity: 2,
			operations: []struct {
				op   string
				key  int
				value model.PackResult
			}{
				{"set", 1, model.PackResult{OrderedItems: 1}},
				{"set", 2, model.PackResult{OrderedItems: 2}},
				{"set", 3, model.PackResult{OrderedItems: 3}},
			},
			validate: func(t *testing.T, c *ttlCache) {
				_, ok1 := c.Get(1)
				_, ok2 := c.Get(2)
				_, ok3 := c.Get(3)
				assert.False(t, ok1, "first entry evicted")
				assert.True(t, ok2)
				assert.True(t, ok3)
			},
		},
		{
			name:     "updates existing entry",
			capacity: 10,
			operations: []struct {
				op   string
				key  int
				value model.PackResult
			}{
				{"set", 100, model.PackResult{OrderedItems: 100, TotalItems: 250}},
				{"set", 100, model.PackResult{OrderedItems: 100, TotalItems: 500}},
			},
			validate: func(t *testing.T, c *ttlCache) {
				value, ok := c.Get(100)
				assert.True(t, ok)
				assert.Equal(t, 500, value.TotalItems)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newTTLCache(tt.capacity, time.Minute)
			for _, op := range tt.operations {
				cache.Set(op.key, op.value)
			}
			if tt.validate != nil {
				tt.validate(t, cache)
			}
		})
	}
}

func TestTTLCache_Stop(t *testing.T) {
	cache := newTTLCache(10, time.Minute)
	cache.Set(100, model.PackResult{OrderedItems: 100})

	// Stop should not panic
	assert.NotPanics(t, func() {
		cache.Stop()
	})
}

func TestTTLCache_Metrics(t *testing.T) {
	cache := newTTLCache(10, time.Minute)

	// Perform operations
	cache.Set(100, model.PackResult{OrderedItems: 100})
	cache.Get(100) // hit
	cache.Get(200) // miss
	cache.Set(200, model.PackResult{OrderedItems: 200})
	cache.Set(300, model.PackResult{OrderedItems: 300})

	metrics := cache.Metrics()
	assert.Greater(t, metrics.Hits, int64(0))
	assert.Greater(t, metrics.Misses, int64(0))
	assert.Equal(t, 3, metrics.Size)
	assert.Equal(t, 10, metrics.Capacity)
}

func TestTTLCache_ImplementsInterface(t *testing.T) {
	var _ cache.Cache = (*ttlCache)(nil)
	var _ cache.CacheWithMetrics = (*ttlCache)(nil)
}

func TestTTLCache_Concurrency(t *testing.T) {
	cache := newTTLCache(100, time.Minute)
	defer cache.Stop()

	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(key int) {
			for j := 0; j < 10; j++ {
				cache.Set(key*100+j, model.PackResult{OrderedItems: key*100 + j})
				cache.Get(key*100 + j)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := cache.Metrics()
	assert.Greater(t, metrics.Size, 0)
}

func TestTTLCache_Eviction(t *testing.T) {
	cache := newTTLCache(3, time.Minute)
	defer cache.Stop()

	// Fill cache to capacity
	cache.Set(1, model.PackResult{OrderedItems: 1})
	cache.Set(2, model.PackResult{OrderedItems: 2})
	cache.Set(3, model.PackResult{OrderedItems: 3})

	// Access 2 and 3 to make 1 the LRU
	cache.Get(2)
	cache.Get(3)

	// Add 4, should evict 1
	cache.Set(4, model.PackResult{OrderedItems: 4})

	_, ok1 := cache.Get(1)
	_, ok2 := cache.Get(2)
	_, ok3 := cache.Get(3)
	_, ok4 := cache.Get(4)

	assert.False(t, ok1, "entry 1 should be evicted")
	assert.True(t, ok2)
	assert.True(t, ok3)
	assert.True(t, ok4)

	metrics := cache.Metrics()
	assert.Equal(t, int64(1), metrics.Evictions)
}

func TestTTLCache_Cleanup(t *testing.T) {
	cache := newTTLCache(10, 50*time.Millisecond)
	defer cache.Stop()

	// Add entries
	cache.Set(1, model.PackResult{OrderedItems: 1})
	cache.Set(2, model.PackResult{OrderedItems: 2})

	// Wait for expiration (must be > TTL + cachedTime update interval of 100ms)
	time.Sleep(200 * time.Millisecond)

	// Manually trigger cleanup
	cache.cleanup()

	// Entries should be removed
	metrics := cache.Metrics()
	assert.Equal(t, 0, metrics.Size)
}

func TestTTLCache_RemoveTail(t *testing.T) {
	cache := newTTLCache(2, time.Minute)
	defer cache.Stop()

	cache.Set(1, model.PackResult{OrderedItems: 1})
	cache.Set(2, model.PackResult{OrderedItems: 2})

	// Force eviction by adding third item
	cache.Set(3, model.PackResult{OrderedItems: 3})

	// First item should be evicted (LRU)
	_, ok := cache.Get(1)
	assert.False(t, ok)
}

func TestTTLCache_MoveToFront(t *testing.T) {
	cache := newTTLCache(3, time.Minute)
	defer cache.Stop()

	cache.Set(1, model.PackResult{OrderedItems: 1})
	cache.Set(2, model.PackResult{OrderedItems: 2})
	cache.Set(3, model.PackResult{OrderedItems: 3})

	// Access 1 to move it to front (making 2 the LRU)
	cache.Get(1)

	// Add 4, should evict 2 (LRU) since capacity is 3
	cache.Set(4, model.PackResult{OrderedItems: 4})

	_, ok1 := cache.Get(1)
	_, ok2 := cache.Get(2)
	_, ok3 := cache.Get(3)
	_, ok4 := cache.Get(4)

	assert.True(t, ok1, "entry 1 should still exist (was accessed)")
	assert.False(t, ok2, "entry 2 should be evicted (was LRU)")
	assert.True(t, ok3, "entry 3 should still exist")
	assert.True(t, ok4, "entry 4 should exist")
}

func TestTTLCache_ExpiredEntryRemoval(t *testing.T) {
	cache := newTTLCache(10, 50*time.Millisecond)
	defer cache.Stop()

	cache.Set(100, model.PackResult{OrderedItems: 100})

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Get should return false and remove expired entry
	value, found := cache.Get(100)
	assert.False(t, found)
	assert.Equal(t, model.PackResult{}, value)

	metrics := cache.Metrics()
	assert.Equal(t, 0, metrics.Size)
}

func TestTTLCache_UpdateExistingEntry(t *testing.T) {
	cache := newTTLCache(10, time.Minute)
	defer cache.Stop()

	cache.Set(100, model.PackResult{OrderedItems: 100, TotalItems: 250})
	value1, _ := cache.Get(100)
	assert.Equal(t, 250, value1.TotalItems)

	// Update same key
	cache.Set(100, model.PackResult{OrderedItems: 100, TotalItems: 500})
	value2, found := cache.Get(100)

	assert.True(t, found)
	assert.Equal(t, 500, value2.TotalItems)

	metrics := cache.Metrics()
	assert.Equal(t, 1, metrics.Size, "should still have only one entry")
}
