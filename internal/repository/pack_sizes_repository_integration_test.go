//go:build integration

package repository

import (
	"context"
	"testing"

	"github.com/guttosm/pack-service/internal/circuitbreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackSizesRepository_Integration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Use shared container with unique database name
	db := setupTestDBFromSharedContainer(t)
	defer func() {
		require.NoError(t, db.Close(ctx))
	}()

	repo := NewPackSizesRepository(db)

	t.Run("get active when none exists", func(t *testing.T) {
		active, err := repo.GetActive(ctx)
		assert.NoError(t, err)
		assert.Nil(t, active)
	})

	t.Run("create pack sizes", func(t *testing.T) {
		sizes := []int{100, 200, 500}
		config, err := repo.Create(ctx, sizes, "test-user")
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, sizes, config.Sizes)
		assert.True(t, config.Active)
		assert.Equal(t, 1, config.Version)
		assert.Equal(t, "test-user", config.CreatedBy)
		assert.False(t, config.ID.IsZero())
	})

	t.Run("get active after create", func(t *testing.T) {
		active, err := repo.GetActive(ctx)
		require.NoError(t, err)
		require.NotNil(t, active)
		assert.Equal(t, []int{100, 200, 500}, active.Sizes)
		assert.True(t, active.Active)
	})

	t.Run("create new active deactivates old", func(t *testing.T) {
		oldActive, err := repo.GetActive(ctx)
		require.NoError(t, err)
		require.NotNil(t, oldActive)

		newSizes := []int{250, 500, 1000}
		newConfig, err := repo.Create(ctx, newSizes, "test-user-2")
		require.NoError(t, err)
		assert.NotNil(t, newConfig)

		active, err := repo.GetActive(ctx)
		require.NoError(t, err)
		require.NotNil(t, active)
		assert.Equal(t, newSizes, active.Sizes)
		assert.NotEqual(t, oldActive.ID, active.ID)
	})

	t.Run("update pack sizes", func(t *testing.T) {
		active, err := repo.GetActive(ctx)
		require.NoError(t, err)
		require.NotNil(t, active)

		updatedSizes := []int{150, 300, 600}
		updatedConfig, err := repo.Update(ctx, active.ID, updatedSizes, "test-updater")
		require.NoError(t, err)
		assert.Equal(t, updatedSizes, updatedConfig.Sizes)
		assert.Equal(t, active.Version+1, updatedConfig.Version)
	})

	t.Run("list all configs", func(t *testing.T) {
		configs, err := repo.List(ctx, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(configs), 2)
	})

	t.Run("list with limit", func(t *testing.T) {
		configs, err := repo.List(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, 1, len(configs))
	})
}

func TestPackSizesRepositoryWithCircuitBreaker_Integration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Use shared container with unique database name
	db := setupTestDBFromSharedContainer(t)
	defer func() {
		require.NoError(t, db.Close(ctx))
	}()

	repo := NewPackSizesRepository(db)
	cb := circuitbreaker.New(circuitbreaker.DefaultConfig())
	wrappedRepo := NewPackSizesRepositoryWithCircuitBreaker(repo, cb)

	t.Run("circuit breaker allows successful operations", func(t *testing.T) {
		sizes := []int{100, 200}
		config, err := wrappedRepo.Create(ctx, sizes, "test")
		require.NoError(t, err)
		assert.NotNil(t, config)

		active, err := wrappedRepo.GetActive(ctx)
		require.NoError(t, err)
		assert.NotNil(t, active)
	})

	t.Run("circuit breaker stats", func(t *testing.T) {
		stats := cb.GetStats()
		assert.Equal(t, "closed", stats.State)
		assert.True(t, stats.IsHealthy)
	})

	t.Run("circuit breaker GetCircuitBreaker", func(t *testing.T) {
		returnedCB := wrappedRepo.GetCircuitBreaker()
		assert.NotNil(t, returnedCB)
		assert.Equal(t, cb, returnedCB)
	})

	t.Run("circuit breaker Update", func(t *testing.T) {
		active, err := wrappedRepo.GetActive(ctx)
		require.NoError(t, err)
		if active != nil {
			updatedConfig, err := wrappedRepo.Update(ctx, active.ID, []int{300, 600}, "test-updater")
			require.NoError(t, err)
			assert.NotNil(t, updatedConfig)
		}
	})

	t.Run("circuit breaker List", func(t *testing.T) {
		configs, err := wrappedRepo.List(ctx, 5)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(configs), 0)
	})
}
