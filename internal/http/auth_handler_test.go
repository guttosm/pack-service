package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/guttosm/pack-service/internal/service"
)

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*mocks.MockAuthService, *mocks.MockLoggingService)
		expectedStatus int
		validateResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "successful login",
			requestBody: dto.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func(mockAuth *mocks.MockAuthService, mockLogging *mocks.MockLoggingService) {
				userID := primitive.NewObjectID()
				tokenPair := &dto.TokenPair{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresIn:    900,
				}
				user := &model.User{
					ID:    userID,
					Email: "test@example.com",
					Name:  "Test User",
				}
				mockAuth.On("Login", mock.Anything, "test@example.com", "password123").Return(tokenPair, user, nil)
				mockLogging.On("CreateLog", mock.Anything, mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response dto.SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotNil(t, response.Data)
			},
		},
		{
			name: "invalid credentials",
			requestBody: dto.LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			setupMocks: func(mockAuth *mocks.MockAuthService, mockLogging *mocks.MockLoggingService) {
				mockAuth.On("Login", mock.Anything, "test@example.com", "wrongpassword").Return(nil, nil, service.ErrInvalidCredentials)
				mockLogging.On("CreateLog", mock.Anything, mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusUnauthorized,
			validateResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response dto.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Error)
			},
		},
		{
			name: "invalid request body",
			requestBody: map[string]interface{}{
				"email": "invalid-email",
			},
			setupMocks: func(mockAuth *mocks.MockAuthService, mockLogging *mocks.MockLoggingService) {
				// No calls expected
			},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response dto.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Error)
			},
		},
		{
			name: "validation error",
			requestBody: dto.LoginRequest{
				Email:    "",
				Password: "password123",
			},
			setupMocks: func(mockAuth *mocks.MockAuthService, mockLogging *mocks.MockLoggingService) {
				// No calls expected
			},
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response dto.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			mockAuthService := new(mocks.MockAuthService)
			mockLoggingService := new(mocks.MockLoggingService)

			tt.setupMocks(mockAuthService, mockLoggingService)

			handler := NewAuthHandler(mockAuthService)
			router.Use(func(c *gin.Context) {
				c.Set("logging_service", mockLoggingService)
				c.Next()
			})
			router.POST("/login", handler.Login)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateResponse != nil {
				tt.validateResponse(t, w)
			}

			mockAuthService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMocks     func(*mocks.MockAuthService, *mocks.MockLoggingService)
		expectedStatus int
	}{
		{
			name: "successful registration",
			requestBody: dto.RegisterRequest{
				Email:    "new@example.com",
				Username: "newuser",
				Password: "password123",
				Name:     "New User",
			},
			setupMocks: func(mockAuth *mocks.MockAuthService, mockLogging *mocks.MockLoggingService) {
				userID := primitive.NewObjectID()
				tokenPair := &dto.TokenPair{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresIn:    900,
				}
				user := &model.User{
					ID:       userID,
					Email:    "new@example.com",
					Username: "newuser",
					Name:     "New User",
				}
				mockAuth.On("Register", mock.Anything, "new@example.com", "newuser", "password123", "New User").Return(tokenPair, user, nil)
				mockLogging.On("CreateLog", mock.Anything, mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "user already exists",
			requestBody: dto.RegisterRequest{
				Email:    "existing@example.com",
				Username: "existinguser",
				Password: "password123",
				Name:     "Existing User",
			},
			setupMocks: func(mockAuth *mocks.MockAuthService, mockLogging *mocks.MockLoggingService) {
				mockAuth.On("Register", mock.Anything, "existing@example.com", "existinguser", "password123", "Existing User").Return(nil, nil, service.ErrUserExists)
				mockLogging.On("CreateLog", mock.Anything, mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			mockAuthService := new(mocks.MockAuthService)
			mockLoggingService := new(mocks.MockLoggingService)

			tt.setupMocks(mockAuthService, mockLoggingService)

			handler := NewAuthHandler(mockAuthService)
			router.Use(func(c *gin.Context) {
				c.Set("logging_service", mockLoggingService)
				c.Next()
			})
			router.POST("/register", handler.Register)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			mockAuthService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	tests := []struct {
		name            string
		refreshTokenHeader string
		setupMocks      func(*mocks.MockAuthService)
		expectedStatus  int
	}{
		{
			name: "successful refresh",
			refreshTokenHeader: "valid-refresh-token",
			setupMocks: func(mockAuth *mocks.MockAuthService) {
				tokenPair := &dto.TokenPair{
					AccessToken:  "new-access-token",
					RefreshToken: "new-refresh-token",
					ExpiresIn:    900,
				}
				mockAuth.On("RefreshToken", mock.Anything, "valid-refresh-token").Return(tokenPair, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing refresh token header",
			refreshTokenHeader: "",
			setupMocks: func(mockAuth *mocks.MockAuthService) {
				// No mock calls expected
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid refresh token",
			refreshTokenHeader: "invalid-token",
			setupMocks: func(mockAuth *mocks.MockAuthService) {
				mockAuth.On("RefreshToken", mock.Anything, "invalid-token").Return(nil, service.ErrInvalidToken)
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			mockAuthService := new(mocks.MockAuthService)

			tt.setupMocks(mockAuthService)

			handler := NewAuthHandler(mockAuthService)
			router.POST("/refresh", handler.RefreshToken)

			req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
			req.Header.Set("Content-Type", "application/json")
			if tt.refreshTokenHeader != "" {
				req.Header.Set("X-Refresh-Token", tt.refreshTokenHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			mockAuthService.AssertExpectations(t)
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	tests := []struct {
		name               string
		authHeader         string
		refreshTokenHeader string
		setupMocks         func(*mocks.MockAuthService, *mocks.MockLoggingService)
		expectedStatus     int
	}{
		{
			name:               "successful logout",
			authHeader:         "Bearer access-token",
			refreshTokenHeader: "refresh-token",
			setupMocks: func(mockAuth *mocks.MockAuthService, mockLogging *mocks.MockLoggingService) {
				mockAuth.On("Logout", mock.Anything, "access-token", "refresh-token").Return(nil)
				mockLogging.On("CreateLog", mock.Anything, mock.Anything).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:               "missing authorization header",
			authHeader:         "",
			refreshTokenHeader: "refresh-token",
			setupMocks:         func(mockAuth *mocks.MockAuthService, mockLogging *mocks.MockLoggingService) {},
			expectedStatus:     http.StatusUnauthorized,
		},
		{
			name:               "invalid authorization header format",
			authHeader:         "Token access-token",
			refreshTokenHeader: "refresh-token",
			setupMocks:         func(mockAuth *mocks.MockAuthService, mockLogging *mocks.MockLoggingService) {},
			expectedStatus:     http.StatusUnauthorized,
		},
		{
			name:               "missing refresh token header",
			authHeader:         "Bearer access-token",
			refreshTokenHeader: "",
			setupMocks:          func(mockAuth *mocks.MockAuthService, mockLogging *mocks.MockLoggingService) {},
			expectedStatus:     http.StatusBadRequest,
		},
		{
			name:               "logout error",
			authHeader:         "Bearer access-token",
			refreshTokenHeader: "refresh-token",
			setupMocks: func(mockAuth *mocks.MockAuthService, mockLogging *mocks.MockLoggingService) {
				mockAuth.On("Logout", mock.Anything, "access-token", "refresh-token").Return(assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:               "logout without logging service",
			authHeader:         "Bearer access-token",
			refreshTokenHeader: "refresh-token",
			setupMocks: func(mockAuth *mocks.MockAuthService, mockLogging *mocks.MockLoggingService) {
				mockAuth.On("Logout", mock.Anything, "access-token", "refresh-token").Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			mockAuthService := new(mocks.MockAuthService)
			mockLoggingService := new(mocks.MockLoggingService)

			tt.setupMocks(mockAuthService, mockLoggingService)

			handler := NewAuthHandler(mockAuthService)
			if tt.name != "logout without logging service" {
				router.Use(func(c *gin.Context) {
					c.Set("logging_service", mockLoggingService)
					c.Next()
				})
			}
			router.POST("/logout", handler.Logout)

			req := httptest.NewRequest(http.MethodPost, "/logout", nil)
			req.Header.Set("Content-Type", "application/json")
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.refreshTokenHeader != "" {
				req.Header.Set("X-Refresh-Token", tt.refreshTokenHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			mockAuthService.AssertExpectations(t)
		})
	}
}
