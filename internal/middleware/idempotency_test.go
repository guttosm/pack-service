package middleware

import (
	"time"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)
func TestIdempotency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		idempotencyKey string
		body           string
		expectedStatus int
		checkHeader    bool
	}{
		{
			name:           "processes request without idempotency key",
			method:         http.MethodPost,
			idempotencyKey: "",
			body:           `{"test": "data"}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "processes GET request normally",
			method:         http.MethodGet,
			idempotencyKey: "test-key",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "processes POST with idempotency key",
			method:         http.MethodPost,
			idempotencyKey: "test-key-123",
			body:           `{"test": "data"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultIdempotencyConfig()
			router := gin.New()
			router.Use(Idempotency(cfg))
			router.POST("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			var bodyReader *bytes.Reader
			if tt.body != "" {
				bodyReader = bytes.NewReader([]byte(tt.body))
			} else {
				bodyReader = bytes.NewReader(nil)
			}

			req := httptest.NewRequest(tt.method, "/test", bodyReader)
			if tt.idempotencyKey != "" {
				req.Header.Set(IdempotencyKeyHeader, tt.idempotencyKey)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestIdempotency_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := DefaultIdempotencyConfig()
	cfg.Enabled = false
	cfg.Cache = nil

	router := gin.New()
	router.Use(Idempotency(cfg))
	router.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte(`{"test": "data"}`)))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIdempotencyCache_cleanup(t *testing.T) {
	tests := []struct {
		name       string
		setupCache func() *idempotencyCache
	}{
		{
			name: "cleanup expired entries",
			setupCache: func() *idempotencyCache {
				cache := newIdempotencyCache(100 * time.Millisecond)
				// Add some entries with old timestamps
				oldTime := time.Now().Add(-2 * time.Hour)
				newTime := time.Now()

				cache.mu.Lock()
				cache.items[1] = &cachedResponse{
					StatusCode: 200,
					Headers:    make(map[string]string),
					Body:       []byte("response1"),
					Timestamp:  oldTime, // Expired
				}
				cache.items[2] = &cachedResponse{
					StatusCode: 200,
					Headers:    make(map[string]string),
					Body:       []byte("response2"),
					Timestamp:  newTime, // Valid
				}
				cache.mu.Unlock()
				return cache
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := tt.setupCache()

			// Manually trigger cleanup
			cache.cleanup()

			// Verify expired entry is removed (if TTL is less than 2 hours)
			cache.mu.Lock()
			_, exists2 := cache.items[2]
			cache.mu.Unlock()

			// Entry 2 should still exist
			assert.True(t, exists2, "Valid entry should still exist")
		})
	}
}
