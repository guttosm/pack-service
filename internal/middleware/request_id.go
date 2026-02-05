// Package middleware provides HTTP middleware components for the pack service.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the HTTP header name for request ID.
	RequestIDHeader = "X-Request-ID"
)

// ContextKey type for context keys to avoid collisions.
type ContextKey string

const (
	// RequestIDKey is the context key for request ID.
	RequestIDKey ContextKey = "request_id"
)

// RequestID returns a middleware that ensures each request has a unique ID.
// If the client provides X-Request-ID header, it will be used.
// Otherwise, a new UUID v4 will be generated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(string(RequestIDKey), requestID)
		c.Header(RequestIDHeader, requestID)
		c.Next()
	}
}

// GetRequestID retrieves the request ID from the gin context.
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(string(RequestIDKey)); exists {
		if requestID, ok := id.(string); ok {
			return requestID
		}
	}
	return ""
}
