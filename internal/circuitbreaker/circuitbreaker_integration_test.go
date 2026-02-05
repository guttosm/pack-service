//go:build integration

package circuitbreaker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/guttosm/pack-service/internal/circuitbreaker"
	"github.com/guttosm/pack-service/internal/repository"
	"github.com/guttosm/pack-service/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreakerWithMongoDB_Integration(t *testing.T) {
	ctx := context.Background()

	mongoContainer, err := testutil.SetupMongoDB(ctx)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, mongoContainer.Cleanup(ctx))
	}()

	t.Run("circuit breaker protects pack sizes repository", func(t *testing.T) {
		db, err := repository.NewMongoDB(mongoContainer.URI, "test_pack_service")
		require.NoError(t, err)
		defer func() {
			_ = db.Close(ctx)
		}()

		repo := repository.NewPackSizesRepository(db)
		cb := circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold: 2,
			SuccessThreshold: 1,
			Timeout:          100 * time.Millisecond,
			Name:             "test-pack-sizes",
		})
		wrappedRepo := repository.NewPackSizesRepositoryWithCircuitBreaker(repo, cb)

		// Successful operations
		_, err = wrappedRepo.Create(ctx, []int{100, 200}, "test")
		require.NoError(t, err)

		active, err := wrappedRepo.GetActive(ctx)
		require.NoError(t, err)
		assert.NotNil(t, active)

		stats := cb.GetStats()
		assert.Equal(t, circuitbreaker.StateClosed, cb.State())
		assert.True(t, stats.IsHealthy)
	})

	t.Run("circuit breaker protects logs repository", func(t *testing.T) {
		db, err := repository.NewMongoDB(mongoContainer.URI, "test_pack_service")
		require.NoError(t, err)
		defer func() {
			_ = db.Close(ctx)
		}()

		repo := repository.NewLogsRepository(db)
		cb := circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold: 2,
			SuccessThreshold: 1,
			Timeout:          100 * time.Millisecond,
			Name:             "test-logs",
		})
		wrappedRepo := repository.NewLogsRepositoryWithCircuitBreaker(repo, cb)

		entry := &repository.LogEntryDocument{
			Level:   "info",
			Message: "Test",
		}

		// Successful operation
		err = wrappedRepo.Create(ctx, entry)
		assert.NoError(t, err)

		assert.Equal(t, circuitbreaker.StateClosed, cb.State())
		assert.True(t, cb.GetStats().IsHealthy)
	})

	t.Run("circuit breaker opens on failures", func(t *testing.T) {
		cb := circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold: 2,
			SuccessThreshold: 1,
			Timeout:          100 * time.Millisecond,
			Name:             "test-failures",
		})

		// Simulate failures
		for i := 0; i < 2; i++ {
			err := cb.Execute(ctx, func() error {
				return errors.New("simulated error")
			})
			assert.Error(t, err)
		}

		assert.Equal(t, circuitbreaker.StateOpen, cb.State())
		assert.True(t, cb.IsOpen())

		err := cb.Execute(ctx, func() error {
			return nil // This won't be called
		})
		assert.Equal(t, circuitbreaker.ErrCircuitOpen, err)
	})

	t.Run("circuit breaker recovers after timeout", func(t *testing.T) {
		cb := circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold: 1,
			SuccessThreshold: 1,
			Timeout:          50 * time.Millisecond,
			Name:             "test-recovery",
		})

		// Open the circuit
		_ = cb.Execute(ctx, func() error {
			return errors.New("error")
		})
		assert.Equal(t, circuitbreaker.StateOpen, cb.State())

		// Wait for timeout
		time.Sleep(60 * time.Millisecond)

		// Should transition to half-open
		err := cb.Execute(ctx, func() error {
			return nil // Success
		})
		assert.NoError(t, err)
		assert.Equal(t, circuitbreaker.StateClosed, cb.State())
	})
}
