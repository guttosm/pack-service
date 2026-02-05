package http

import (
	"github.com/gin-gonic/gin"
)

// RouteGroup defines a group of routes that can be registered.
type RouteGroup interface {
	// RegisterRoutes registers routes to the given router group.
	RegisterRoutes(rg *gin.RouterGroup, cfg *RouterConfig)
}

// PublicRouteGroup defines routes that don't require authentication.
type PublicRouteGroup interface {
	// RegisterPublicRoutes registers public routes to the given router group.
	RegisterPublicRoutes(rg *gin.RouterGroup)
}

// ProtectedRouteGroup defines routes that require authentication.
type ProtectedRouteGroup interface {
	// RegisterProtectedRoutes registers protected routes to the given router group.
	RegisterProtectedRoutes(rg *gin.RouterGroup, cfg *RouterConfig)
}
