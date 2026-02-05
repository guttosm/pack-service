// Package repository provides data access layer for MongoDB.
package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoConfig holds MongoDB connection pool configuration.
type MongoConfig struct {
	// MaxPoolSize is the maximum number of connections in the pool.
	MaxPoolSize uint64
	// MinPoolSize is the minimum number of connections to keep in the pool.
	MinPoolSize uint64
	// MaxConnIdleTime is how long a connection can remain idle before being closed.
	MaxConnIdleTime time.Duration
	// ConnectTimeout is the timeout for establishing a connection.
	ConnectTimeout time.Duration
	// ServerSelectionTimeout is how long to wait for server selection.
	ServerSelectionTimeout time.Duration
	// SocketTimeout is the timeout for socket read/write operations.
	SocketTimeout time.Duration
	// EnableCompression enables wire protocol compression.
	EnableCompression bool
}

// DefaultMongoConfig returns production-optimized MongoDB configuration.
func DefaultMongoConfig() MongoConfig {
	return MongoConfig{
		MaxPoolSize:            50,
		MinPoolSize:            10,
		MaxConnIdleTime:        10 * time.Minute,
		ConnectTimeout:         10 * time.Second,
		ServerSelectionTimeout: 5 * time.Second,
		SocketTimeout:          30 * time.Second,
		EnableCompression:      true,
	}
}

// MongoDB provides MongoDB client and database access.
type MongoDB struct {
	Client      *mongo.Client
	Database    *mongo.Database
	PackSizes   *mongo.Collection
	Logs        *mongo.Collection
	Users       *mongo.Collection
	Roles       *mongo.Collection
	Permissions *mongo.Collection
	Tokens      *mongo.Collection
}

// NewMongoDB creates a new MongoDB connection with default configuration.
func NewMongoDB(uri, databaseName string) (*MongoDB, error) {
	return NewMongoDBWithConfig(uri, databaseName, DefaultMongoConfig())
}

// NewMongoDBWithConfig creates a new MongoDB connection with custom configuration.
func NewMongoDBWithConfig(uri, databaseName string, cfg MongoConfig) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	// Build client options with connection pool configuration
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize).
		SetMaxConnIdleTime(cfg.MaxConnIdleTime).
		SetConnectTimeout(cfg.ConnectTimeout).
		SetServerSelectionTimeout(cfg.ServerSelectionTimeout).
		SetSocketTimeout(cfg.SocketTimeout)

	// Enable compression if configured
	if cfg.EnableCompression {
		clientOptions.SetCompressors([]string{"zstd", "snappy", "zlib"})
	}

	// Set read/write concerns for better performance
	clientOptions.SetRetryWrites(true)
	clientOptions.SetRetryReads(true)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	db := client.Database(databaseName)
	mongoDB := &MongoDB{
		Client:      client,
		Database:    db,
		PackSizes:   db.Collection("pack_sizes"),
		Logs:        db.Collection("logs"),
		Users:       db.Collection("users"),
		Roles:       db.Collection("roles"),
		Permissions: db.Collection("permissions"),
		Tokens:      db.Collection("tokens"),
	}

	// Create indexes
	if err := mongoDB.createIndexes(ctx); err != nil {
		return nil, err
	}

	return mongoDB, nil
}

// createIndexes creates necessary indexes for collections.
func (m *MongoDB) createIndexes(ctx context.Context) error {
	// Pack sizes index: active pack sizes
	packSizesIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"active": 1},
		Options: options.Index().SetUnique(false),
	}
	_, err := m.PackSizes.Indexes().CreateOne(ctx, packSizesIndex)
	if err != nil {
		return err
	}

	// Logs index: TTL index for automatic cleanup (will be updated by SetLogsTTL)
	// Don't create here to avoid conflicts - SetLogsTTL will handle it

	// Logs index: request_id for querying
	requestIDIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"request_id": 1},
		Options: options.Index().SetUnique(false),
	}
	_, _ = m.Logs.Indexes().CreateOne(ctx, requestIDIndex)
	// Ignore errors if index already exists (index might already exist, that's okay)

	// Users indexes
	emailIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"email": 1},
		Options: options.Index().SetUnique(true),
	}
	_, _ = m.Users.Indexes().CreateOne(ctx, emailIndex)

	// Roles indexes
	roleNameIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"name": 1},
		Options: options.Index().SetUnique(true),
	}
	_, _ = m.Roles.Indexes().CreateOne(ctx, roleNameIndex)

	// Permissions indexes
	permissionResourceActionIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"resource": 1, "action": 1},
		Options: options.Index().SetUnique(true),
	}
	_, _ = m.Permissions.Indexes().CreateOne(ctx, permissionResourceActionIndex)

	// Tokens indexes
	tokenIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"token": 1},
		Options: options.Index().SetUnique(true),
	}
	_, _ = m.Tokens.Indexes().CreateOne(ctx, tokenIndex)

	userIDTypeIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"user_id": 1, "type": 1},
		Options: options.Index().SetUnique(false),
	}
	_, _ = m.Tokens.Indexes().CreateOne(ctx, userIDTypeIndex)

	// TTL index for tokens (auto-delete expired tokens)
	tokenTTLIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"expires_at": 1},
		Options: options.Index().SetExpireAfterSeconds(0), // 0 means use expires_at field
	}
	_, _ = m.Tokens.Indexes().CreateOne(ctx, tokenTTLIndex)

	return nil
}

// SetLogsTTL updates the TTL index for logs collection.
func (m *MongoDB) SetLogsTTL(ctx context.Context, ttlDays int) error {
	// Try to drop existing TTL index if it exists (ignore errors - index might not exist)
	_, _ = m.Logs.Indexes().DropOne(ctx, "timestamp_1")

	// Create new TTL index
	ttlSeconds := int32(ttlDays * 24 * 60 * 60)
	ttlIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"timestamp": 1},
		Options: options.Index().SetExpireAfterSeconds(ttlSeconds),
	}
	_, err := m.Logs.Indexes().CreateOne(ctx, ttlIndex)
	// Ignore errors if index already exists with different options
	if err != nil {
		errMsg := err.Error()
		if errMsg != "" && (errMsg == "index already exists" || errMsg == "IndexOptionsConflict") {
			return nil // Index exists, that's fine
		}
	}
	return err
}

// Close closes the MongoDB connection.
func (m *MongoDB) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}

// HealthCheck verifies the MongoDB connection is healthy.
func (m *MongoDB) HealthCheck(ctx context.Context) error {
	// Use a short timeout for health checks
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return m.Client.Ping(ctx, nil)
}
