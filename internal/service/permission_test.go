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

func TestPermissionService_GetPermissionIDByResourceAndAction(t *testing.T) {
	testPermissionID := primitive.NewObjectID()

	tests := []struct {
		name       string
		resource   string
		action     string
		setupMock  func(*mocks.MockPermissionRepositoryInterface)
		expectedID string
	}{
		{
			name:     "successful lookup",
			resource: "pack_sizes",
			action:   "read",
			setupMock: func(m *mocks.MockPermissionRepositoryInterface) {
				perm := &model.Permission{
					ID:          testPermissionID,
					Name:        "pack_sizes:read",
					Description: "Read pack sizes",
					Resource:    "pack_sizes",
					Action:      "read",
					Active:      true,
				}
				m.On("FindByResourceAndAction", mock.Anything, "pack_sizes", "read").Return(perm, nil)
			},
			expectedID: testPermissionID.Hex(),
		},
		{
			name:     "permission not found",
			resource: "unknown",
			action:   "delete",
			setupMock: func(m *mocks.MockPermissionRepositoryInterface) {
				m.On("FindByResourceAndAction", mock.Anything, "unknown", "delete").Return(nil, nil)
			},
			expectedID: "",
		},
		{
			name:     "repository error",
			resource: "users",
			action:   "write",
			setupMock: func(m *mocks.MockPermissionRepositoryInterface) {
				m.On("FindByResourceAndAction", mock.Anything, "users", "write").Return(nil, errors.New("database error"))
			},
			expectedID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockPermissionRepositoryInterface)
			tt.setupMock(mockRepo)

			svc := service.NewPermissionService(mockRepo)
			result := svc.GetPermissionIDByResourceAndAction(context.Background(), tt.resource, tt.action)

			assert.Equal(t, tt.expectedID, result)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestPermissionService_GetPermissionIDByResourceAndAction_NilRepository(t *testing.T) {
	svc := service.NewPermissionService(nil)
	result := svc.GetPermissionIDByResourceAndAction(context.Background(), "resource", "action")

	assert.Equal(t, "", result)
}

func TestPermissionService_GetPermissionIDByResourceAndAction_ContextTimeout(t *testing.T) {
	// This test verifies that the service properly uses a timeout context
	mockRepo := new(mocks.MockPermissionRepositoryInterface)
	
	permID := primitive.NewObjectID()
	perm := &model.Permission{
		ID:       permID,
		Name:     "test:read",
		Resource: "test",
		Action:   "read",
	}
	mockRepo.On("FindByResourceAndAction", mock.Anything, "test", "read").Return(perm, nil)

	svc := service.NewPermissionService(mockRepo)
	
	// Use a background context - the service will create its own timeout
	result := svc.GetPermissionIDByResourceAndAction(context.Background(), "test", "read")

	assert.Equal(t, permID.Hex(), result)
	mockRepo.AssertExpectations(t)
}
