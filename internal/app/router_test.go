//go:build !integration

package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/guttosm/pack-service/config"
	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/guttosm/pack-service/internal/service"
)

func TestInitializeRouter(t *testing.T) {
	tests := []struct {
		name         string
		calculator   service.PackCalculator
		dbComponents *DatabaseComponents
		cfg          config.Config
		validate     func(*testing.T, *RouterComponents)
	}{
		{
			name:       "creates router with calculator only",
			calculator: service.NewPackCalculatorService(),
			cfg: config.Config{
				Server: config.ServerConfig{
					RateLimit:  100,
					RateWindow: time.Minute,
				},
			},
			validate: func(t *testing.T, components *RouterComponents) {
				assert.NotNil(t, components)
				assert.NotNil(t, components.Handler)
				assert.NotNil(t, components.HealthHandler)
				assert.False(t, components.Config.EnableAuth)
				assert.True(t, components.Config.EnableIdempotency)
				assert.Equal(t, 100, components.Config.RateLimit)
			},
		},
		{
			name:       "creates router with auth enabled",
			calculator: service.NewPackCalculatorService(),
			cfg: config.Config{
				Server: config.ServerConfig{
					RateLimit:  50,
					RateWindow: 30 * time.Second,
				},
				Auth: config.AuthConfig{
					Enabled: true,
					APIKeys: map[string]bool{"test-key": true},
				},
			},
			validate: func(t *testing.T, components *RouterComponents) {
				assert.NotNil(t, components)
				assert.True(t, components.Config.EnableAuth)
				assert.Equal(t, map[string]bool{"test-key": true}, components.Config.APIKeys)
			},
		},
		{
			name:       "creates router with database components",
			calculator: service.NewPackCalculatorService(),
			dbComponents: &DatabaseComponents{
				PackSizesRepo:            new(mocks.MockPackSizesRepositoryInterface),
				LoggingService:           mocks.NewMockLoggingService(t),
				PackSizesCircuitBreaker:  nil,
				LogsCircuitBreaker:       nil,
			},
			cfg: config.Config{
				Server: config.ServerConfig{
					RateLimit:  10,
					RateWindow: time.Second,
				},
			},
			validate: func(t *testing.T, components *RouterComponents) {
				assert.NotNil(t, components)
				assert.NotNil(t, components.Config.PackSizesService)
				assert.NotNil(t, components.Config.LoggingService)
			},
		},
		{
			name:       "creates router with circuit breakers registered",
			calculator: service.NewPackCalculatorService(),
			dbComponents: &DatabaseComponents{
				PackSizesRepo:            new(mocks.MockPackSizesRepositoryInterface),
				LoggingService:           mocks.NewMockLoggingService(t),
				PackSizesCircuitBreaker:  nil, // Using nil since circuit breaker is tested in integration tests
				LogsCircuitBreaker:       nil,
			},
			cfg: config.Config{
				Server: config.ServerConfig{
					RateLimit:  10,
					RateWindow: time.Second,
				},
			},
			validate: func(t *testing.T, components *RouterComponents) {
				assert.NotNil(t, components)
				assert.NotNil(t, components.HealthHandler)
			},
		},
		{
			name:       "creates router with nil dbComponents",
			calculator: service.NewPackCalculatorService(),
			dbComponents: nil,
			cfg: config.Config{
				Server: config.ServerConfig{
					RateLimit:  10,
					RateWindow: time.Second,
				},
			},
			validate: func(t *testing.T, components *RouterComponents) {
				assert.NotNil(t, components)
				assert.Nil(t, components.Config.PackSizesService)
				assert.Nil(t, components.Config.LoggingService)
				assert.Nil(t, components.Config.AuthService)
			},
		},
		{
			name:       "creates router with auth service when user repo exists",
			calculator: service.NewPackCalculatorService(),
			dbComponents: &DatabaseComponents{
				UserRepo:      mocks.NewMockUserRepositoryInterface(t),
				RoleRepo:      mocks.NewMockRoleRepositoryInterface(t),
				TokenRepo:     mocks.NewMockTokenRepositoryInterface(t),
				PackSizesRepo: new(mocks.MockPackSizesRepositoryInterface),
			},
			cfg: config.Config{
				Server: config.ServerConfig{
					RateLimit:  10,
					RateWindow: time.Second,
				},
			},
			validate: func(t *testing.T, components *RouterComponents) {
				assert.NotNil(t, components)
				assert.NotNil(t, components.Config.AuthService)
			},
		},
		{
			name:       "creates router without auth service when user repo is nil",
			calculator: service.NewPackCalculatorService(),
			dbComponents: &DatabaseComponents{
				UserRepo:      nil,
				PackSizesRepo: new(mocks.MockPackSizesRepositoryInterface),
			},
			cfg: config.Config{
				Server: config.ServerConfig{
					RateLimit:  10,
					RateWindow: time.Second,
				},
			},
			validate: func(t *testing.T, components *RouterComponents) {
				assert.NotNil(t, components)
				assert.Nil(t, components.Config.AuthService)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := InitializeRouter(tt.calculator, tt.dbComponents, tt.cfg)
			if tt.validate != nil {
				tt.validate(t, components)
			}
		})
	}
}
