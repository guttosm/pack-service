// Package middleware provides authorization middleware based on roles and permissions.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/i18n"
	"github.com/guttosm/pack-service/internal/service"
)

// AuthorizationConfig configures authorization requirements for a route.
type AuthorizationConfig struct {
	// RequiredRoles is a list of role IDs that are allowed to access the route.
	// If empty, any authenticated user can access.
	RequiredRoles []string
	// RequiredPermissions is a list of permission IDs that are required.
	// User must have at least one of these permissions through their roles.
	RequiredPermissions []string
	// RequireAllPermissions if true, user must have ALL permissions, otherwise ANY is sufficient.
	RequireAllPermissions bool
}

// RequireAuthorization returns a middleware that checks if the user has the required roles/permissions.
// This middleware must be used after JWTAuth middleware.
func RequireAuthorization(cfg AuthorizationConfig, roleService service.RoleService, permissionService service.PermissionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := i18n.GetLocale(c)
		requestID := GetRequestID(c)

		claimsInterface, exists := c.Get("user_claims")
		if !exists {
			message := i18n.GetTranslator().Translate(i18n.ErrKeyUnauthorized, locale)
			errorResp := dto.NewError(dto.ErrCodeUnauthorized, message).
				WithRequestID(requestID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResp)
			return
		}

		claims, ok := claimsInterface.(*dto.Claims)
		if !ok {
			message := i18n.GetTranslator().Translate(i18n.ErrKeyUnauthorized, locale)
			errorResp := dto.NewError(dto.ErrCodeUnauthorized, message).
				WithRequestID(requestID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResp)
			return
		}

		if len(cfg.RequiredRoles) > 0 {
			hasRequiredRole := false
			for _, requiredRole := range cfg.RequiredRoles {
				for _, userRole := range claims.Roles {
					if userRole == requiredRole {
						hasRequiredRole = true
						break
					}
				}
				if hasRequiredRole {
					break
				}
			}
			if !hasRequiredRole {
				message := i18n.GetTranslator().Translate(i18n.ErrKeyForbidden, locale)
				errorResp := dto.NewError(dto.ErrCodeForbidden, message).
					WithRequestID(requestID)
				c.AbortWithStatusJSON(http.StatusForbidden, errorResp)
				return
			}
		}

		if len(cfg.RequiredPermissions) > 0 {
			var userRoles []*model.Role
			if roleService != nil {
				for _, roleIDStr := range claims.Roles {
					roleID, err := primitive.ObjectIDFromHex(roleIDStr)
					if err != nil {
						continue
					}
					role, err := roleService.FindByID(c.Request.Context(), roleID)
					if err == nil && role != nil {
						userRoles = append(userRoles, role)
					}
				}
			}

			// Collect all permissions from user roles
			userPermissionIDs := make(map[string]bool)
			for _, role := range userRoles {
				for _, permID := range role.Permissions {
					userPermissionIDs[permID] = true
				}
			}

			// Check if user has required permissions
			if cfg.RequireAllPermissions {
				// User must have ALL required permissions
				for _, requiredPerm := range cfg.RequiredPermissions {
					if !userPermissionIDs[requiredPerm] {
						message := i18n.GetTranslator().Translate(i18n.ErrKeyForbidden, locale)
						errorResp := dto.NewError(dto.ErrCodeForbidden, message).
							WithRequestID(requestID)
						c.AbortWithStatusJSON(http.StatusForbidden, errorResp)
						return
					}
				}
			} else {
				// User must have AT LEAST ONE required permission
				hasPermission := false
				for _, requiredPerm := range cfg.RequiredPermissions {
					if userPermissionIDs[requiredPerm] {
						hasPermission = true
						break
					}
				}
				if !hasPermission {
					message := i18n.GetTranslator().Translate(i18n.ErrKeyForbidden, locale)
					errorResp := dto.NewError(dto.ErrCodeForbidden, message).
						WithRequestID(requestID)
					c.AbortWithStatusJSON(http.StatusForbidden, errorResp)
					return
				}
			}
		}

		c.Next()
	}
}
