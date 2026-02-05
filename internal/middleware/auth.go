package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/i18n"
)

const (
	// APIKeyHeader is the HTTP header name for API key authentication.
	APIKeyHeader = "X-API-Key"
	// APIKeyQuery is the query parameter name for API key authentication.
	APIKeyQuery = "api_key"
)

// APIKeyAuth returns a middleware that validates API keys.
// It checks the X-API-Key header first, then falls back to api_key query parameter.
// If validKeys is nil or empty, authentication is disabled.
func APIKeyAuth(validKeys map[string]bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(validKeys) == 0 {
			c.Next()
			return
		}

		key := c.GetHeader(APIKeyHeader)
		if key == "" {
			key = c.Query(APIKeyQuery)
		}

		locale := i18n.GetLocale(c)
		requestID := GetRequestID(c)

		if key == "" {
			errorResp := dto.NewError(dto.ErrCodeUnauthorized, i18n.GetTranslator().Translate(i18n.ErrKeyAPIKeyRequired, locale)).
				WithRequestID(requestID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResp)
			return
		}

		if !validKeys[key] {
			errorResp := dto.NewError(dto.ErrCodeUnauthorized, i18n.GetTranslator().Translate(i18n.ErrKeyInvalidAPIKey, locale)).
				WithRequestID(requestID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResp)
			return
		}

		c.Next()
	}
}
