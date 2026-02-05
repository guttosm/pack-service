// Code generated manually. DO NOT EDIT.

package mocks

import (
	"context"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/guttosm/pack-service/internal/repository"
)

type MockPackSizesRepositoryInterface struct {
	mock.Mock
}

func (m *MockPackSizesRepositoryInterface) GetActive(ctx context.Context) (*repository.PackSizeConfig, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PackSizeConfig), args.Error(1)
}

func (m *MockPackSizesRepositoryInterface) Create(ctx context.Context, sizes []int, createdBy string) (*repository.PackSizeConfig, error) {
	args := m.Called(ctx, sizes, createdBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PackSizeConfig), args.Error(1)
}

func (m *MockPackSizesRepositoryInterface) Update(ctx context.Context, id primitive.ObjectID, sizes []int, updatedBy string) (*repository.PackSizeConfig, error) {
	args := m.Called(ctx, id, sizes, updatedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PackSizeConfig), args.Error(1)
}

func (m *MockPackSizesRepositoryInterface) List(ctx context.Context, limit int) ([]repository.PackSizeConfig, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.PackSizeConfig), args.Error(1)
}
