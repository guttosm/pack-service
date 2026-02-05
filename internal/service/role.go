package service

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/repository"
)

// RoleService provides role-related operations.
type RoleService interface {
	FindByID(ctx context.Context, id primitive.ObjectID) (*model.Role, error)
	FindByIDs(ctx context.Context, ids []string) ([]*model.Role, error)
}

// RoleServiceImpl implements RoleService.
type RoleServiceImpl struct {
	roleRepo repository.RoleRepositoryInterface
}

// NewRoleService creates a new role service.
func NewRoleService(roleRepo repository.RoleRepositoryInterface) RoleService {
	return &RoleServiceImpl{
		roleRepo: roleRepo,
	}
}

func (s *RoleServiceImpl) FindByID(ctx context.Context, id primitive.ObjectID) (*model.Role, error) {
	if s.roleRepo == nil {
		return nil, ErrRepositoryNotConfigured
	}
	return s.roleRepo.FindByID(ctx, id)
}

func (s *RoleServiceImpl) FindByIDs(ctx context.Context, ids []string) ([]*model.Role, error) {
	if s.roleRepo == nil {
		return nil, ErrRepositoryNotConfigured
	}
	return s.roleRepo.FindByIDs(ctx, ids)
}
