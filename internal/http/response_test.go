package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseBuilder_Success(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		data       interface{}
		validate   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:       "SuccessOK with PackResult",
			statusCode: http.StatusOK,
			data:       model.PackResult{OrderedItems: 100, TotalItems: 250, Packs: []model.Pack{{Size: 250, Quantity: 1}}},
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, w.Code)
				assert.NotEmpty(t, resp.RequestID)
				assert.NotZero(t, resp.Timestamp)
				assert.NotNil(t, resp.Data)
			},
		},
		{
			name:       "Success with custom status",
			statusCode: http.StatusCreated,
			data:       map[string]string{"message": "created"},
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, http.StatusCreated, w.Code)
				assert.NotEmpty(t, resp.RequestID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/test", nil)
			middleware.RequestID()(c)

			builder := NewResponseBuilder(c)
			builder.Success(tt.statusCode, tt.data)

			if tt.validate != nil {
				tt.validate(t, w)
			}
		})
	}
}

func TestResponseBuilder_SuccessOK(t *testing.T) {
	tests := []struct {
		name       string
		data       interface{}
		validate   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "SuccessOK with map data",
			data: map[string]string{"test": "data"},
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, w.Code)
				assert.NotEmpty(t, resp.RequestID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/test", nil)
			middleware.RequestID()(c)

			builder := NewResponseBuilder(c)
			builder.SuccessOK(tt.data)

			if tt.validate != nil {
				tt.validate(t, w)
			}
		})
	}
}

func TestResponseBuilder_Error(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
		err        error
		validate   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:       "Error 400 Bad Request",
			statusCode: http.StatusBadRequest,
			message:    "invalid input",
			err:        nil,
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.Equal(t, dto.ErrCodeInvalidRequest, resp.Error)
				assert.Equal(t, "invalid input", resp.Message)
				assert.NotEmpty(t, resp.RequestID)
			},
		},
		{
			name:       "Error 500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			message:    "server error",
			err:        nil,
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, http.StatusInternalServerError, w.Code)
				assert.Equal(t, dto.ErrCodeInternal, resp.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/test", nil)
			middleware.RequestID()(c)

			builder := NewResponseBuilder(c)
			builder.Error(tt.statusCode, tt.message, tt.err)

			if tt.validate != nil {
				tt.validate(t, w)
			}
		})
	}
}

func TestResponseBuilder_SuccessAccepted(t *testing.T) {
	tests := []struct {
		name           string
		data           interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "SuccessAccepted with map data",
			data:           map[string]interface{}{"status": "accepted"},
			expectedStatus: http.StatusAccepted,
			expectedBody:   "accepted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(middleware.RequestID())
			router.POST("/test", func(c *gin.Context) {
				builder := NewResponseBuilder(c)
				builder.SuccessAccepted(tt.data)
			})

			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
		})
	}
}

func TestSuccessResponse_JSON(t *testing.T) {
	tests := []struct {
		name           string
		resp           dto.SuccessResponse
		expectedFields []string
	}{
		{
			name: "SuccessResponse JSON marshaling",
			resp: dto.SuccessResponse{
				Data:      model.PackResult{OrderedItems: 100, TotalItems: 250},
				RequestID: "test-id",
				Timestamp: time.Now(),
			},
			expectedFields: []string{"test-id", "data", "request_id", "timestamp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.resp)
			require.NoError(t, err)
			for _, field := range tt.expectedFields {
				assert.Contains(t, string(data), field)
			}
		})
	}
}
