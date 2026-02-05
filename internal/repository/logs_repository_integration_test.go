//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/guttosm/pack-service/internal/circuitbreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestLogsRepository_Integration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Use shared container with unique database name
	db := setupTestDBFromSharedContainer(t)
	defer func() {
		require.NoError(t, db.Close(ctx))
	}()

	err := db.SetLogsTTL(ctx, 30)
	require.NoError(t, err)

	repo := NewLogsRepository(db)

	t.Run("create log entry", func(t *testing.T) {
		entry := &LogEntryDocument{
			ID:         primitive.NewObjectID(),
			Timestamp:  time.Now(),
			Level:      "info",
			Message:    "Test log entry",
			RequestID:  "test-request-id",
			Method:     "POST",
			Path:       "/api/test",
			StatusCode: 200,
			Duration:   100,
			IP:         "127.0.0.1",
			UserAgent:  "test-agent",
		}

		err := repo.Create(ctx, entry)
		assert.NoError(t, err)
		assert.False(t, entry.ID.IsZero())
	})

	t.Run("create many log entries", func(t *testing.T) {
		entries := []*LogEntryDocument{
			{Level: "info", Message: "Entry 1", RequestID: "req-1"},
			{Level: "error", Message: "Entry 2", RequestID: "req-2"},
			{Level: "warn", Message: "Entry 3", RequestID: "req-3"},
		}

		err := repo.CreateMany(ctx, entries)
		assert.NoError(t, err)
	})

	t.Run("query by request ID", func(t *testing.T) {
		opts := LogQueryOptions{RequestID: "test-request-id"}
		entries, err := repo.Query(ctx, opts)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 1)
		assert.Equal(t, "test-request-id", entries[0].RequestID)
	})

	t.Run("query by level", func(t *testing.T) {
		opts := LogQueryOptions{Level: "error"}
		entries, err := repo.Query(ctx, opts)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 1)
		assert.Equal(t, "error", entries[0].Level)
	})

	t.Run("count logs", func(t *testing.T) {
		count, err := repo.Count(ctx, LogQueryOptions{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(4))
	})

	t.Run("count with filter", func(t *testing.T) {
		opts := LogQueryOptions{Level: "info"}
		count, err := repo.Count(ctx, opts)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(1))
	})
}

func TestLogsRepositoryWithCircuitBreaker_Integration(t *testing.T) {
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

	t.Run("circuit breaker allows successful operations", func(t *testing.T) {
		entry := &LogEntryDocument{
			Level:   "info",
			Message: "Test entry",
		}

		err := wrappedRepo.Create(ctx, entry)
		assert.NoError(t, err)
	})

	t.Run("circuit breaker stats", func(t *testing.T) {
		stats := cb.GetStats()
		assert.Equal(t, "closed", stats.State)
		assert.True(t, stats.IsHealthy)
	})
}
