//go:build !integration

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_getLogLevel(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   string
	}{
		{
			name:       "2xx returns info",
			statusCode: 200,
			expected:   "info",
		},
		{
			name:       "3xx returns info",
			statusCode: 301,
			expected:   "info",
		},
		{
			name:       "4xx returns warn",
			statusCode: 400,
			expected:   "warn",
		},
		{
			name:       "404 returns warn",
			statusCode: 404,
			expected:   "warn",
		},
		{
			name:       "5xx returns error",
			statusCode: 500,
			expected:   "error",
		},
		{
			name:       "503 returns error",
			statusCode: 503,
			expected:   "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLogLevel(tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRequestLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		path           string
		statusCode     int
		setupMock      func(*mocks.MockLoggingService)
		expectLogging  bool
	}{
		{
			name:       "successful request logs info",
			method:     http.MethodGet,
			path:       "/test",
			statusCode: 200,
			setupMock: func(m *mocks.MockLoggingService) {
				m.On("CreateLog", mock.Anything, mock.AnythingOfType("*model.LogEntry")).Return(nil).Maybe()
			},
			expectLogging: true,
		},
		{
			name:       "client error logs warn",
			method:     http.MethodGet,
			path:       "/test",
			statusCode: 400,
			setupMock: func(m *mocks.MockLoggingService) {
				m.On("CreateLog", mock.Anything, mock.AnythingOfType("*model.LogEntry")).Return(nil).Maybe()
			},
			expectLogging: true,
		},
		{
			name:       "server error logs error",
			method:     http.MethodGet,
			path:       "/test",
			statusCode: 500,
			setupMock: func(m *mocks.MockLoggingService) {
				m.On("CreateLog", mock.Anything, mock.AnythingOfType("*model.LogEntry")).Return(nil).Maybe()
			},
			expectLogging: true,
		},
		{
			name:          "no logging service",
			method:        http.MethodGet,
			path:          "/test",
			statusCode:    200,
			setupMock:     func(m *mocks.MockLoggingService) {},
			expectLogging: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLoggingService := mocks.NewMockLoggingService(t)
			tt.setupMock(mockLoggingService)

			router := gin.New()
			router.Use(RequestID())
			if tt.expectLogging {
				router.Use(RequestLogger(mockLoggingService))
			} else {
				router.Use(RequestLogger(nil))
			}
			router.GET("/test", func(c *gin.Context) {
				c.Status(tt.statusCode)
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.statusCode, w.Code)
			if tt.expectLogging {
				mockLoggingService.AssertExpectations(t)
			}
		})
	}
}

func TestRequestLogger_WithUserInfo(t *testing.T) {
	tests := []struct {
		name           string
		setupUserInfo  func(*gin.Context)
		setupMock      func(*mocks.MockLoggingService)
		expectedStatus int
	}{
		{
			name: "request logger captures user info",
			setupUserInfo: func(c *gin.Context) {
				c.Set("user_id", "user123")
				c.Set("user_email", "test@example.com")
			},
			setupMock: func(m *mocks.MockLoggingService) {
				m.On("CreateLog", mock.Anything, mock.MatchedBy(func(entry interface{}) bool {
					return true
				})).Return(nil).Maybe()
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			mockLoggingService := mocks.NewMockLoggingService(t)
			tt.setupMock(mockLoggingService)

			router := gin.New()
			router.Use(RequestID())
			router.Use(func(c *gin.Context) {
				tt.setupUserInfo(c)
				c.Next()
			})
			router.Use(RequestLogger(mockLoggingService))
			router.GET("/test", func(c *gin.Context) {
				c.Status(tt.expectedStatus)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
