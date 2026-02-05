// Package repository provides circuit breaker wrappers for MongoDB operations.
package repository

import (
	"context"

	"github.com/guttosm/pack-service/internal/circuitbreaker"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PackSizesRepositoryWithCircuitBreaker wraps PackSizesRepository with circuit breaker protection.
type PackSizesRepositoryWithCircuitBreaker struct {
	repo           *PackSizesRepository
	circuitBreaker *circuitbreaker.CircuitBreaker
}

// NewPackSizesRepositoryWithCircuitBreaker creates a new repository wrapper with circuit breaker.
func NewPackSizesRepositoryWithCircuitBreaker(repo *PackSizesRepository, cb *circuitbreaker.CircuitBreaker) *PackSizesRepositoryWithCircuitBreaker {
	return &PackSizesRepositoryWithCircuitBreaker{
		repo:           repo,
		circuitBreaker: cb,
	}
}

// GetActive returns the active pack size configuration with circuit breaker protection.
func (r *PackSizesRepositoryWithCircuitBreaker) GetActive(ctx context.Context) (*PackSizeConfig, error) {
	var result *PackSizeConfig
	err := r.circuitBreaker.Execute(ctx, func() error {
		var cbErr error
		result, cbErr = r.repo.GetActive(ctx)
		return cbErr
	})
	if err == circuitbreaker.ErrCircuitOpen {
		// Circuit is open - return nil to use default pack sizes
		return nil, nil
	}
	return result, err
}

// Create creates a new pack size configuration with circuit breaker protection.
func (r *PackSizesRepositoryWithCircuitBreaker) Create(ctx context.Context, sizes []int, createdBy string) (*PackSizeConfig, error) {
	var result *PackSizeConfig
	err := r.circuitBreaker.Execute(ctx, func() error {
		var cbErr error
		result, cbErr = r.repo.Create(ctx, sizes, createdBy)
		return cbErr
	})
	return result, err
}

// Update updates an existing pack size configuration with circuit breaker protection.
func (r *PackSizesRepositoryWithCircuitBreaker) Update(ctx context.Context, id primitive.ObjectID, sizes []int, updatedBy string) (*PackSizeConfig, error) {
	var result *PackSizeConfig
	err := r.circuitBreaker.Execute(ctx, func() error {
		var cbErr error
		result, cbErr = r.repo.Update(ctx, id, sizes, updatedBy)
		return cbErr
	})
	return result, err
}

// List returns all pack size configurations with circuit breaker protection.
func (r *PackSizesRepositoryWithCircuitBreaker) List(ctx context.Context, limit int) ([]PackSizeConfig, error) {
	var result []PackSizeConfig
	err := r.circuitBreaker.Execute(ctx, func() error {
		var cbErr error
		result, cbErr = r.repo.List(ctx, limit)
		return cbErr
	})
	return result, err
}

// GetCircuitBreaker returns the underlying circuit breaker for monitoring.
func (r *PackSizesRepositoryWithCircuitBreaker) GetCircuitBreaker() *circuitbreaker.CircuitBreaker {
	return r.circuitBreaker
}

// LogsRepositoryWithCircuitBreaker wraps LogsRepository with circuit breaker protection.
type LogsRepositoryWithCircuitBreaker struct {
	repo           *LogsRepository
	circuitBreaker *circuitbreaker.CircuitBreaker
}

// NewLogsRepositoryWithCircuitBreaker creates a new repository wrapper with circuit breaker.
func NewLogsRepositoryWithCircuitBreaker(repo *LogsRepository, cb *circuitbreaker.CircuitBreaker) *LogsRepositoryWithCircuitBreaker {
	return &LogsRepositoryWithCircuitBreaker{
		repo:           repo,
		circuitBreaker: cb,
	}
}

// Create stores a single log entry with circuit breaker protection.
// If circuit is open, silently fails (logging is non-critical).
func (r *LogsRepositoryWithCircuitBreaker) Create(ctx context.Context, entry *LogEntryDocument) error {
	err := r.circuitBreaker.Execute(ctx, func() error {
		return r.repo.Create(ctx, entry)
	})
	if err == circuitbreaker.ErrCircuitOpen {
		// Circuit is open - silently fail (logging is non-critical)
		return nil
	}
	return err
}

// CreateMany stores multiple log entries with circuit breaker protection.
// If circuit is open, silently fails (logging is non-critical).
func (r *LogsRepositoryWithCircuitBreaker) CreateMany(ctx context.Context, entries []*LogEntryDocument) error {
	err := r.circuitBreaker.Execute(ctx, func() error {
		return r.repo.CreateMany(ctx, entries)
	})
	if err == circuitbreaker.ErrCircuitOpen {
		// Circuit is open - silently fail (logging is non-critical)
		return nil
	}
	return err
}

// Query retrieves log entries with circuit breaker protection.
func (r *LogsRepositoryWithCircuitBreaker) Query(ctx context.Context, opts LogQueryOptions) ([]*LogEntryDocument, error) {
	var result []*LogEntryDocument
	err := r.circuitBreaker.Execute(ctx, func() error {
		var cbErr error
		result, cbErr = r.repo.Query(ctx, opts)
		return cbErr
	})
	return result, err
}

// Count returns the count of log entries with circuit breaker protection.
func (r *LogsRepositoryWithCircuitBreaker) Count(ctx context.Context, opts LogQueryOptions) (int64, error) {
	var result int64
	err := r.circuitBreaker.Execute(ctx, func() error {
		var cbErr error
		result, cbErr = r.repo.Count(ctx, opts)
		return cbErr
	})
	return result, err
}

// GetCircuitBreaker returns the underlying circuit breaker for monitoring.
func (r *LogsRepositoryWithCircuitBreaker) GetCircuitBreaker() *circuitbreaker.CircuitBreaker {
	return r.circuitBreaker
}
