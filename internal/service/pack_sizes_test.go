package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/guttosm/pack-service/internal/repository"
	"github.com/guttosm/pack-service/internal/service"
)

func TestPackSizesService_GetActive(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*mocks.MockPackSizesRepositoryInterface)
		expectedError error
		expectedSizes []int
	}{
		{
			name: "successful get active",
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				config := &repository.PackSizeConfig{
					ID:        primitive.NewObjectID(),
					Sizes:     []int{250, 500, 1000, 2000, 5000},
					Active:    true,
					Version:   1,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("GetActive", mock.Anything).Return(config, nil)
			},
			expectedError: nil,
			expectedSizes: []int{250, 500, 1000, 2000, 5000},
		},
		{
			name: "no active config",
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				m.On("GetActive", mock.Anything).Return(nil, nil)
			},
			expectedError: nil,
			expectedSizes: nil,
		},
		{
			name: "repository error",
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				m.On("GetActive", mock.Anything).Return(nil, errors.New("database error"))
			},
			expectedError: errors.New("database error"),
			expectedSizes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockPackSizesRepositoryInterface)
			tt.setupMock(mockRepo)

			svc := service.NewPackSizesService(mockRepo)
			config, err := svc.GetActive(context.Background())

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.expectedSizes != nil {
				assert.NotNil(t, config)
				assert.Equal(t, tt.expectedSizes, config.Sizes)
			} else if tt.expectedError == nil {
				assert.Nil(t, config)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestPackSizesService_GetActive_NilRepository(t *testing.T) {
	svc := service.NewPackSizesService(nil)
	config, err := svc.GetActive(context.Background())

	assert.Error(t, err)
	assert.Equal(t, service.ErrRepositoryNotConfigured, err)
	assert.Nil(t, config)
}

func TestPackSizesService_Create(t *testing.T) {
	tests := []struct {
		name          string
		sizes         []int
		createdBy     string
		setupMock     func(*mocks.MockPackSizesRepositoryInterface)
		expectedError error
	}{
		{
			name:      "successful create",
			sizes:     []int{100, 250, 500},
			createdBy: "admin@example.com",
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				config := &repository.PackSizeConfig{
					ID:        primitive.NewObjectID(),
					Sizes:     []int{100, 250, 500},
					Active:    true,
					Version:   1,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
					CreatedBy: "admin@example.com",
				}
				m.On("Create", mock.Anything, []int{100, 250, 500}, "admin@example.com").Return(config, nil)
			},
			expectedError: nil,
		},
		{
			name:      "repository error",
			sizes:     []int{100, 250},
			createdBy: "user@example.com",
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				m.On("Create", mock.Anything, []int{100, 250}, "user@example.com").Return(nil, errors.New("duplicate key"))
			},
			expectedError: errors.New("duplicate key"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockPackSizesRepositoryInterface)
			tt.setupMock(mockRepo)

			svc := service.NewPackSizesService(mockRepo)
			config, err := svc.Create(context.Background(), tt.sizes, tt.createdBy)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, tt.sizes, config.Sizes)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestPackSizesService_Create_NilRepository(t *testing.T) {
	svc := service.NewPackSizesService(nil)
	config, err := svc.Create(context.Background(), []int{100, 250}, "admin")

	assert.Error(t, err)
	assert.Equal(t, service.ErrRepositoryNotConfigured, err)
	assert.Nil(t, config)
}

func TestPackSizesService_Update(t *testing.T) {
	testID := primitive.NewObjectID()

	tests := []struct {
		name          string
		id            primitive.ObjectID
		sizes         []int
		updatedBy     string
		setupMock     func(*mocks.MockPackSizesRepositoryInterface)
		expectedError error
	}{
		{
			name:      "successful update",
			id:        testID,
			sizes:     []int{200, 400, 800},
			updatedBy: "admin@example.com",
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				config := &repository.PackSizeConfig{
					ID:        testID,
					Sizes:     []int{200, 400, 800},
					Active:    true,
					Version:   2,
					UpdatedAt: time.Now(),
				}
				m.On("Update", mock.Anything, testID, []int{200, 400, 800}, "admin@example.com").Return(config, nil)
			},
			expectedError: nil,
		},
		{
			name:      "not found",
			id:        primitive.NewObjectID(),
			sizes:     []int{100},
			updatedBy: "user@example.com",
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				m.On("Update", mock.Anything, mock.AnythingOfType("primitive.ObjectID"), []int{100}, "user@example.com").Return(nil, errors.New("not found"))
			},
			expectedError: errors.New("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockPackSizesRepositoryInterface)
			tt.setupMock(mockRepo)

			svc := service.NewPackSizesService(mockRepo)
			config, err := svc.Update(context.Background(), tt.id, tt.sizes, tt.updatedBy)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, tt.sizes, config.Sizes)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestPackSizesService_Update_NilRepository(t *testing.T) {
	svc := service.NewPackSizesService(nil)
	config, err := svc.Update(context.Background(), primitive.NewObjectID(), []int{100}, "admin")

	assert.Error(t, err)
	assert.Equal(t, service.ErrRepositoryNotConfigured, err)
	assert.Nil(t, config)
}

func TestPackSizesService_List(t *testing.T) {
	tests := []struct {
		name          string
		limit         int
		setupMock     func(*mocks.MockPackSizesRepositoryInterface)
		expectedError error
		expectedCount int
	}{
		{
			name:  "successful list",
			limit: 10,
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				configs := []repository.PackSizeConfig{
					{ID: primitive.NewObjectID(), Sizes: []int{100, 200}, Active: true},
					{ID: primitive.NewObjectID(), Sizes: []int{300, 400}, Active: false},
				}
				m.On("List", mock.Anything, 10).Return(configs, nil)
			},
			expectedError: nil,
			expectedCount: 2,
		},
		{
			name:  "empty list",
			limit: 5,
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				m.On("List", mock.Anything, 5).Return([]repository.PackSizeConfig{}, nil)
			},
			expectedError: nil,
			expectedCount: 0,
		},
		{
			name:  "repository error",
			limit: 10,
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				m.On("List", mock.Anything, 10).Return(nil, errors.New("connection error"))
			},
			expectedError: errors.New("connection error"),
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockPackSizesRepositoryInterface)
			tt.setupMock(mockRepo)

			svc := service.NewPackSizesService(mockRepo)
			configs, err := svc.List(context.Background(), tt.limit)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, configs)
			} else {
				assert.NoError(t, err)
				assert.Len(t, configs, tt.expectedCount)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestPackSizesService_List_NilRepository(t *testing.T) {
	svc := service.NewPackSizesService(nil)
	configs, err := svc.List(context.Background(), 10)

	assert.Error(t, err)
	assert.Equal(t, service.ErrRepositoryNotConfigured, err)
	assert.Nil(t, configs)
}
