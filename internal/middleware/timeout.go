package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/i18n"
)

// TimeoutConfig holds configuration for the timeout middleware.
type TimeoutConfig struct {
	// Timeout is the maximum duration for request processing.
	Timeout time.Duration
	// ErrorMessage is the message to return when a timeout occurs.
	ErrorMessage string
}

// DefaultTimeoutConfig returns sensible defaults for the timeout middleware.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout:      30 * time.Second,
		ErrorMessage: "Request timeout",
	}
}

// Timeout returns a middleware that enforces request timeouts.
// This prevents slow requests from consuming resources indefinitely.
func Timeout(cfg TimeoutConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.Timeout)
		defer cancel()

		// Replace request context with the timeout context
		c.Request = c.Request.WithContext(ctx)

		// Mutex to protect concurrent access to gin context
		var mu sync.Mutex
		var finished bool

		// Channel to signal completion
		done := make(chan struct{})

		go func() {
			defer func() {
				// Recover from any panic in the handler
				recover() //nolint:errcheck
				close(done)
			}()
			c.Next()
			mu.Lock()
			finished = true
			mu.Unlock()
		}()

		select {
		case <-done:
			// Request completed normally
			return
		case <-ctx.Done():
			// Timeout occurred - check if handler already finished
			mu.Lock()
			defer mu.Unlock()
			if finished {
				return
			}
			if !c.Writer.Written() {
				locale := i18n.GetLocale(c)
				requestID := GetRequestID(c)
				translator := i18n.GetTranslator()

				message := cfg.ErrorMessage
				if translator != nil {
					message = translator.Translate(i18n.ErrKeyTimeout, locale)
				}

				errorResp := dto.NewError(dto.ErrCodeTimeout, message).
					WithRequestID(requestID)
				c.AbortWithStatusJSON(http.StatusGatewayTimeout, errorResp)
			}
		}
	}
}

// TimeoutWithDuration is a convenience function to create timeout middleware with a specific duration.
func TimeoutWithDuration(timeout time.Duration) gin.HandlerFunc {
	cfg := DefaultTimeoutConfig()
	cfg.Timeout = timeout
	return Timeout(cfg)
}
