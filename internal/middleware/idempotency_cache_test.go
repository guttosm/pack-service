package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIdempotencyCache_Get(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*idempotencyCache)
		key           int
		expectedFound bool
	}{
		{
			name: "returns cached response when exists",
			setup: func(cache *idempotencyCache) {
				resp := &cachedResponse{
					StatusCode: 200,
					Headers:    map[string]string{"Content-Type": "application/json"},
					Body:       []byte(`{"data": "test"}`),
					Timestamp:  time.Now(),
				}
				cache.Set(123, resp)
			},
			key:           123,
			expectedFound: true,
		},
		{
			name: "returns false when key not found",
			setup: func(cache *idempotencyCache) {
				// No setup
			},
			key:           999,
			expectedFound: false,
		},
		{
			name: "returns false when expired",
			setup: func(cache *idempotencyCache) {
				// Set with old timestamp by directly manipulating cache
				cache.mu.Lock()
				cache.items[456] = &cachedResponse{
					StatusCode: 200,
					Headers:    map[string]string{},
					Body:       []byte(`{}`),
					Timestamp:  time.Now().Add(-2 * time.Minute),
				}
				cache.mu.Unlock()
			},
			key:           456,
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newIdempotencyCache(50 * time.Millisecond)
			tt.setup(cache)
			resp, found := cache.Get(tt.key)

			assert.Equal(t, tt.expectedFound, found, "Cache lookup result mismatch for test: %s", tt.name)
			if tt.expectedFound {
				assert.NotNil(t, resp)
				if resp != nil {
					assert.Equal(t, 200, resp.StatusCode)
				}
			}
		})
	}
}

func TestIdempotencyCache_Set(t *testing.T) {
	cache := newIdempotencyCache(time.Minute)

	resp := &cachedResponse{
		StatusCode: 200,
		Headers:    map[string]string{"X-Test": "value"},
		Body:       []byte(`{"test": "data"}`),
		Timestamp:  time.Now(),
	}

	cache.Set(100, resp)

	retrieved, found := cache.Get(100)
	assert.True(t, found)
	assert.Equal(t, resp.StatusCode, retrieved.StatusCode)
	assert.Equal(t, resp.Headers, retrieved.Headers)
}

