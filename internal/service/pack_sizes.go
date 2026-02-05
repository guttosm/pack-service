package service

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/guttosm/pack-service/internal/repository"
)

// ErrRepositoryNotConfigured is returned when the repository is not configured.
var ErrRepositoryNotConfigured = errors.New("repository not configured")

// PackSizesService provides pack sizes-related operations.
type PackSizesService interface {
	GetActive(ctx context.Context) (*repository.PackSizeConfig, error)
	Create(ctx context.Context, sizes []int, createdBy string) (*repository.PackSizeConfig, error)
	Update(ctx context.Context, id primitive.ObjectID, sizes []int, updatedBy string) (*repository.PackSizeConfig, error)
	List(ctx context.Context, limit int) ([]repository.PackSizeConfig, error)
}

// PackSizesServiceImpl implements PackSizesService.
type PackSizesServiceImpl struct {
	packSizesRepo repository.PackSizesRepositoryInterface
}

// NewPackSizesService creates a new pack sizes service.
func NewPackSizesService(packSizesRepo repository.PackSizesRepositoryInterface) PackSizesService {
	if packSizesRepo == nil {
		return &PackSizesServiceImpl{}
	}
	return &PackSizesServiceImpl{
		packSizesRepo: packSizesRepo,
	}
}

func (s *PackSizesServiceImpl) GetActive(ctx context.Context) (*repository.PackSizeConfig, error) {
	if s.packSizesRepo == nil {
		return nil, ErrRepositoryNotConfigured
	}
	return s.packSizesRepo.GetActive(ctx)
}

func (s *PackSizesServiceImpl) Create(ctx context.Context, sizes []int, createdBy string) (*repository.PackSizeConfig, error) {
	if s.packSizesRepo == nil {
		return nil, ErrRepositoryNotConfigured
	}
	return s.packSizesRepo.Create(ctx, sizes, createdBy)
}

func (s *PackSizesServiceImpl) Update(ctx context.Context, id primitive.ObjectID, sizes []int, updatedBy string) (*repository.PackSizeConfig, error) {
	if s.packSizesRepo == nil {
		return nil, ErrRepositoryNotConfigured
	}
	return s.packSizesRepo.Update(ctx, id, sizes, updatedBy)
}

func (s *PackSizesServiceImpl) List(ctx context.Context, limit int) ([]repository.PackSizeConfig, error) {
	if s.packSizesRepo == nil {
		return nil, ErrRepositoryNotConfigured
	}
	return s.packSizesRepo.List(ctx, limit)
}
