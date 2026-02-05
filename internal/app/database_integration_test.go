//go:build integration

package app

import (
	"context"
	"testing"
	"time"

	"github.com/guttosm/pack-service/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeDatabase_Integration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Use shared container with unique database names for each subtest
	uri := getSharedContainerURI()

	t.Run("initialize with enabled database", func(t *testing.T) {
		t.Parallel()
		dbName := sanitizeDBNameForApp(t.Name())
		cfg := config.DatabaseConfig{
			URI:                            uri,
			DatabaseName:                   dbName,
			LogsTTL:                        30 * 24 * time.Hour,
			Enabled:                        true,
			CircuitBreakerFailureThreshold: 5,
			CircuitBreakerSuccessThreshold: 2,
			CircuitBreakerTimeout:          30 * time.Second,
		}

		defaultPackSizes := []int{100, 200, 500}
		components := InitializeDatabase(cfg, defaultPackSizes)

		require.NotNil(t, components)
		assert.NotNil(t, components.PackSizesRepo)
		assert.NotNil(t, components.LoggingService)
		assert.NotNil(t, components.PackSizesCircuitBreaker)
		assert.NotNil(t, components.LogsCircuitBreaker)
	})

	t.Run("initialize with disabled database", func(t *testing.T) {
		t.Parallel()
		cfg := config.DatabaseConfig{
			Enabled: false,
		}

		components := InitializeDatabase(cfg, []int{100, 200})
		assert.Nil(t, components)
	})

	t.Run("default pack sizes initialization", func(t *testing.T) {
		t.Parallel()
		dbName := sanitizeDBNameForApp(t.Name())
		cfg := config.DatabaseConfig{
			URI:                            uri,
			DatabaseName:                   dbName,
			LogsTTL:                        30 * 24 * time.Hour,
			Enabled:                        true,
			CircuitBreakerFailureThreshold: 5,
			CircuitBreakerSuccessThreshold: 2,
			CircuitBreakerTimeout:          30 * time.Second,
		}

		defaultPackSizes := []int{250, 500, 1000}
		components := InitializeDatabase(cfg, defaultPackSizes)

		require.NotNil(t, components)

		// Verify default pack sizes were created
		active, err := components.PackSizesRepo.GetActive(ctx)
		require.NoError(t, err)
		require.NotNil(t, active)
		assert.Equal(t, defaultPackSizes, active.Sizes)
	})

	t.Run("circuit breaker integration", func(t *testing.T) {
		t.Parallel()
		dbName := sanitizeDBNameForApp(t.Name())
		cfg := config.DatabaseConfig{
			URI:                            uri,
			DatabaseName:                   dbName,
			LogsTTL:                        30 * 24 * time.Hour,
			Enabled:                        true,
			CircuitBreakerFailureThreshold: 2,
			CircuitBreakerSuccessThreshold: 1,
			CircuitBreakerTimeout:          100 * time.Millisecond,
		}

		components := InitializeDatabase(cfg, []int{100, 200})
		require.NotNil(t, components)

		// Verify circuit breakers are initialized
		stats := components.PackSizesCircuitBreaker.GetStats()
		assert.Equal(t, "closed", stats.State)
		assert.True(t, stats.IsHealthy)

		logsStats := components.LogsCircuitBreaker.GetStats()
		assert.Equal(t, "closed", logsStats.State)
		assert.True(t, logsStats.IsHealthy)
	})
}
