//go:build !integration

package app

import (
	"errors"
	"testing"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestInitializeDefaultRolesAndPermissions(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*mocks.MockRoleRepositoryInterface, *mocks.MockPermissionRepositoryInterface)
		wantError      bool
	}{
		{
			name: "successful initialization",
			setupMocks: func(roleRepo *mocks.MockRoleRepositoryInterface, permRepo *mocks.MockPermissionRepositoryInterface) {
				permResources := []string{"packs", "packs", "users", "users", "users", "roles", "roles"}
				permActions := []string{"read", "write", "read", "write", "delete", "read", "write"}
				for i := 0; i < 7; i++ {
					permRepo.On("FindByResourceAndAction", mock.Anything, permResources[i], permActions[i]).Return(nil, nil).Once()
					permRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Permission")).Return(nil).Once()
				}
				roleRepo.On("FindByName", mock.Anything, "user").Return(nil, nil).Once()
				roleRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *model.Role) bool {
					return r.Name == "user"
				})).Return(nil).Once()
				roleRepo.On("FindByName", mock.Anything, "admin").Return(nil, nil).Once()
				roleRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *model.Role) bool {
					return r.Name == "admin"
				})).Return(nil).Once()
			},
			wantError: false,
		},
		{
			name: "permissions already exist",
			setupMocks: func(roleRepo *mocks.MockRoleRepositoryInterface, permRepo *mocks.MockPermissionRepositoryInterface) {
				permResources := []string{"packs", "packs", "users", "users", "users", "roles", "roles"}
				permActions := []string{"read", "write", "read", "write", "delete", "read", "write"}
				for i := 0; i < 7; i++ {
					existingPerm := &model.Permission{
						ID:       primitive.NewObjectID(),
						Resource: permResources[i],
						Action:   permActions[i],
					}
					permRepo.On("FindByResourceAndAction", mock.Anything, permResources[i], permActions[i]).Return(existingPerm, nil).Once()
				}
				roleRepo.On("FindByName", mock.Anything, "user").Return(nil, nil).Once()
				roleRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
				roleRepo.On("FindByName", mock.Anything, "admin").Return(nil, nil).Once()
				roleRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
			},
			wantError: false,
		},
		{
			name: "roles already exist",
			setupMocks: func(roleRepo *mocks.MockRoleRepositoryInterface, permRepo *mocks.MockPermissionRepositoryInterface) {
				permResources := []string{"packs", "packs", "users", "users", "users", "roles", "roles"}
				permActions := []string{"read", "write", "read", "write", "delete", "read", "write"}
				for i := 0; i < 7; i++ {
					permRepo.On("FindByResourceAndAction", mock.Anything, permResources[i], permActions[i]).Return(nil, nil).Once()
					permRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
				}
				existingUserRole := &model.Role{
					ID:   primitive.NewObjectID(),
					Name: "user",
				}
				existingAdminRole := &model.Role{
					ID:   primitive.NewObjectID(),
					Name: "admin",
				}
				roleRepo.On("FindByName", mock.Anything, "user").Return(existingUserRole, nil).Once()
				roleRepo.On("FindByName", mock.Anything, "admin").Return(existingAdminRole, nil).Once()
			},
			wantError: false,
		},
		{
			name: "permission creation error",
			setupMocks: func(roleRepo *mocks.MockRoleRepositoryInterface, permRepo *mocks.MockPermissionRepositoryInterface) {
				permRepo.On("FindByResourceAndAction", mock.Anything, "packs", "read").Return(nil, nil).Once()
				permRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("database error")).Once()
				for i := 1; i < 7; i++ {
					permRepo.On("FindByResourceAndAction", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil).Maybe()
					permRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
				}
				roleRepo.On("FindByName", mock.Anything, mock.Anything).Return(nil, nil).Maybe()
				roleRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantError: false,
		},
		{
			name: "role creation error",
			setupMocks: func(roleRepo *mocks.MockRoleRepositoryInterface, permRepo *mocks.MockPermissionRepositoryInterface) {
				permResources := []string{"packs", "packs", "users", "users", "users", "roles", "roles"}
				permActions := []string{"read", "write", "read", "write", "delete", "read", "write"}
				for i := 0; i < 7; i++ {
					permRepo.On("FindByResourceAndAction", mock.Anything, permResources[i], permActions[i]).Return(nil, nil).Once()
					permRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
				}
				roleRepo.On("FindByName", mock.Anything, "user").Return(nil, nil).Once()
				roleRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("database error")).Once()
				roleRepo.On("FindByName", mock.Anything, "admin").Return(nil, nil).Maybe()
				roleRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roleRepo := mocks.NewMockRoleRepositoryInterface(t)
			permRepo := mocks.NewMockPermissionRepositoryInterface(t)
			tt.setupMocks(roleRepo, permRepo)

			err := initializeDefaultRolesAndPermissions(roleRepo, permRepo)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			roleRepo.AssertExpectations(t)
			permRepo.AssertExpectations(t)
		})
	}
}
