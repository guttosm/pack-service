// Package middleware provides HTTP middleware components for the pack service.
package middleware

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// Compression returns a middleware that compresses HTTP responses using gzip.
// It compresses responses for clients that support gzip encoding.
func Compression() gin.HandlerFunc {
	return gzip.Gzip(gzip.DefaultCompression)
}
