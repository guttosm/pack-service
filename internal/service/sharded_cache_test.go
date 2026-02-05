package service

import (
	"testing"
	"time"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/stretchr/testify/assert"
)

func TestNewShardedCache(t *testing.T) {
	tests := []struct {
		name      string
		capacity  int
		ttl       time.Duration
		numShards int
		wantShards int
	}{
		{
			name:       "default shards when zero",
			capacity:   100,
			ttl:        time.Minute,
			numShards:  0,
			wantShards: 16,
		},
		{
			name:       "default shards when negative",
			capacity:   100,
			ttl:        time.Minute,
			numShards:  -1,
			wantShards: 16,
		},
		{
			name:       "rounds up to power of 2",
			capacity:   100,
			ttl:        time.Minute,
			numShards:  3,
			wantShards: 4,
		},
		{
			name:       "exact power of 2",
			capacity:   100,
			ttl:        time.Minute,
			numShards:  8,
			wantShards: 8,
		},
		{
			name:       "rounds 5 to 8",
			capacity:   100,
			ttl:        time.Minute,
			numShards:  5,
			wantShards: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewShardedCache(tt.capacity, tt.ttl, tt.numShards)
			defer cache.Stop()

			assert.NotNil(t, cache)
			assert.Equal(t, tt.wantShards, cache.numShards)
			assert.Equal(t, tt.wantShards-1, cache.shardMask)
			assert.Len(t, cache.shards, tt.wantShards)
		})
	}
}

func TestShardedCache_GetSet(t *testing.T) {
	tests := []struct {
		name     string
		key      int
		value    model.PackResult
		wantHit  bool
	}{
		{
			name: "set and get single value",
			key:  100,
			value: model.PackResult{
				TotalItems: 100,
				Packs:      []model.Pack{{Size: 100, Quantity: 1}},
			},
			wantHit: true,
		},
		{
			name: "set and get zero key",
			key:  0,
			value: model.PackResult{
				TotalItems: 250,
				Packs:      []model.Pack{{Size: 250, Quantity: 1}},
			},
			wantHit: true,
		},
		{
			name: "set and get large key",
			key:  999999,
			value: model.PackResult{
				TotalItems: 1000000,
				Packs:      []model.Pack{{Size: 500, Quantity: 2000}},
			},
			wantHit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewShardedCache(100, time.Minute, 4)
			defer cache.Stop()

			// Initially should miss
			_, found := cache.Get(tt.key)
			assert.False(t, found)

			// Set value
			cache.Set(tt.key, tt.value)

			// Should now hit
			result, found := cache.Get(tt.key)
			assert.Equal(t, tt.wantHit, found)
			if tt.wantHit {
				assert.Equal(t, tt.value.TotalItems, result.TotalItems)
				assert.Equal(t, len(tt.value.Packs), len(result.Packs))
			}
		})
	}
}

func TestShardedCache_Invalidate(t *testing.T) {
	tests := []struct {
		name string
		keys []int
		invalidateKey int
	}{
		{
			name:          "invalidate existing key",
			keys:          []int{1, 2, 3},
			invalidateKey: 2,
		},
		{
			name:          "invalidate non-existing key",
			keys:          []int{1, 3},
			invalidateKey: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewShardedCache(100, time.Minute, 4)
			defer cache.Stop()

			// Set initial values
			for _, key := range tt.keys {
				cache.Set(key, model.PackResult{TotalItems: key})
			}

			// Invalidate
			cache.Invalidate(tt.invalidateKey)

			// Check invalidated key is gone
			_, found := cache.Get(tt.invalidateKey)
			assert.False(t, found)

			// Other keys should still exist
			for _, key := range tt.keys {
				if key != tt.invalidateKey {
					_, found := cache.Get(key)
					assert.True(t, found)
				}
			}
		})
	}
}

func TestShardedCache_Clear(t *testing.T) {
	cache := NewShardedCache(100, time.Minute, 4)
	defer cache.Stop()

	// Add some values
	for i := 0; i < 10; i++ {
		cache.Set(i, model.PackResult{TotalItems: i})
	}

	// Verify they exist
	for i := 0; i < 10; i++ {
		_, found := cache.Get(i)
		assert.True(t, found)
	}

	// Clear
	cache.Clear()

	// All should be gone
	for i := 0; i < 10; i++ {
		_, found := cache.Get(i)
		assert.False(t, found)
	}
}

func TestShardedCache_Metrics(t *testing.T) {
	cache := NewShardedCache(100, time.Minute, 4)
	defer cache.Stop()

	// Set some values
	for i := 0; i < 5; i++ {
		cache.Set(i, model.PackResult{TotalItems: i})
	}

	// Generate hits
	for i := 0; i < 5; i++ {
		cache.Get(i)
	}

	// Generate misses
	for i := 100; i < 105; i++ {
		cache.Get(i)
	}

	metrics := cache.Metrics()
	assert.Equal(t, int64(5), metrics.Hits)
	assert.Equal(t, int64(5), metrics.Misses)
}

func TestShardedCache_ShardDistribution(t *testing.T) {
	cache := NewShardedCache(100, time.Minute, 4)
	defer cache.Stop()

	// Add values that should be distributed across shards
	for i := 0; i < 100; i++ {
		cache.Set(i, model.PackResult{TotalItems: i})
	}

	// Verify all can be retrieved
	for i := 0; i < 100; i++ {
		result, found := cache.Get(i)
		assert.True(t, found)
		assert.Equal(t, i, result.TotalItems)
	}
}
