// Package middleware provides HTTP middleware components for the pack service.
package middleware

import (
	"bytes"
	"crypto/sha256"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// IdempotencyKeyHeader is the HTTP header name for idempotency key (RFC standard).
	IdempotencyKeyHeader = "Idempotency-Key"
	// IdempotencyKeyTTL is the TTL for cached idempotency responses.
	IdempotencyKeyTTL = 5 * time.Minute
)

// cachedResponse stores a cached HTTP response for idempotency.
type cachedResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	Timestamp  time.Time
}

// IdempotencyConfig holds configuration for idempotency middleware.
type IdempotencyConfig struct {
	Cache   *idempotencyCache
	TTL     time.Duration
	Enabled bool
}

// DefaultIdempotencyConfig returns default idempotency configuration.
func DefaultIdempotencyConfig() IdempotencyConfig {
	return IdempotencyConfig{
		Cache:   newIdempotencyCache(IdempotencyKeyTTL),
		TTL:     IdempotencyKeyTTL,
		Enabled: true,
	}
}

// Idempotency returns a middleware that handles idempotency using the Idempotency-Key header.
// If a request with the same idempotency key was processed recently, the cached response is returned.
func Idempotency(cfg IdempotencyConfig) gin.HandlerFunc {
	if !cfg.Enabled || cfg.Cache == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	return func(c *gin.Context) {
		// Only apply idempotency to POST, PUT, PATCH methods
		if c.Request.Method != http.MethodPost &&
			c.Request.Method != http.MethodPut &&
			c.Request.Method != http.MethodPatch {
			c.Next()
			return
		}

		key := c.GetHeader(IdempotencyKeyHeader)
		if key == "" {
			c.Next()
			return
		}

		// Create cache key from idempotency key + request method + path + body hash
		cacheKey := generateCacheKey(key, c.Request)

		// Try to get cached response
		if cachedResp, ok := cfg.Cache.Get(cacheKey); ok {
			// Return cached response
			for k, v := range cachedResp.Headers {
				c.Header(k, v)
			}
			c.Header("X-Idempotency-Replayed", "true")
			c.Data(cachedResp.StatusCode, "application/json", cachedResp.Body)
			c.Abort()
			return
		}

		// Capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
			statusCode:     http.StatusOK,
			headers:        make(map[string]string),
		}
		c.Writer = writer

		c.Next()

		// Cache successful responses (2xx)
		if writer.statusCode >= 200 && writer.statusCode < 300 {
			cachedResp := &cachedResponse{
				StatusCode: writer.statusCode,
				Headers:    writer.headers,
				Body:       writer.body.Bytes(),
				Timestamp:  time.Now(),
			}
			cfg.Cache.Set(cacheKey, cachedResp)
		}
	}
}

// generateCacheKey creates a unique cache key from idempotency key and request details.
func generateCacheKey(idempotencyKey string, req *http.Request) int {
	hasher := sha256.New()
	hasher.Write([]byte(idempotencyKey))
	hasher.Write([]byte(req.Method))
	hasher.Write([]byte(req.URL.Path))

	// Include request body hash if present
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		if len(bodyBytes) > 0 {
			hasher.Write(bodyBytes)
		}
	}

	hash := hasher.Sum(nil)
	// Convert first 8 bytes to int for cache key
	var key int
	for i := 0; i < 8 && i < len(hash); i++ {
		key = key<<8 | int(hash[i])
	}
	if key < 0 {
		key = -key
	}
	return key
}

// responseWriter captures the response for caching.
type responseWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	headers    map[string]string
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Header() http.Header {
	headers := w.ResponseWriter.Header()
	// Capture headers for caching
	for k, v := range headers {
		if len(v) > 0 {
			w.headers[k] = v[0]
		}
	}
	return headers
}
