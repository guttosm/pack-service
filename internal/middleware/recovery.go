package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/logger"
)

// Recovery returns a middleware that recovers from panics and returns a 500 error.
// It logs the panic details with the request ID for debugging.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID := GetRequestID(c)
				log := logger.Logger()
				log.Error().
					Str("request_id", requestID).
					Interface("panic", err).
					Msg("PANIC recovered")

				c.AbortWithStatusJSON(http.StatusInternalServerError, dto.ErrorResponse{
					Error:   dto.ErrCodeInternal,
					Message: "An unexpected error occurred",
				})
			}
		}()
		c.Next()
	}
}
