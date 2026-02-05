package service

import (
	"context"
	"time"

	"github.com/guttosm/pack-service/internal/repository"
)

// PermissionService provides permission-related operations.
type PermissionService interface {
	GetPermissionIDByResourceAndAction(ctx context.Context, resource, action string) string
}

// PermissionServiceImpl implements PermissionService.
type PermissionServiceImpl struct {
	permissionRepo repository.PermissionRepositoryInterface
}

// NewPermissionService creates a new permission service.
func NewPermissionService(permissionRepo repository.PermissionRepositoryInterface) PermissionService {
	return &PermissionServiceImpl{
		permissionRepo: permissionRepo,
	}
}

func (s *PermissionServiceImpl) GetPermissionIDByResourceAndAction(ctx context.Context, resource, action string) string {
	if s.permissionRepo == nil {
		return ""
	}

	lookupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	perm, err := s.permissionRepo.FindByResourceAndAction(lookupCtx, resource, action)
	if err != nil || perm == nil {
		return ""
	}

	return perm.ID.Hex()
}
