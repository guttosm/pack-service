package http

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/metrics"
	"github.com/guttosm/pack-service/internal/middleware"
	"github.com/guttosm/pack-service/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RouterConfig holds router configuration options.
type RouterConfig struct {
	RateLimit         int
	RateWindow        time.Duration
	APIKeys           map[string]bool
	EnableAuth        bool
	EnableIdempotency bool
	CORSOrigins       []string
	SwaggerUser       string
	SwaggerPass       string
	LoggingService    service.LoggingService
	PackSizesService  service.PackSizesService
	AuthService       service.AuthService
	RoleService       service.RoleService
	PermissionService service.PermissionService
	Calculator        service.PackCalculator
}

// DefaultRouterConfig returns the default router configuration.
func DefaultRouterConfig() RouterConfig {
	return RouterConfig{
		RateLimit:  100,
		RateWindow: time.Minute,
		EnableAuth: false,
	}
}

// NewRouter creates and configures the Gin router for the pack service.
func NewRouter(handler *Handler, healthHandler *HealthHandler, cfg RouterConfig) *gin.Engine {
	router := gin.New()

	// Configure global middleware
	configureGlobalMiddleware(router, &cfg)

	// Register infrastructure routes (health, metrics, swagger)
	registerInfrastructureRoutes(router, healthHandler, &cfg)

	// Configure API routes
	api := router.Group("/api")
	configureAPIMiddleware(api, &cfg)

	// Register business routes based on authentication mode
	if cfg.AuthService != nil {
		registerAuthenticatedRoutes(api, handler, &cfg)
	} else {
		registerPublicRoutes(api, handler, &cfg)
	}

	return router
}

// configureGlobalMiddleware sets up middleware applied to all routes.
func configureGlobalMiddleware(router *gin.Engine, cfg *RouterConfig) {
	// CORS configuration
	allowedOrigins := cfg.CORSOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}
	corsConfig := cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "Accept-Language", "X-CSRF-Token", "Authorization", "X-Refresh-Token", "accept", "Cache-Control", "X-Requested-With", "X-API-Key", "Idempotency-Key", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
	router.Use(cors.New(corsConfig))

	// Core middleware stack
	router.Use(
		middleware.RequestID(),
		middleware.Recovery(),
		metrics.PrometheusMiddleware(),
		middleware.Compression(),
		middleware.RequestLogger(cfg.LoggingService),
		middleware.ErrorHandler(),
	)

	// Context setup middleware
	router.Use(func(c *gin.Context) {
		c.Set("logging_service", cfg.LoggingService)
		c.Next()
	})

	// Global rate limiting
	if cfg.RateLimit > 0 {
		limiter := middleware.NewRateLimiter(cfg.RateLimit, cfg.RateWindow)
		router.Use(limiter.RateLimit())
	}
}

// registerInfrastructureRoutes registers health, metrics, and documentation routes.
func registerInfrastructureRoutes(router *gin.Engine, healthHandler *HealthHandler, cfg *RouterConfig) {
	healthHandler.Register(router)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Swagger with optional basic auth
	if cfg.SwaggerUser != "" && cfg.SwaggerPass != "" {
		authorized := router.Group("/swagger", gin.BasicAuth(gin.Accounts{
			cfg.SwaggerUser: cfg.SwaggerPass,
		}))
		authorized.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	} else {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
}

// configureAPIMiddleware sets up middleware for the API group.
func configureAPIMiddleware(api *gin.RouterGroup, cfg *RouterConfig) {
	// Idempotency middleware
	if cfg.EnableIdempotency {
		idempotencyCfg := middleware.DefaultIdempotencyConfig()
		api.Use(middleware.Idempotency(idempotencyCfg))
	}

	// API key authentication (when JWT auth is not enabled)
	if cfg.EnableAuth && cfg.AuthService == nil && len(cfg.APIKeys) > 0 {
		api.Use(middleware.APIKeyAuth(cfg.APIKeys))
	}
}

// registerAuthenticatedRoutes registers routes when JWT authentication is enabled.
func registerAuthenticatedRoutes(api *gin.RouterGroup, handler *Handler, cfg *RouterConfig) {
	// Create auth routes
	authRoutes := NewAuthRoutes(cfg.AuthService)

	// Register public auth routes (login, register, refresh)
	authRoutes.RegisterPublicRoutes(api)

	// Get protected group with JWT auth
	protected := authRoutes.GetProtectedGroup(api, cfg)

	// Register logout route
	protected.POST("/auth/logout", authRoutes.handler.Logout)

	// Create and register pack routes
	packRoutes := NewPackRoutes(handler.calculator, cfg.PackSizesService)
	packRoutes.RegisterProtectedRoutes(protected, cfg)
}

// registerPublicRoutes registers routes when authentication is disabled.
func registerPublicRoutes(api *gin.RouterGroup, handler *Handler, cfg *RouterConfig) {
	if handler == nil {
		return
	}
	packRoutes := NewPackRoutes(handler.calculator, cfg.PackSizesService)
	packRoutes.RegisterPublicRoutes(api)
}
