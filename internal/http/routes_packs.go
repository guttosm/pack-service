package http

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/middleware"
	"github.com/guttosm/pack-service/internal/service"
)

// PackRoutes handles pack-related route registration.
type PackRoutes struct {
	handler          *Handler
	packSizesHandler *PackSizesHandler
}

// NewPackRoutes creates a new PackRoutes instance.
func NewPackRoutes(calculator service.PackCalculator, packSizesService service.PackSizesService) *PackRoutes {
	handler := NewHandler(calculator, packSizesService)
	
	var packSizesHandler *PackSizesHandler
	if packSizesService != nil {
		packSizesHandler = NewPackSizesHandler(packSizesService, calculator)
	}
	
	return &PackRoutes{
		handler:          handler,
		packSizesHandler: packSizesHandler,
	}
}

// RegisterPublicRoutes registers public pack routes (when auth is disabled).
func (r *PackRoutes) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/calculate", r.handler.CalculatePacks)
	
	if r.packSizesHandler != nil {
		rg.GET("/pack-sizes", r.packSizesHandler.GetActivePackSizes)
		rg.PUT("/pack-sizes", r.packSizesHandler.UpdatePackSizes)
		rg.GET("/pack-sizes/history", r.packSizesHandler.ListPackSizes)
	}
}

// RegisterProtectedRoutes registers protected pack routes (when auth is enabled).
func (r *PackRoutes) RegisterProtectedRoutes(protected *gin.RouterGroup, cfg *RouterConfig) {
	// Get permission IDs for authorization
	packsReadPermID, packsWritePermID := r.getPermissionIDs(cfg)
	
	// Helper to create authorization middleware
	authMiddleware := func(permID string) []gin.HandlerFunc {
		if permID != "" && cfg.RoleService != nil && cfg.PermissionService != nil {
			return []gin.HandlerFunc{
				middleware.RequireAuthorization(middleware.AuthorizationConfig{
					RequiredPermissions: []string{permID},
				}, cfg.RoleService, cfg.PermissionService),
			}
		}
		return nil
	}
	
	// Register calculate endpoint
	if writeAuth := authMiddleware(packsWritePermID); writeAuth != nil {
		protected.POST("/calculate", append(writeAuth, r.handler.CalculatePacks)...)
	} else {
		protected.POST("/calculate", r.handler.CalculatePacks)
	}
	
	// Register pack sizes endpoints if service is available
	if r.packSizesHandler != nil {
		r.registerPackSizesRoutes(protected, authMiddleware, packsReadPermID, packsWritePermID)
	}
}

// registerPackSizesRoutes registers pack sizes endpoints with optional authorization.
func (r *PackRoutes) registerPackSizesRoutes(
	protected *gin.RouterGroup,
	authMiddleware func(string) []gin.HandlerFunc,
	packsReadPermID, packsWritePermID string,
) {
	// GET /pack-sizes
	if readAuth := authMiddleware(packsReadPermID); readAuth != nil {
		protected.GET("/pack-sizes", append(readAuth, r.packSizesHandler.GetActivePackSizes)...)
		protected.GET("/pack-sizes/history", append(readAuth, r.packSizesHandler.ListPackSizes)...)
	} else {
		protected.GET("/pack-sizes", r.packSizesHandler.GetActivePackSizes)
		protected.GET("/pack-sizes/history", r.packSizesHandler.ListPackSizes)
	}
	
	// PUT /pack-sizes
	if writeAuth := authMiddleware(packsWritePermID); writeAuth != nil {
		protected.PUT("/pack-sizes", append(writeAuth, r.packSizesHandler.UpdatePackSizes)...)
	} else {
		protected.PUT("/pack-sizes", r.packSizesHandler.UpdatePackSizes)
	}
}

// getPermissionIDs fetches permission IDs from the permission service.
func (r *PackRoutes) getPermissionIDs(cfg *RouterConfig) (packsReadPermID, packsWritePermID string) {
	if cfg.PermissionService == nil {
		return "", ""
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	packsReadPermID = cfg.PermissionService.GetPermissionIDByResourceAndAction(ctx, "packs", "read")
	packsWritePermID = cfg.PermissionService.GetPermissionIDByResourceAndAction(ctx, "packs", "write")
	
	return packsReadPermID, packsWritePermID
}

// GetHandler returns the underlying pack handler.
func (r *PackRoutes) GetHandler() *Handler {
	return r.handler
}
