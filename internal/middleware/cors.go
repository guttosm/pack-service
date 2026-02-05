package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing (CORS).
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Always set CORS headers for development (localhost)
		// In production, restrict to specific domains
		if origin == "http://localhost:3000" || origin == "http://127.0.0.1:3000" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			// Default to allowing localhost:3000 in development
			c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		}
		
		// Set CORS headers using Writer.Header() to ensure they're set before compression
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-API-Key, Idempotency-Key, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")

		// Handle preflight OPTIONS request
		if c.Request.Method == "OPTIONS" {
			c.Writer.WriteHeader(204)
			c.Abort()
			return
		}

		c.Next()
	}
}
