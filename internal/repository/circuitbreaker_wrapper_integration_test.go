//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/guttosm/pack-service/internal/circuitbreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackSizesRepositoryWithCircuitBreaker_Update(t *testing.T) {
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

	// Create initial config
	sizes := []int{100, 200, 500}
	config, err := wrappedRepo.Create(ctx, sizes, "test-user")
	require.NoError(t, err)
	require.NotNil(t, config)

	// Update via circuit breaker wrapper
	updatedSizes := []int{150, 300, 600}
	updatedConfig, err := wrappedRepo.Update(ctx, config.ID, updatedSizes, "test-updater")
	require.NoError(t, err)
	assert.NotNil(t, updatedConfig)
	assert.Equal(t, updatedSizes, updatedConfig.Sizes)
	assert.Equal(t, config.Version+1, updatedConfig.Version)
}

func TestPackSizesRepositoryWithCircuitBreaker_List(t *testing.T) {
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

	// Create some configs
	_, _ = wrappedRepo.Create(ctx, []int{100, 200}, "user1")
	_, _ = wrappedRepo.Create(ctx, []int{250, 500}, "user2")

	// List via circuit breaker wrapper
	configs, err := wrappedRepo.List(ctx, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(configs), 2)
}

func TestPackSizesRepositoryWithCircuitBreaker_GetCircuitBreaker(t *testing.T) {
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

	// Get circuit breaker
	returnedCB := wrappedRepo.GetCircuitBreaker()
	assert.NotNil(t, returnedCB)
	assert.Equal(t, cb, returnedCB)

	// Verify stats
	stats := returnedCB.GetStats()
	assert.Equal(t, "closed", stats.State)
}

func TestLogsRepositoryWithCircuitBreaker_CreateMany(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Use shared container with unique database name
	db := setupTestDBFromSharedContainer(t)
	defer func() {
		require.NoError(t, db.Close(ctx))
	}()

	repo := NewLogsRepository(db)
	cb := circuitbreaker.New(circuitbreaker.DefaultConfig())
	wrappedRepo := NewLogsRepositoryWithCircuitBreaker(repo, cb)

	entries := []*LogEntryDocument{
		{
			Level:     "info",
			Message:   "Entry 1",
			RequestID: "req-1",
			Timestamp: time.Now(),
		},
		{
			Level:     "error",
			Message:   "Entry 2",
			RequestID: "req-2",
			Timestamp: time.Now(),
		},
	}

	err := wrappedRepo.CreateMany(ctx, entries)
	assert.NoError(t, err)
}

func TestLogsRepositoryWithCircuitBreaker_Query(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

		// Use shared container with unique database name
		db := setupTestDBFromSharedContainer(t)
		defer func() {
			require.NoError(t, db.Close(ctx))
		}()

	repo := NewLogsRepository(db)
	cb := circuitbreaker.New(circuitbreaker.DefaultConfig())
	wrappedRepo := NewLogsRepositoryWithCircuitBreaker(repo, cb)

	// Create test entries
	entry := &LogEntryDocument{
		Level:     "info",
		Message:   "Test query",
		RequestID: "query-test-id",
		Timestamp: time.Now(),
	}
	_ = wrappedRepo.Create(ctx, entry)

	// Query via circuit breaker wrapper
	opts := LogQueryOptions{
		RequestID: "query-test-id",
	}
	entries, err := wrappedRepo.Query(ctx, opts)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 1)
}

func TestLogsRepositoryWithCircuitBreaker_Count(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

		// Use shared container with unique database name
		db := setupTestDBFromSharedContainer(t)
		defer func() {
			require.NoError(t, db.Close(ctx))
		}()

	repo := NewLogsRepository(db)
	cb := circuitbreaker.New(circuitbreaker.DefaultConfig())
	wrappedRepo := NewLogsRepositoryWithCircuitBreaker(repo, cb)

	// Create test entries
	_ = wrappedRepo.Create(ctx, &LogEntryDocument{
		Level:     "info",
		Message:   "Count test 1",
		Timestamp: time.Now(),
	})
	_ = wrappedRepo.Create(ctx, &LogEntryDocument{
		Level:     "error",
		Message:   "Count test 2",
		Timestamp: time.Now(),
	})

	// Count via circuit breaker wrapper
	count, err := wrappedRepo.Count(ctx, LogQueryOptions{})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(2))

	// Count with filter
	opts := LogQueryOptions{
		Level: "info",
	}
	countFiltered, err := wrappedRepo.Count(ctx, opts)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, countFiltered, int64(1))
}

func TestLogsRepositoryWithCircuitBreaker_GetCircuitBreaker(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Use shared container with unique database name
	db := setupTestDBFromSharedContainer(t)
	defer func() {
		require.NoError(t, db.Close(ctx))
	}()

	repo := NewLogsRepository(db)
	cb := circuitbreaker.New(circuitbreaker.DefaultConfig())
	wrappedRepo := NewLogsRepositoryWithCircuitBreaker(repo, cb)

	// Get circuit breaker
	returnedCB := wrappedRepo.GetCircuitBreaker()
	assert.NotNil(t, returnedCB)
	assert.Equal(t, cb, returnedCB)

	// Verify stats
	stats := returnedCB.GetStats()
	assert.Equal(t, "closed", stats.State)
}
