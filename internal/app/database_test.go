//go:build !integration

package app

import (
	"errors"
	"testing"

	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/guttosm/pack-service/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestInitializeDefaultPackSizes(t *testing.T) {
	tests := []struct {
		name        string
		defaultSizes []int
		setupMock   func(*mocks.MockPackSizesRepositoryInterface)
		wantError   bool
	}{
		{
			name:        "no active config creates default",
			defaultSizes: []int{5000, 2000, 1000},
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				m.On("GetActive", mock.Anything).Return(nil, nil).Once()
				config := &repository.PackSizeConfig{
					ID:     primitive.NewObjectID(),
					Sizes:  []int{5000, 2000, 1000},
					Active: true,
				}
				m.On("Create", mock.Anything, []int{5000, 2000, 1000}, "system").Return(config, nil).Once()
			},
			wantError: false,
		},
		{
			name:        "active config exists skips creation",
			defaultSizes: []int{5000, 2000, 1000},
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				activeConfig := &repository.PackSizeConfig{
					ID:     primitive.NewObjectID(),
					Sizes:  []int{5000, 2000, 1000},
					Active: true,
				}
				m.On("GetActive", mock.Anything).Return(activeConfig, nil).Once()
			},
			wantError: false,
		},
		{
			name:        "empty default sizes uses service defaults",
			defaultSizes: []int{},
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				m.On("GetActive", mock.Anything).Return(nil, nil).Once()
				config := &repository.PackSizeConfig{
					ID:     primitive.NewObjectID(),
					Sizes:  []int{5000, 2000, 1000, 500, 250},
					Active: true,
				}
				m.On("Create", mock.Anything, mock.Anything, "system").Return(config, nil).Once()
			},
			wantError: false,
		},
		{
			name:        "get active error",
			defaultSizes: []int{5000, 2000, 1000},
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				m.On("GetActive", mock.Anything).Return(nil, errors.New("database error")).Once()
			},
			wantError: true,
		},
		{
			name:        "create error",
			defaultSizes: []int{5000, 2000, 1000},
			setupMock: func(m *mocks.MockPackSizesRepositoryInterface) {
				m.On("GetActive", mock.Anything).Return(nil, nil).Once()
				m.On("Create", mock.Anything, mock.Anything, "system").Return(nil, errors.New("database error")).Once()
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockPackSizesRepositoryInterface)
			mockRepo.Test(t)
			tt.setupMock(mockRepo)

			err := initializeDefaultPackSizes(mockRepo, tt.defaultSizes)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}
