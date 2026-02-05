package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/guttosm/pack-service/internal/service"
)

func TestRoleService_FindByID(t *testing.T) {
	testRoleID := primitive.NewObjectID()

	tests := []struct {
		name          string
		roleID        primitive.ObjectID
		setupMock     func(*mocks.MockRoleRepositoryInterface)
		expectedError error
		expectedRole  *model.Role
	}{
		{
			name:   "successful find",
			roleID: testRoleID,
			setupMock: func(m *mocks.MockRoleRepositoryInterface) {
				role := &model.Role{
					ID:          testRoleID,
					Name:        "admin",
					Description: "Administrator role",
					Permissions: []string{"perm1", "perm2"},
					Active:      true,
				}
				m.On("FindByID", mock.Anything, testRoleID).Return(role, nil)
			},
			expectedError: nil,
			expectedRole: &model.Role{
				ID:          testRoleID,
				Name:        "admin",
				Description: "Administrator role",
				Permissions: []string{"perm1", "perm2"},
				Active:      true,
			},
		},
		{
			name:   "role not found",
			roleID: primitive.NewObjectID(),
			setupMock: func(m *mocks.MockRoleRepositoryInterface) {
				m.On("FindByID", mock.Anything, mock.AnythingOfType("primitive.ObjectID")).Return(nil, nil)
			},
			expectedError: nil,
			expectedRole:  nil,
		},
		{
			name:   "repository error",
			roleID: primitive.NewObjectID(),
			setupMock: func(m *mocks.MockRoleRepositoryInterface) {
				m.On("FindByID", mock.Anything, mock.AnythingOfType("primitive.ObjectID")).Return(nil, errors.New("database error"))
			},
			expectedError: errors.New("database error"),
			expectedRole:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockRoleRepositoryInterface)
			tt.setupMock(mockRepo)

			svc := service.NewRoleService(mockRepo)
			role, err := svc.FindByID(context.Background(), tt.roleID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.expectedRole != nil {
				assert.NotNil(t, role)
				assert.Equal(t, tt.expectedRole.Name, role.Name)
				assert.Equal(t, tt.expectedRole.Description, role.Description)
			} else if tt.expectedError == nil {
				assert.Nil(t, role)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRoleService_FindByID_NilRepository(t *testing.T) {
	svc := service.NewRoleService(nil)
	role, err := svc.FindByID(context.Background(), primitive.NewObjectID())

	assert.Error(t, err)
	assert.Equal(t, service.ErrRepositoryNotConfigured, err)
	assert.Nil(t, role)
}

func TestRoleService_FindByIDs(t *testing.T) {
	roleID1 := primitive.NewObjectID()
	roleID2 := primitive.NewObjectID()

	tests := []struct {
		name          string
		ids           []string
		setupMock     func(*mocks.MockRoleRepositoryInterface)
		expectedError error
		expectedCount int
	}{
		{
			name: "successful find multiple",
			ids:  []string{roleID1.Hex(), roleID2.Hex()},
			setupMock: func(m *mocks.MockRoleRepositoryInterface) {
				roles := []*model.Role{
					{ID: roleID1, Name: "admin", Active: true},
					{ID: roleID2, Name: "user", Active: true},
				}
				m.On("FindByIDs", mock.Anything, []string{roleID1.Hex(), roleID2.Hex()}).Return(roles, nil)
			},
			expectedError: nil,
			expectedCount: 2,
		},
		{
			name: "empty ids",
			ids:  []string{},
			setupMock: func(m *mocks.MockRoleRepositoryInterface) {
				m.On("FindByIDs", mock.Anything, []string{}).Return([]*model.Role{}, nil)
			},
			expectedError: nil,
			expectedCount: 0,
		},
		{
			name: "partial match",
			ids:  []string{roleID1.Hex(), "nonexistent"},
			setupMock: func(m *mocks.MockRoleRepositoryInterface) {
				roles := []*model.Role{
					{ID: roleID1, Name: "admin", Active: true},
				}
				m.On("FindByIDs", mock.Anything, []string{roleID1.Hex(), "nonexistent"}).Return(roles, nil)
			},
			expectedError: nil,
			expectedCount: 1,
		},
		{
			name: "repository error",
			ids:  []string{roleID1.Hex()},
			setupMock: func(m *mocks.MockRoleRepositoryInterface) {
				m.On("FindByIDs", mock.Anything, []string{roleID1.Hex()}).Return(nil, errors.New("connection error"))
			},
			expectedError: errors.New("connection error"),
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockRoleRepositoryInterface)
			tt.setupMock(mockRepo)

			svc := service.NewRoleService(mockRepo)
			roles, err := svc.FindByIDs(context.Background(), tt.ids)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, roles)
			} else {
				assert.NoError(t, err)
				assert.Len(t, roles, tt.expectedCount)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestRoleService_FindByIDs_NilRepository(t *testing.T) {
	svc := service.NewRoleService(nil)
	roles, err := svc.FindByIDs(context.Background(), []string{"id1", "id2"})

	assert.Error(t, err)
	assert.Equal(t, service.ErrRepositoryNotConfigured, err)
	assert.Nil(t, roles)
}
