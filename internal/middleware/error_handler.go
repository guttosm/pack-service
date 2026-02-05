package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/i18n"
	"github.com/guttosm/pack-service/internal/logger"
)

// ErrorHandler returns a middleware that handles gin context errors.
// It provides centralized error handling and logging.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			requestID := GetRequestID(c)
			locale := i18n.GetLocale(c)
			
			log := logger.Logger()
			log.Error().
				Str("request_id", requestID).
				Str("error", err.Error()).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Msg("Request error")

			if !c.Writer.Written() {
				message := i18n.GetTranslator().Translate(i18n.ErrKeyInternalError, locale)
				errorResp := dto.NewError(dto.ErrCodeInternal, message).
					WithRequestID(requestID)
				c.JSON(http.StatusInternalServerError, errorResp)
			}
		}
	}
}
