// Package repository provides interfaces for repository operations.
package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PackSizesRepositoryInterface defines the interface for pack sizes repository operations.
type PackSizesRepositoryInterface interface {
	GetActive(ctx context.Context) (*PackSizeConfig, error)
	Create(ctx context.Context, sizes []int, createdBy string) (*PackSizeConfig, error)
	Update(ctx context.Context, id primitive.ObjectID, sizes []int, updatedBy string) (*PackSizeConfig, error)
	List(ctx context.Context, limit int) ([]PackSizeConfig, error)
}

// LogsRepositoryInterface defines the interface for logs repository operations.
type LogsRepositoryInterface interface {
	Create(ctx context.Context, entry *LogEntryDocument) error
	CreateMany(ctx context.Context, entries []*LogEntryDocument) error
	Query(ctx context.Context, opts LogQueryOptions) ([]*LogEntryDocument, error)
	Count(ctx context.Context, opts LogQueryOptions) (int64, error)
}
