//go:build integration

// Package testutil provides test utilities and testcontainers setup for integration tests.
package testutil

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

// MongoDBContainer wraps a MongoDB testcontainer.
type MongoDBContainer struct {
	Container testcontainers.Container
	URI      string
}

// SetupMongoDB creates and starts a MongoDB testcontainer.
// Returns the container, connection URI, and MongoDB client.
// For better performance, consider using GetSharedMongoDB() from test_container.go with TestMain for container reuse.
func SetupMongoDB(ctx context.Context) (*MongoDBContainer, error) {
	// Use standard MongoDB image (alpine variant not available for 7.0)
	mongoContainer, err := mongodb.Run(ctx, "mongo:7.0")
	if err != nil {
		return nil, fmt.Errorf("failed to start MongoDB container: %w", err)
	}

	uri, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		mongoContainer.Terminate(ctx)
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	return &MongoDBContainer{
		Container: mongoContainer,
		URI:       uri,
	}, nil
}

// Cleanup terminates the MongoDB container.
func (m *MongoDBContainer) Cleanup(ctx context.Context) error {
	if m.Container != nil {
		if err := m.Container.Terminate(ctx); err != nil {
			return fmt.Errorf("failed to terminate container: %w", err)
		}
	}
	return nil
}
