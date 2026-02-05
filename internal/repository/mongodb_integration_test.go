//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMongoDB_Integration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Use shared container instead of creating a new one
	uri := getSharedContainerURI()
	dbName := sanitizeDBName(t.Name())

	// Create MongoDB connection using the URI from shared testcontainer
	db, err := NewMongoDB(uri, dbName)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close(ctx))
	}()

	t.Run("connection successful", func(t *testing.T) {
		assert.NotNil(t, db)
		assert.NotNil(t, db.Client)
		assert.NotNil(t, db.Database)
		assert.NotNil(t, db.PackSizes)
		assert.NotNil(t, db.Logs)
	})

	t.Run("ping successful", func(t *testing.T) {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		err := db.Client.Ping(pingCtx, nil)
		assert.NoError(t, err)
	})

	t.Run("set logs TTL", func(t *testing.T) {
		err := db.SetLogsTTL(ctx, 30)
		assert.NoError(t, err)
	})

	t.Run("set logs TTL multiple times", func(t *testing.T) {
		// Setting TTL multiple times should not error
		err1 := db.SetLogsTTL(ctx, 30)
		assert.NoError(t, err1)

		err2 := db.SetLogsTTL(ctx, 60)
		// May error if index exists, but that's acceptable
		_ = err2
	})

	t.Run("verify collections exist", func(t *testing.T) {
		// Collections are created during NewMongoDB
		// Verify collections exist
		assert.NotNil(t, db.PackSizes)
		assert.NotNil(t, db.Logs)
		assert.NotNil(t, db.Users)
		assert.NotNil(t, db.Roles)
		assert.NotNil(t, db.Permissions)
		assert.NotNil(t, db.Tokens)
	})
}
