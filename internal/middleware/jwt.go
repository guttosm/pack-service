// Package middleware provides JWT authentication middleware.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/i18n"
	"github.com/guttosm/pack-service/internal/service"
)


// JWTAuth returns a middleware that validates JWT tokens.
func JWTAuth(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := i18n.GetLocale(c)
		requestID := GetRequestID(c)

		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			message := i18n.GetTranslator().Translate(i18n.ErrKeyTokenRequired, locale)
			errorResp := dto.NewError(dto.ErrCodeUnauthorized, message).
				WithRequestID(requestID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResp)
			return
		}

		// Extract token from "Bearer <token>"
		if !strings.HasPrefix(authHeader, "Bearer ") {
			message := i18n.GetTranslator().Translate(i18n.ErrKeyInvalidToken, locale)
			errorResp := dto.NewError(dto.ErrCodeUnauthorized, message).
				WithRequestID(requestID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResp)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			message := i18n.GetTranslator().Translate(i18n.ErrKeyTokenRequired, locale)
			errorResp := dto.NewError(dto.ErrCodeUnauthorized, message).
				WithRequestID(requestID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResp)
			return
		}

		// Validate token
		claims, err := authService.ValidateToken(c.Request.Context(), tokenString)
		if err != nil {
			message := i18n.GetTranslator().Translate(i18n.ErrKeyInvalidToken, locale)
			errorResp := dto.NewError(dto.ErrCodeUnauthorized, message).
				WithRequestID(requestID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResp)
			return
		}

		// Store user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Name)
		c.Set("user_roles", claims.Roles)
		c.Set("user_claims", claims)

		c.Next()
	}
}
