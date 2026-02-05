package http

import (
	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/middleware"
	"github.com/guttosm/pack-service/internal/service"
)

// AuthRoutes handles authentication route registration.
type AuthRoutes struct {
	handler     *AuthHandler
	authService service.AuthService
}

// NewAuthRoutes creates a new AuthRoutes instance.
func NewAuthRoutes(authService service.AuthService) *AuthRoutes {
	return &AuthRoutes{
		handler:     NewAuthHandler(authService),
		authService: authService,
	}
}

// RegisterPublicRoutes registers public authentication routes.
// These routes don't require authentication.
func (r *AuthRoutes) RegisterPublicRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/login", r.handler.Login)
		auth.POST("/register", r.handler.Register)
		auth.POST("/refresh", r.handler.RefreshToken)
	}
}

// RegisterProtectedRoutes registers protected authentication routes.
// These routes require JWT authentication.
func (r *AuthRoutes) RegisterProtectedRoutes(rg *gin.RouterGroup, cfg *RouterConfig) {
	// Apply JWT authentication middleware
	protected := rg.Group("")
	protected.Use(middleware.JWTAuth(r.authService))

	// Apply user-specific rate limiting if configured
	if cfg.RateLimit > 0 {
		userLimiter := middleware.NewRateLimiter(cfg.RateLimit, cfg.RateWindow)
		protected.Use(userLimiter.UserRateLimit())
	}

	// Register logout endpoint
	protected.POST("/auth/logout", r.handler.Logout)
}

// GetProtectedGroup returns a protected router group with JWT auth middleware applied.
// This is useful for other route registrars that need to register protected routes.
func (r *AuthRoutes) GetProtectedGroup(rg *gin.RouterGroup, cfg *RouterConfig) *gin.RouterGroup {
	protected := rg.Group("")
	protected.Use(middleware.JWTAuth(r.authService))

	if cfg.RateLimit > 0 {
		userLimiter := middleware.NewRateLimiter(cfg.RateLimit, cfg.RateWindow)
		protected.Use(userLimiter.UserRateLimit())
	}

	return protected
}
