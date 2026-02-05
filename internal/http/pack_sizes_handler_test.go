package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/guttosm/pack-service/internal/repository"
	"github.com/guttosm/pack-service/internal/service"
)

func TestPackSizesHandler_GetActivePackSizes(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*mocks.MockPackSizesRepositoryInterface, *mocks.MockLoggingService)
		expectedStatus int
	}{
		{
			name: "successful get active pack sizes",
			setupMocks: func(mockRepo *mocks.MockPackSizesRepositoryInterface, mockLogging *mocks.MockLoggingService) {
				config := &repository.PackSizeConfig{
					ID:        primitive.NewObjectID(),
					Sizes:     []int{250, 500, 1000, 2000, 5000},
					Version:   1,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				mockRepo.On("GetActive", mock.Anything).Return(config, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "no active pack sizes found",
			setupMocks: func(mockRepo *mocks.MockPackSizesRepositoryInterface, mockLogging *mocks.MockLoggingService) {
				mockRepo.On("GetActive", mock.Anything).Return(nil, nil)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "repository error",
			setupMocks: func(mockRepo *mocks.MockPackSizesRepositoryInterface, mockLogging *mocks.MockLoggingService) {
				mockRepo.On("GetActive", mock.Anything).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			mockRepo := new(mocks.MockPackSizesRepositoryInterface)
			mockLogging := new(mocks.MockLoggingService)

			tt.setupMocks(mockRepo, mockLogging)

			mockService := service.NewPackSizesService(mockRepo)
			handler := NewPackSizesHandler(mockService, nil)
			router.Use(func(c *gin.Context) {
				c.Set("logging_service", mockLogging)
				c.Next()
			})
			router.GET("/pack-sizes", handler.GetActivePackSizes)

			req := httptest.NewRequest(http.MethodGet, "/pack-sizes", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestPackSizesHandler_UpdatePackSizes(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*mocks.MockPackSizesRepositoryInterface, *mocks.MockLoggingService)
		expectedStatus int
	}{
		{
			name: "successful update",
			requestBody: map[string]interface{}{
				"sizes": []int{250, 500, 1000},
			},
			setupMocks: func(mockRepo *mocks.MockPackSizesRepositoryInterface, mockLogging *mocks.MockLoggingService) {
				config := &repository.PackSizeConfig{
					ID:        primitive.NewObjectID(),
					Sizes:     []int{250, 500, 1000},
					Version:   1,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				mockRepo.On("Create", mock.Anything, []int{250, 500, 1000}, mock.Anything).Return(config, nil)
				// Audit logging is async, so we allow it but don't assert
				mockLogging.On("CreateLog", mock.Anything, mock.Anything).Maybe().Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid request body",
			requestBody: map[string]interface{}{
				"pack_sizes": "invalid",
			},
			setupMocks: func(mockRepo *mocks.MockPackSizesRepositoryInterface, mockLogging *mocks.MockLoggingService) {
				// No calls expected
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "empty pack sizes",
			requestBody: map[string]interface{}{
				"sizes": []int{},
			},
			setupMocks: func(mockRepo *mocks.MockPackSizesRepositoryInterface, mockLogging *mocks.MockLoggingService) {
				// No calls expected
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "repository create error",
			requestBody: map[string]interface{}{
				"sizes": []int{250, 500},
			},
			setupMocks: func(mockRepo *mocks.MockPackSizesRepositoryInterface, mockLogging *mocks.MockLoggingService) {
				mockRepo.On("Create", mock.Anything, []int{250, 500}, mock.Anything).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			mockRepo := new(mocks.MockPackSizesRepositoryInterface)
			mockLogging := new(mocks.MockLoggingService)

			tt.setupMocks(mockRepo, mockLogging)

			mockService := service.NewPackSizesService(mockRepo)
			handler := NewPackSizesHandler(mockService, nil)
			router.Use(func(c *gin.Context) {
				c.Set("logging_service", mockLogging)
				c.Next()
			})
			router.PUT("/pack-sizes", handler.UpdatePackSizes)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPut, "/pack-sizes", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestPackSizesHandler_ListPackSizes(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*mocks.MockPackSizesRepositoryInterface)
		expectedStatus int
	}{
		{
			name: "successful list",
			setupMocks: func(mockRepo *mocks.MockPackSizesRepositoryInterface) {
				configs := []repository.PackSizeConfig{
					{
						ID:        primitive.NewObjectID(),
						Sizes:     []int{250, 500},
						Version:   1,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
					{
						ID:        primitive.NewObjectID(),
						Sizes:     []int{100, 200},
						Version:   2,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				}
				mockRepo.On("List", mock.Anything, mock.Anything).Return(configs, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "repository error",
			setupMocks: func(mockRepo *mocks.MockPackSizesRepositoryInterface) {
				mockRepo.On("List", mock.Anything, mock.Anything).Return(nil, assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			mockRepo := new(mocks.MockPackSizesRepositoryInterface)

			tt.setupMocks(mockRepo)

			mockService := service.NewPackSizesService(mockRepo)
			handler := NewPackSizesHandler(mockService, nil)
			router.GET("/pack-sizes/list", handler.ListPackSizes)

			req := httptest.NewRequest(http.MethodGet, "/pack-sizes/list", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestPackSizesHandler_parseInt(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValue int
		wantError bool
	}{
		{
			name:      "valid integer",
			input:     "123",
			wantValue: 123,
			wantError: false,
		},
		{
			name:      "invalid integer",
			input:     "abc",
			wantValue: 0,
			wantError: true,
		},
		{
			name:      "empty string",
			input:     "",
			wantValue: 0,
			wantError: true,
		},
		{
			name:      "negative integer",
			input:     "-10",
			wantValue: -10,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := parseInt(tt.input)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantValue, value)
			}
		})
	}
}
