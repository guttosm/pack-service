//go:build integration

package repository

import (
	"context"
	"os"
	"testing"

	"github.com/guttosm/pack-service/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestMain sets up a shared MongoDB container for all integration tests in this package.
// This significantly reduces test execution time by reusing a single container instead of
// creating one for each test (~30-40s per container → ~30-40s total for all tests).
func TestMain(m *testing.M) {
	os.Exit(testutil.SetupTestMainWithMongoDB(context.Background(), m))
}

// getSharedContainerURI returns the URI of the shared MongoDB container.
func getSharedContainerURI() string {
	return testutil.GetSharedContainerURI()
}

// sanitizeDBName sanitizes a test name to be a valid MongoDB database name.
func sanitizeDBName(testName string) string {
	return testutil.SanitizeDBName(testName)
}

// setupTestDBFromSharedContainer creates a MongoDB connection using the shared container
// with a unique database name for test isolation.
func setupTestDBFromSharedContainer(t *testing.T) *MongoDB {
	dbName := sanitizeDBName(t.Name())
	uri := getSharedContainerURI()
	db, err := NewMongoDB(uri, dbName)
	require.NoError(t, err)
	return db
}
