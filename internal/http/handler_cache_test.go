package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPackSizesCache_NewPackSizesCache(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{
			name: "create cache with 30s TTL",
			ttl:  30 * time.Second,
		},
		{
			name: "create cache with 1 minute TTL",
			ttl:  time.Minute,
		},
		{
			name: "create cache with zero TTL",
			ttl:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newPackSizesCache(tt.ttl)

			assert.NotNil(t, cache)
			assert.Equal(t, tt.ttl, cache.ttl)

			// Should return nil initially
			assert.Nil(t, cache.get())
		})
	}
}

func TestPackSizesCache_SetAndGet(t *testing.T) {
	tests := []struct {
		name     string
		ttl      time.Duration
		sizes    []int
		wantGet  bool
		waitTime time.Duration
	}{
		{
			name:    "set and get immediately",
			ttl:     time.Second,
			sizes:   []int{250, 500, 1000},
			wantGet: true,
		},
		{
			name:    "set empty slice",
			ttl:     time.Second,
			sizes:   []int{},
			wantGet: true,
		},
		{
			name:     "get after expiration",
			ttl:      50 * time.Millisecond,
			sizes:    []int{100, 200},
			wantGet:  false,
			waitTime: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newPackSizesCache(tt.ttl)

			cache.set(tt.sizes)

			if tt.waitTime > 0 {
				time.Sleep(tt.waitTime)
			}

			result := cache.get()

			if tt.wantGet {
				assert.Equal(t, tt.sizes, result)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestPackSizesCache_Invalidate(t *testing.T) {
	cache := newPackSizesCache(time.Minute)

	// Set some values
	sizes := []int{250, 500, 1000}
	cache.set(sizes)

	// Should be cached
	assert.Equal(t, sizes, cache.get())

	// Invalidate
	cache.invalidate()

	// Should be nil now
	assert.Nil(t, cache.get())
}

func TestPackSizesCache_SetDoesNotOverwriteValid(t *testing.T) {
	cache := newPackSizesCache(time.Minute)

	// Set first values
	firstSizes := []int{100, 200}
	cache.set(firstSizes)

	// Try to set different values (should not overwrite since cache is still valid)
	secondSizes := []int{500, 1000}
	cache.set(secondSizes)

	// Should still have first values
	result := cache.get()
	assert.Equal(t, firstSizes, result)
}

func TestPackSizesCache_SetAfterExpiration(t *testing.T) {
	cache := newPackSizesCache(50 * time.Millisecond)

	// Set first values
	firstSizes := []int{100, 200}
	cache.set(firstSizes)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Set new values
	secondSizes := []int{500, 1000}
	cache.set(secondSizes)

	// Should have second values
	result := cache.get()
	assert.Equal(t, secondSizes, result)
}

func TestWithPackSizesCacheTTL(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{
			name: "1 minute TTL",
			ttl:  time.Minute,
		},
		{
			name: "5 seconds TTL",
			ttl:  5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(nil, nil, WithPackSizesCacheTTL(tt.ttl))

			assert.NotNil(t, handler)
			assert.NotNil(t, handler.packSizesCache)
			assert.Equal(t, tt.ttl, handler.packSizesCache.ttl)
		})
	}
}

func TestHandler_InvalidatePackSizesCache(t *testing.T) {
	handler := NewHandler(nil, nil)

	// Set some values in cache
	handler.packSizesCache.set([]int{100, 200, 500})

	// Verify cache is set
	assert.NotNil(t, handler.packSizesCache.get())

	// Invalidate
	handler.InvalidatePackSizesCache()

	// Verify cache is cleared
	assert.Nil(t, handler.packSizesCache.get())
}

func TestPackSizesCache_ConcurrentAccess(t *testing.T) {
	cache := newPackSizesCache(time.Minute)
	done := make(chan bool)

	// Concurrent sets
	go func() {
		for i := 0; i < 100; i++ {
			cache.set([]int{i, i * 2})
		}
		done <- true
	}()

	// Concurrent gets
	go func() {
		for i := 0; i < 100; i++ {
			cache.get()
		}
		done <- true
	}()

	// Concurrent invalidates
	go func() {
		for i := 0; i < 100; i++ {
			cache.invalidate()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Should not panic - just verify it completes
	assert.True(t, true)
}
