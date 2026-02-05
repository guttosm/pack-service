package app

import (
	"testing"
	"time"

	"github.com/guttosm/pack-service/config"
	"github.com/stretchr/testify/assert"
)

func TestInitializeApp(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		validate func(*testing.T, interface{})
	}{
		{
			name: "creates router with default config",
			cfg: config.Config{
				Server: config.ServerConfig{
					Port:       "8080",
					RateLimit:  100,
					RateWindow: time.Minute,
				},
				Cache: config.CacheConfig{
					Size: 1000,
					TTL:  5 * time.Minute,
				},
			},
			validate: func(t *testing.T, router interface{}) {
				assert.NotNil(t, router)
			},
		},
		{
			name: "creates router with auth enabled",
			cfg: config.Config{
				Server: config.ServerConfig{
					Port: "8080",
				},
				Auth: config.AuthConfig{
					Enabled: true,
					APIKeys: map[string]bool{"test-key": true},
				},
			},
			validate: func(t *testing.T, router interface{}) {
				assert.NotNil(t, router)
			},
		},
		{
			name: "creates router with custom pack sizes",
			cfg: config.Config{
				Server: config.ServerConfig{
					Port: "8080",
				},
				Cache: config.CacheConfig{
					PackSizes: []int{100, 200, 500},
				},
			},
			validate: func(t *testing.T, router interface{}) {
				assert.NotNil(t, router)
			},
		},
		{
			name: "creates router with cache disabled",
			cfg: config.Config{
				Server: config.ServerConfig{
					Port: "8080",
				},
				Cache: config.CacheConfig{
					Size: 0, // Disabled
				},
			},
			validate: func(t *testing.T, router interface{}) {
				assert.NotNil(t, router)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := InitializeApp(tt.cfg)
			if tt.validate != nil {
				tt.validate(t, router)
			}
		})
	}
}
