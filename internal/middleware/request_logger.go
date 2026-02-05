package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/logger"
	"github.com/guttosm/pack-service/internal/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RequestLogger returns a middleware that logs HTTP request details in JSON format.
// It logs: request ID, method, path, status code, latency, IP, and user agent.
// Uses async logger with worker pool when available, falls back to goroutine-per-request.
func RequestLogger(loggingService service.LoggingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := GetRequestID(c)

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		ip := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Create structured log entry for console
		log := logger.Logger().With().
			Str("request_id", requestID).
			Str("method", method).
			Str("path", path).
			Int("status_code", statusCode).
			Int64("duration_ms", latency.Milliseconds()).
			Str("ip", ip).
			Str("user_agent", userAgent).
			Logger()

		// Log level based on status code
		switch {
		case statusCode >= 500:
			log.Error().Msg("HTTP request")
		case statusCode >= 400:
			log.Warn().Msg("HTTP request")
		default:
			log.Info().Msg("HTTP request")
		}

		// Store in MongoDB if logging service is provided
		if loggingService != nil {
			entry := &model.LogEntry{
				Timestamp:  time.Now(),
				Level:      getLogLevel(statusCode),
				Message:    "HTTP request",
				RequestID:  requestID,
				Method:     method,
				Path:       path,
				StatusCode: statusCode,
				Duration:   latency.Milliseconds(),
				IP:         ip,
				UserAgent:  userAgent,
			}

			// Capture user information if available (from JWT middleware)
			if userID, exists := c.Get("user_id"); exists {
				if id, ok := userID.(primitive.ObjectID); ok {
					entry.UserID = id.Hex()
				}
			}
			if userEmail, exists := c.Get("user_email"); exists {
				if email, ok := userEmail.(string); ok {
					entry.UserEmail = email
				}
			}

			// Use async logger with worker pool if available
			if asyncLogger := GetAsyncLogger(); asyncLogger != nil {
				asyncLogger.Log(entry)
			} else {
				// Fallback to goroutine-per-request (legacy behavior)
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					_ = loggingService.CreateLog(ctx, entry)
				}()
			}
		}
	}
}

// getLogLevel returns the log level based on HTTP status code.
func getLogLevel(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "error"
	case statusCode >= 400:
		return "warn"
	default:
		return "info"
	}
}
