//go:build integration

package app

import (
	"testing"
	"time"

	"github.com/guttosm/pack-service/config"
	"github.com/stretchr/testify/assert"
)

func TestInitializeApp_Integration(t *testing.T) {
	t.Parallel()

	// Use shared container with unique database names for each subtest
	uri := getSharedContainerURI()

	t.Run("initialize app with MongoDB enabled", func(t *testing.T) {
		t.Parallel()
		dbName := sanitizeDBNameForApp(t.Name())
		cfg := config.Config{
			Server: config.ServerConfig{
				Port:       "8080",
				RateLimit:  100,
				RateWindow: time.Minute,
			},
			Cache: config.CacheConfig{
				Size:      1000,
				TTL:       5 * time.Minute,
				PackSizes: []int{100, 200, 500},
			},
			Auth: config.AuthConfig{
				Enabled: false,
			},
			Database: config.DatabaseConfig{
				URI:                            uri,
				DatabaseName:                   dbName,
				LogsTTL:                        30 * 24 * time.Hour,
				Enabled:                        true,
				CircuitBreakerFailureThreshold: 5,
				CircuitBreakerSuccessThreshold: 2,
				CircuitBreakerTimeout:          30 * time.Second,
			},
		}

		router := InitializeApp(cfg)
		assert.NotNil(t, router)
	})

	t.Run("initialize app with MongoDB disabled", func(t *testing.T) {
		t.Parallel()
		cfg := config.Config{
			Server: config.ServerConfig{
				Port: "8080",
			},
			Database: config.DatabaseConfig{
				Enabled: false,
			},
		}

		router := InitializeApp(cfg)
		assert.NotNil(t, router)
	})

	t.Run("initialize app with custom pack sizes", func(t *testing.T) {
		t.Parallel()
		dbName := sanitizeDBNameForApp(t.Name())
		cfg := config.Config{
			Server: config.ServerConfig{
				Port: "8080",
			},
			Cache: config.CacheConfig{
				PackSizes: []int{250, 500, 1000},
			},
			Database: config.DatabaseConfig{
				URI:                            uri,
				DatabaseName:                   dbName,
				LogsTTL:                        30 * 24 * time.Hour,
				Enabled:                        true,
				CircuitBreakerFailureThreshold: 5,
				CircuitBreakerSuccessThreshold: 2,
				CircuitBreakerTimeout:          30 * time.Second,
			},
		}

		router := InitializeApp(cfg)
		assert.NotNil(t, router)
	})
}
