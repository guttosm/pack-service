// Package app provides application initialization and dependency injection.
package app

import (
	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/config"
	"github.com/guttosm/pack-service/internal/http"
	"github.com/guttosm/pack-service/internal/service"
)

// InitializeApp creates and wires all application dependencies.
// This is the main orchestration function that initializes all components.
func InitializeApp(cfg config.Config) *gin.Engine {
	// Initialize logger first (needed by other components)
	InitializeLogger()

	// Initialize business services
	serviceComponents := InitializeServices(cfg.Cache)

	// Initialize database components (MongoDB repositories and services)
	defaultPackSizes := cfg.Cache.PackSizes
	if len(defaultPackSizes) == 0 {
		defaultPackSizes = service.DefaultPackSizes
	}
	dbComponents := InitializeDatabase(cfg.Database, defaultPackSizes)

	// Initialize router components (handlers and configuration)
	routerComponents := InitializeRouter(serviceComponents.Calculator, dbComponents, cfg)

	return http.NewRouter(routerComponents.Handler, routerComponents.HealthHandler, routerComponents.Config)
}
