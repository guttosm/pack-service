//go:build !integration

package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/guttosm/pack-service/config"
)

func TestInitializeServices(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.CacheConfig
		validate func(*testing.T, *ServiceComponents)
	}{
		{
			name: "creates service with default config",
			cfg: config.CacheConfig{
				Size: 0,
				TTL:  0,
			},
			validate: func(t *testing.T, components *ServiceComponents) {
				assert.NotNil(t, components)
				assert.NotNil(t, components.Calculator)
			},
		},
		{
			name: "creates service with cache enabled",
			cfg: config.CacheConfig{
				Size: 1000,
				TTL:  5 * time.Minute,
			},
			validate: func(t *testing.T, components *ServiceComponents) {
				assert.NotNil(t, components)
				assert.NotNil(t, components.Calculator)
			},
		},
		{
			name: "creates service with custom pack sizes",
			cfg: config.CacheConfig{
				PackSizes: []int{100, 200, 500},
				Size:      0,
				TTL:       0,
			},
			validate: func(t *testing.T, components *ServiceComponents) {
				assert.NotNil(t, components)
				assert.NotNil(t, components.Calculator)
			},
		},
		{
			name: "creates service with cache and custom pack sizes",
			cfg: config.CacheConfig{
				PackSizes: []int{50, 100, 250},
				Size:      500,
				TTL:       10 * time.Minute,
			},
			validate: func(t *testing.T, components *ServiceComponents) {
				assert.NotNil(t, components)
				assert.NotNil(t, components.Calculator)
			},
		},
		{
			name: "creates service with zero cache size disables cache",
			cfg: config.CacheConfig{
				Size: 0,
				TTL:  5 * time.Minute,
			},
			validate: func(t *testing.T, components *ServiceComponents) {
				assert.NotNil(t, components)
				assert.NotNil(t, components.Calculator)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := InitializeServices(tt.cfg)
			if tt.validate != nil {
				tt.validate(t, components)
			}
		})
	}
}

func TestServiceComponents_Calculator(t *testing.T) {
	components := InitializeServices(config.CacheConfig{
		Size: 100,
		TTL:  time.Minute,
	})

	assert.NotNil(t, components.Calculator)

	// Test that calculator works
	result := components.Calculator.Calculate(251)
	assert.Equal(t, 251, result.OrderedItems)
	assert.Greater(t, result.TotalItems, result.OrderedItems)
	assert.NotEmpty(t, result.Packs)
}
