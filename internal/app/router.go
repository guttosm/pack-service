// Package app provides router configuration.
package app

import (
	"github.com/guttosm/pack-service/config"
	"github.com/guttosm/pack-service/internal/http"
	"github.com/guttosm/pack-service/internal/repository"
	"github.com/guttosm/pack-service/internal/service"
)

// RouterComponents holds router-related components.
type RouterComponents struct {
	Handler       *http.Handler
	HealthHandler *http.HealthHandler
	Config        http.RouterConfig
}

// InitializeRouter initializes HTTP handlers and router configuration.
func InitializeRouter(
	calculator service.PackCalculator,
	dbComponents *DatabaseComponents,
	cfg config.Config,
) *RouterComponents {
	var packSizesRepo repository.PackSizesRepositoryInterface
	var loggingService service.LoggingService
	if dbComponents != nil {
		packSizesRepo = dbComponents.PackSizesRepo
		loggingService = dbComponents.LoggingService
	}

	// Initialize pack sizes service
	var packSizesService service.PackSizesService
	if packSizesRepo != nil {
		packSizesService = service.NewPackSizesService(packSizesRepo)
	}

	handler := http.NewHandler(calculator, packSizesService)
	healthHandler := http.NewHealthHandler()

	// Register circuit breakers for health monitoring
	if dbComponents != nil {
		if dbComponents.PackSizesCircuitBreaker != nil {
			healthHandler.RegisterCircuitBreaker("mongodb_pack_sizes", dbComponents.PackSizesCircuitBreaker)
		}
		if dbComponents.LogsCircuitBreaker != nil {
			healthHandler.RegisterCircuitBreaker("mongodb_logs", dbComponents.LogsCircuitBreaker)
		}
	}

	// Initialize authentication service
	var authService service.AuthService
	if dbComponents != nil && dbComponents.UserRepo != nil {
		authService = service.NewAuthService(
			dbComponents.UserRepo,
			dbComponents.RoleRepo,
			dbComponents.TokenRepo,
			cfg.Auth,
		)
	}

	// Initialize permission service
	var permissionService service.PermissionService
	if dbComponents != nil && dbComponents.PermissionRepo != nil {
		permissionService = service.NewPermissionService(dbComponents.PermissionRepo)
	}

	// Initialize role service
	var roleService service.RoleService
	if dbComponents != nil && dbComponents.RoleRepo != nil {
		roleService = service.NewRoleService(dbComponents.RoleRepo)
	}

	routerCfg := http.RouterConfig{
		RateLimit:         cfg.Server.RateLimit,
		RateWindow:        cfg.Server.RateWindow,
		EnableAuth:        cfg.Auth.Enabled,
		APIKeys:           cfg.Auth.APIKeys,
		EnableIdempotency: true,
		CORSOrigins:       cfg.Server.CORSOrigins,
		SwaggerUser:       cfg.Server.SwaggerUser,
		SwaggerPass:       cfg.Server.SwaggerPass,
		LoggingService:    loggingService,
		PackSizesService:  packSizesService,
		AuthService:       authService,
		RoleService:       roleService,
		PermissionService: permissionService,
	}

	return &RouterComponents{
		Handler:       handler,
		HealthHandler: healthHandler,
		Config:        routerCfg,
	}
}
