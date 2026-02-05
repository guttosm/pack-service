// Package app provides service initialization.
package app

import (
	"github.com/guttosm/pack-service/config"
	"github.com/guttosm/pack-service/internal/service"
)

// ServiceComponents holds service-related components.
type ServiceComponents struct {
	Calculator service.PackCalculator
}

// InitializeServices initializes business logic services.
func InitializeServices(cfg config.CacheConfig) *ServiceComponents {
	var opts []service.Option

	if len(cfg.PackSizes) > 0 {
		opts = append(opts, service.WithPackSizes(cfg.PackSizes))
	}

	if cfg.Size > 0 {
		opts = append(opts, service.WithCache(cfg.Size, cfg.TTL))
	}

	calculator := service.NewPackCalculatorService(opts...)

	return &ServiceComponents{
		Calculator: calculator,
	}
}
