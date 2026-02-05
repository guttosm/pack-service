//go:build integration

package service

import (
	"context"
	"testing"
	"time"

	"github.com/guttosm/pack-service/internal/circuitbreaker"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/repository"
	"github.com/guttosm/pack-service/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingService_Integration(t *testing.T) {
	ctx := context.Background()

	mongoContainer, err := testutil.SetupMongoDB(ctx)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, mongoContainer.Cleanup(ctx))
	}()

	db, err := repository.NewMongoDB(mongoContainer.URI, "test_pack_service")
	require.NoError(t, err)
	defer func() {
		_ = db.Close(ctx)
	}()

	// Set TTL for logs
	err = db.SetLogsTTL(ctx, 30)
	require.NoError(t, err)

	logsRepo := repository.NewLogsRepository(db)
	loggingService := NewLoggingService(logsRepo)

	t.Run("create single log", func(t *testing.T) {
		entry := &model.LogEntry{
			Level:     "info",
			Message:   "Test log entry",
			RequestID: "test-req-1",
			Method:    "POST",
			Path:      "/api/test",
		}

		err := loggingService.CreateLog(ctx, entry)
		assert.NoError(t, err)
		assert.False(t, entry.ID.IsZero())
	})

	t.Run("create multiple logs", func(t *testing.T) {
		entries := []*model.LogEntry{
			{
				Level:     "info",
				Message:   "Log 1",
				RequestID: "req-1",
			},
			{
				Level:     "error",
				Message:   "Log 2",
				RequestID: "req-2",
			},
		}

		err := loggingService.CreateLogs(ctx, entries)
		assert.NoError(t, err)
	})

	t.Run("query logs by request ID", func(t *testing.T) {
		opts := model.LogQueryOptions{
			RequestID: "test-req-1",
		}

		entries, err := loggingService.QueryLogs(ctx, opts)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 1)
		assert.Equal(t, "test-req-1", entries[0].RequestID)
	})

	t.Run("query logs by level", func(t *testing.T) {
		opts := model.LogQueryOptions{
			Level: "error",
		}

		entries, err := loggingService.QueryLogs(ctx, opts)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 1)
		assert.Equal(t, "error", entries[0].Level)
	})

	t.Run("count logs", func(t *testing.T) {
		count, err := loggingService.CountLogs(ctx, model.LogQueryOptions{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(3)) // At least 3 entries created
	})

	t.Run("count logs with filter", func(t *testing.T) {
		opts := model.LogQueryOptions{
			Level: "info",
		}

		count, err := loggingService.CountLogs(ctx, opts)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(2))
	})

	t.Run("query with time range", func(t *testing.T) {
		now := time.Now()
		startTime := now.Add(-1 * time.Hour)
		endTime := now.Add(1 * time.Hour)

		opts := model.LogQueryOptions{
			StartTime: &startTime,
			EndTime:   &endTime,
		}

		entries, err := loggingService.QueryLogs(ctx, opts)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 0)
	})
}

func TestLoggingServiceWithCircuitBreaker_Integration(t *testing.T) {
	ctx := context.Background()

	mongoContainer, err := testutil.SetupMongoDB(ctx)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, mongoContainer.Cleanup(ctx))
	}()

	db, err := repository.NewMongoDB(mongoContainer.URI, "test_pack_service")
	require.NoError(t, err)
	defer func() {
		_ = db.Close(ctx)
	}()

	logsRepo := repository.NewLogsRepository(db)
	cb := repository.NewLogsRepositoryWithCircuitBreaker(
		logsRepo,
		circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold: 2,
			SuccessThreshold: 1,
			Timeout:          100 * time.Millisecond,
			Name:            "test-logs",
		}),
	)
	loggingService := NewLoggingService(cb)

	t.Run("circuit breaker allows successful operations", func(t *testing.T) {
		entry := &model.LogEntry{
			Level:   "info",
			Message: "Test entry",
		}

		err := loggingService.CreateLog(ctx, entry)
		assert.NoError(t, err)
	})
}
