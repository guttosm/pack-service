//go:build integration
// +build integration

package testutil

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

var (
	sharedContainer     *MongoDBContainer
	sharedContainerErr  error
	sharedContainerOnce sync.Once
	sharedContainerMu   sync.RWMutex
)

// GetSharedMongoDB returns a shared MongoDB container for use across tests in a package.
// The container is created once and reused for all tests.
// Call CleanupSharedMongoDB in TestMain to clean up.
func GetSharedMongoDB(ctx context.Context) (*MongoDBContainer, error) {
	sharedContainerOnce.Do(func() {
		sharedContainerMu.Lock()
		defer sharedContainerMu.Unlock()

		sharedContainer, sharedContainerErr = SetupMongoDB(ctx)
	})

	sharedContainerMu.RLock()
	defer sharedContainerMu.RUnlock()

	if sharedContainerErr != nil {
		return nil, sharedContainerErr
	}
	return sharedContainer, nil
}

// CleanupSharedMongoDB cleans up the shared MongoDB container.
// Call this in TestMain after m.Run().
func CleanupSharedMongoDB(ctx context.Context) error {
	sharedContainerMu.Lock()
	defer sharedContainerMu.Unlock()
	
	if sharedContainer != nil {
		return sharedContainer.Cleanup(ctx)
	}
	return nil
}

// SetupTestMainWithMongoDB is a helper for TestMain that sets up and tears down a shared MongoDB container.
// Usage:
//
//	func TestMain(m *testing.M) {
//		os.Exit(testutil.SetupTestMainWithMongoDB(context.Background(), m))
//	}
func SetupTestMainWithMongoDB(ctx context.Context, m *testing.M) int {
	_, err := GetSharedMongoDB(ctx)
	if err != nil {
		panic(err)
	}
	
	code := m.Run()
	
	if err := CleanupSharedMongoDB(ctx); err != nil {
		// Log error but don't fail - container will be cleaned up by Docker
		_, _ = os.Stderr.WriteString("Warning: failed to cleanup shared MongoDB container: " + err.Error() + "\n")
	}
	
	return code
}

// GetSharedContainerURI returns the URI of the shared MongoDB container.
// Panics if the container is not initialized.
func GetSharedContainerURI() string {
	sharedContainerMu.RLock()
	defer sharedContainerMu.RUnlock()
	
	if sharedContainer == nil {
		panic("shared MongoDB container not initialized - call GetSharedMongoDB first")
	}
	return sharedContainer.URI
}

// SanitizeDBName sanitizes a test name to be a valid MongoDB database name.
// It replaces path separators with underscores, truncates to 50 characters,
// and appends a timestamp suffix for uniqueness.
func SanitizeDBName(testName string) string {
	sanitized := ""
	for _, r := range testName {
		if r == '/' || r == '\\' {
			sanitized += "_"
		} else {
			sanitized += string(r)
		}
	}
	
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}
	
	return sanitized + "_" + fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
}
