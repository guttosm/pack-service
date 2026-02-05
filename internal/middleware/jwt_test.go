package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/guttosm/pack-service/internal/service"
)

func TestJWTAuth(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		setupMocks     func(*mocks.MockAuthService)
		expectedStatus int
		expectUserInfo bool
	}{
		{
			name:       "valid token",
			authHeader: "Bearer valid-token",
			setupMocks: func(mockAuth *mocks.MockAuthService) {
				userID := primitive.NewObjectID()
				claims := &dto.Claims{
					UserID: userID,
					Email:  "test@example.com",
					Name:   "Test User",
					Roles:  []string{"user"},
				}
				mockAuth.On("ValidateToken", mock.Anything, "valid-token").Return(claims, nil)
			},
			expectedStatus: http.StatusOK,
			expectUserInfo: true,
		},
		{
			name:       "missing authorization header",
			authHeader: "",
			setupMocks: func(mockAuth *mocks.MockAuthService) {
				// No calls expected
			},
			expectedStatus: http.StatusUnauthorized,
			expectUserInfo: false,
		},
		{
			name:       "invalid bearer prefix",
			authHeader: "Token valid-token",
			setupMocks: func(mockAuth *mocks.MockAuthService) {
				// No calls expected
			},
			expectedStatus: http.StatusUnauthorized,
			expectUserInfo: false,
		},
		{
			name:       "empty token",
			authHeader: "Bearer ",
			setupMocks: func(mockAuth *mocks.MockAuthService) {
				// No calls expected
			},
			expectedStatus: http.StatusUnauthorized,
			expectUserInfo: false,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid-token",
			setupMocks: func(mockAuth *mocks.MockAuthService) {
				mockAuth.On("ValidateToken", mock.Anything, "invalid-token").Return(nil, service.ErrInvalidToken)
			},
			expectedStatus: http.StatusUnauthorized,
			expectUserInfo: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			mockAuthService := new(mocks.MockAuthService)

			tt.setupMocks(mockAuthService)

			router.Use(RequestID())
			router.Use(JWTAuth(mockAuthService))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectUserInfo {
				// Verify user info is set in context (would need to check in handler)
				assert.Contains(t, w.Body.String(), "ok")
			}

			mockAuthService.AssertExpectations(t)
		})
	}
}

func TestJWTAuth_UserInfoInContext(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		setupMocks     func(*mocks.MockAuthService) (*dto.Claims, primitive.ObjectID)
		expectedStatus int
		validateContext func(*testing.T, *gin.Context, primitive.ObjectID, *dto.Claims)
	}{
		{
			name:       "user info set in context with valid token",
			authHeader: "Bearer valid-token",
			setupMocks: func(mockAuth *mocks.MockAuthService) (*dto.Claims, primitive.ObjectID) {
				userID := primitive.NewObjectID()
				claims := &dto.Claims{
					UserID: userID,
					Email:  "test@example.com",
					Name:   "Test User",
					Roles:  []string{"user", "admin"},
				}
				mockAuth.On("ValidateToken", mock.Anything, "valid-token").Return(claims, nil)
				return claims, userID
			},
			expectedStatus: http.StatusOK,
			validateContext: func(t *testing.T, c *gin.Context, expectedUserID primitive.ObjectID, expectedClaims *dto.Claims) {
				userIDVal, exists := c.Get("user_id")
				assert.True(t, exists)
				assert.Equal(t, expectedUserID, userIDVal)

				email, exists := c.Get("user_email")
				assert.True(t, exists)
				assert.Equal(t, expectedClaims.Email, email)

				name, exists := c.Get("user_name")
				assert.True(t, exists)
				assert.Equal(t, expectedClaims.Name, name)

				roles, exists := c.Get("user_roles")
				assert.True(t, exists)
				assert.Equal(t, expectedClaims.Roles, roles)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			mockAuthService := new(mocks.MockAuthService)

			claims, userID := tt.setupMocks(mockAuthService)

			router.Use(RequestID())
			router.Use(JWTAuth(mockAuthService))
			router.GET("/test", func(c *gin.Context) {
				if tt.validateContext != nil {
					tt.validateContext(t, c, userID, claims)
				}
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockAuthService.AssertExpectations(t)
		})
	}
}
