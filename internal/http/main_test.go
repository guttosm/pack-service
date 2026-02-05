//go:build integration

package http

import (
	"context"
	"os"
	"testing"

	"github.com/guttosm/pack-service/internal/testutil"
)

// TestMain sets up a shared MongoDB container for all HTTP integration tests in this package.
func TestMain(m *testing.M) {
	os.Exit(testutil.SetupTestMainWithMongoDB(context.Background(), m))
}

// getSharedContainerURI returns the URI of the shared MongoDB container.
func getSharedContainerURI() string {
	return testutil.GetSharedContainerURI()
}

// sanitizeDBNameForHTTP sanitizes a test name to be a valid MongoDB database name.
func sanitizeDBNameForHTTP(testName string) string {
	return testutil.SanitizeDBName(testName)
}
