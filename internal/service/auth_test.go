package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"

	"github.com/guttosm/pack-service/config"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/guttosm/pack-service/internal/service"
)

// testAuthConfig returns a config.AuthConfig for testing.
func testAuthConfig() config.AuthConfig {
	return config.AuthConfig{
		JWTSecretKey:     "your-secret-key-change-in-production",
		JWTRefreshSecret: "your-refresh-secret-key-change-in-production",
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  7 * 24 * time.Hour,
	}
}

func TestAuthService_Login(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		password      string
		setupMocks    func(*mocks.MockUserRepositoryInterface)
		expectedError error
		validateToken bool
	}{
		{
			name:     "successful login",
			email:    "test@example.com",
			password: "password123",
			setupMocks: func(mockRepo *mocks.MockUserRepositoryInterface) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
				user := &model.User{
					ID:       primitive.NewObjectID(),
					Email:    "test@example.com",
					Password: string(hashedPassword),
					Name:     "Test User",
					Active:   true,
				}
				mockRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
			},
			expectedError: nil,
			validateToken: true,
		},
		{
			name:     "user not found",
			email:    "notfound@example.com",
			password: "password123",
			setupMocks: func(mockRepo *mocks.MockUserRepositoryInterface) {
				mockRepo.On("FindByEmail", mock.Anything, "notfound@example.com").Return(nil, nil)
			},
			expectedError: service.ErrInvalidCredentials,
			validateToken: false,
		},
		{
			name:     "user inactive",
			email:    "inactive@example.com",
			password: "password123",
			setupMocks: func(mockRepo *mocks.MockUserRepositoryInterface) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
				user := &model.User{
					ID:       primitive.NewObjectID(),
					Email:    "inactive@example.com",
					Password: string(hashedPassword),
					Name:     "Inactive User",
					Active:   false,
				}
				mockRepo.On("FindByEmail", mock.Anything, "inactive@example.com").Return(user, nil)
			},
			expectedError: service.ErrInvalidCredentials,
			validateToken: false,
		},
		{
			name:     "wrong password",
			email:    "test@example.com",
			password: "wrongpassword",
			setupMocks: func(mockRepo *mocks.MockUserRepositoryInterface) {
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
				user := &model.User{
					ID:       primitive.NewObjectID(),
					Email:    "test@example.com",
					Password: string(hashedPassword),
					Name:     "Test User",
					Active:   true,
				}
				mockRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
			},
			expectedError: service.ErrInvalidCredentials,
			validateToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(mocks.MockUserRepositoryInterface)
			mockRoleRepo := new(mocks.MockRoleRepositoryInterface)
			mockTokenRepo := new(mocks.MockTokenRepositoryInterface)

			tt.setupMocks(mockUserRepo)

			if tt.validateToken {
				// Mock DeleteByUserID for invalidating existing tokens
				mockTokenRepo.On("DeleteByUserID", mock.Anything, mock.AnythingOfType("primitive.ObjectID"), "refresh").Return(nil)
				mockTokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Token")).Return(nil)
			}

			authService := service.NewAuthService(mockUserRepo, mockRoleRepo, mockTokenRepo, testAuthConfig())

			tokenPair, user, err := authService.Login(context.Background(), tt.email, tt.password)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, tokenPair)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tokenPair)
				assert.NotNil(t, user)
				assert.NotEmpty(t, tokenPair.AccessToken)
				assert.NotEmpty(t, tokenPair.RefreshToken)
				assert.Equal(t, tt.email, user.Email)

				// Validate token can be parsed
				// Note: We need to use a type that implements jwt.Claims, so we parse with map claims
				// and verify the token structure instead
				token, err := jwt.Parse(tokenPair.AccessToken, func(token *jwt.Token) (interface{}, error) {
					return []byte("your-secret-key-change-in-production"), nil
				})
				assert.NoError(t, err)
				assert.True(t, token.Valid)
				// Token is valid and can be parsed
			}

			mockUserRepo.AssertExpectations(t)
			mockTokenRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		username      string
		password      string
		nameField     string
		setupMocks    func(*mocks.MockUserRepositoryInterface, *mocks.MockRoleRepositoryInterface, *mocks.MockTokenRepositoryInterface)
		expectedError error
		validateToken bool
	}{
		{
			name:      "successful registration",
			email:     "new@example.com",
			username:  "newuser",
			password:  "password123",
			nameField: "New User",
			setupMocks: func(mockUserRepo *mocks.MockUserRepositoryInterface, mockRoleRepo *mocks.MockRoleRepositoryInterface, mockTokenRepo *mocks.MockTokenRepositoryInterface) {
				mockUserRepo.On("FindByEmail", mock.Anything, "new@example.com").Return(nil, nil)
				mockUserRepo.On("FindByUsername", mock.Anything, "newuser").Return(nil, nil)
				// Mock "user" role lookup
				userRole := &model.Role{
					ID:          primitive.NewObjectID(),
					Name:        "user",
					Description: "Standard user role",
					Permissions: []string{"packs:read", "packs:write"},
					Active:      true,
				}
				mockRoleRepo.On("FindByName", mock.Anything, "user").Return(userRole, nil)
				mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil).Run(func(args mock.Arguments) {
					user, _ := args.Get(1).(*model.User)
					if user != nil {
						user.ID = primitive.NewObjectID()
						// Verify user has the "user" role assigned
						assert.Equal(t, []string{userRole.ID.Hex()}, user.Roles)
					}
				})
				mockTokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Token")).Return(nil)
			},
			expectedError: nil,
			validateToken: true,
		},
		{
			name:      "user already exists by email",
			email:     "existing@example.com",
			username:  "newuser",
			password:  "password123",
			nameField: "Existing User",
			setupMocks: func(mockUserRepo *mocks.MockUserRepositoryInterface, mockRoleRepo *mocks.MockRoleRepositoryInterface, mockTokenRepo *mocks.MockTokenRepositoryInterface) {
				existingUser := &model.User{
					ID:    primitive.NewObjectID(),
					Email: "existing@example.com",
				}
				mockUserRepo.On("FindByEmail", mock.Anything, "existing@example.com").Return(existingUser, nil)
			},
			expectedError: service.ErrUserExists,
			validateToken: false,
		},
		{
			name:      "user already exists by username",
			email:     "new@example.com",
			username:  "existinguser",
			password:  "password123",
			nameField: "Existing User",
			setupMocks: func(mockUserRepo *mocks.MockUserRepositoryInterface, mockRoleRepo *mocks.MockRoleRepositoryInterface, mockTokenRepo *mocks.MockTokenRepositoryInterface) {
				mockUserRepo.On("FindByEmail", mock.Anything, "new@example.com").Return(nil, nil)
				existingUser := &model.User{
					ID:       primitive.NewObjectID(),
					Username: "existinguser",
				}
				mockUserRepo.On("FindByUsername", mock.Anything, "existinguser").Return(existingUser, nil)
			},
			expectedError: service.ErrUserExists,
			validateToken: false,
		},
		{
			name:      "user role not found",
			email:     "new@example.com",
			username:  "newuser",
			password:  "password123",
			nameField: "New User",
			setupMocks: func(mockUserRepo *mocks.MockUserRepositoryInterface, mockRoleRepo *mocks.MockRoleRepositoryInterface, mockTokenRepo *mocks.MockTokenRepositoryInterface) {
				mockUserRepo.On("FindByEmail", mock.Anything, "new@example.com").Return(nil, nil)
				mockUserRepo.On("FindByUsername", mock.Anything, "newuser").Return(nil, nil)
				mockRoleRepo.On("FindByName", mock.Anything, "user").Return(nil, nil)
			},
			expectedError: errors.New("user role not found - please ensure default roles are initialized"),
			validateToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(mocks.MockUserRepositoryInterface)
			mockRoleRepo := new(mocks.MockRoleRepositoryInterface)
			mockTokenRepo := new(mocks.MockTokenRepositoryInterface)

			tt.setupMocks(mockUserRepo, mockRoleRepo, mockTokenRepo)

			authService := service.NewAuthService(mockUserRepo, mockRoleRepo, mockTokenRepo, testAuthConfig())

			tokenPair, user, err := authService.Register(context.Background(), tt.email, tt.username, tt.password, tt.nameField)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, tokenPair)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tokenPair)
				assert.NotNil(t, user)
				assert.NotEmpty(t, tokenPair.AccessToken)
				assert.NotEmpty(t, tokenPair.RefreshToken)
				assert.Equal(t, tt.email, user.Email)
				assert.Equal(t, tt.nameField, user.Name)
			}

			mockUserRepo.AssertExpectations(t)
			mockRoleRepo.AssertExpectations(t)
			mockTokenRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_RefreshToken(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*mocks.MockTokenRepositoryInterface, *mocks.MockUserRepositoryInterface) string
		expectedError error
	}{
		{
			name: "successful refresh",
			setupMocks: func(mockTokenRepo *mocks.MockTokenRepositoryInterface, mockUserRepo *mocks.MockUserRepositoryInterface) string {
				userID := primitive.NewObjectID()
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
				user := &model.User{
					ID:       userID,
					Email:    "test@example.com",
					Username: "testuser",
					Password: string(hashedPassword),
					Name:     "Test User",
					Active:   true,
				}
				
				// Generate a real refresh token by logging in
				mockUserRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
				mockTokenRepo.On("DeleteByUserID", mock.Anything, userID, "refresh").Return(nil)
				mockTokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Token")).Return(nil).Times(2)
				
				authService := service.NewAuthService(mockUserRepo, new(mocks.MockRoleRepositoryInterface), mockTokenRepo, testAuthConfig())
				tokenPair, _, err := authService.Login(context.Background(), "test@example.com", "password123")
				if err != nil {
					t.Fatalf("Failed to login: %v", err)
				}
				
				refreshToken := tokenPair.RefreshToken
				
				// Set up mocks for RefreshToken call (need to reset mocks)
				mockTokenRepo.ExpectedCalls = nil
				mockUserRepo.ExpectedCalls = nil
				
				token := &model.Token{
					ID:        primitive.NewObjectID(),
					UserID:    userID,
					Token:     refreshToken,
					Type:      "refresh",
					ExpiresAt: time.Now().Add(24 * time.Hour),
				}
				mockTokenRepo.On("FindByToken", mock.Anything, refreshToken).Return(token, nil)
				mockUserRepo.On("FindByID", mock.Anything, userID).Return(user, nil)
				mockTokenRepo.On("DeleteByToken", mock.Anything, refreshToken).Return(nil)
				mockTokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Token")).Return(nil)
				
				return refreshToken
			},
			expectedError: nil,
		},
		{
			name: "token not found",
			setupMocks: func(mockTokenRepo *mocks.MockTokenRepositoryInterface, mockUserRepo *mocks.MockUserRepositoryInterface) string {
				invalidToken := "invalid-refresh-token-string"
				mockTokenRepo.On("FindByToken", mock.Anything, invalidToken).Return(nil, nil)
				return invalidToken
			},
			expectedError: service.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(mocks.MockUserRepositoryInterface)
			mockRoleRepo := new(mocks.MockRoleRepositoryInterface)
			mockTokenRepo := new(mocks.MockTokenRepositoryInterface)

			refreshToken := tt.setupMocks(mockTokenRepo, mockUserRepo)

			authService := service.NewAuthService(mockUserRepo, mockRoleRepo, mockTokenRepo, testAuthConfig())
			tokenPair, err := authService.RefreshToken(context.Background(), refreshToken)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, tokenPair)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tokenPair)
				assert.NotEmpty(t, tokenPair.AccessToken)
				assert.NotEmpty(t, tokenPair.RefreshToken)
			}
		})
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	tests := []struct {
		name          string
		tokenString   string
		setupMocks    func(*mocks.MockTokenRepositoryInterface)
		expectedError error
	}{
		{
			name: "valid token",
			setupMocks: func(mockTokenRepo *mocks.MockTokenRepositoryInterface) {
				mockTokenRepo.On("IsBlacklisted", mock.Anything, mock.AnythingOfType("string")).Return(false, nil)
			},
			expectedError: nil,
		},
		{
			name: "blacklisted token",
			setupMocks: func(mockTokenRepo *mocks.MockTokenRepositoryInterface) {
				mockTokenRepo.On("IsBlacklisted", mock.Anything, mock.AnythingOfType("string")).Return(true, nil)
			},
			expectedError: service.ErrTokenBlacklisted,
		},
		{
			name: "invalid token format",
			setupMocks: func(mockTokenRepo *mocks.MockTokenRepositoryInterface) {
				mockTokenRepo.On("IsBlacklisted", mock.Anything, "invalid").Return(false, nil)
			},
			expectedError: service.ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(mocks.MockUserRepositoryInterface)
			mockRoleRepo := new(mocks.MockRoleRepositoryInterface)
			mockTokenRepo := new(mocks.MockTokenRepositoryInterface)

			authService := service.NewAuthService(mockUserRepo, mockRoleRepo, mockTokenRepo, testAuthConfig())

			// Generate a valid token for testing
			var tokenString string
			switch tt.name {
			case "valid token":
				userID := primitive.NewObjectID()
				hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
				user := &model.User{
					ID:       userID,
					Email:    "test@example.com",
					Password: string(hashedPassword),
					Name:     "Test User",
					Active:   true,
				}
				mockUserRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(user, nil)
				mockTokenRepo.On("DeleteByUserID", mock.Anything, userID, "refresh").Return(nil)
				mockTokenRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Token")).Return(nil)

				tokenPair, _, _ := authService.Login(context.Background(), "test@example.com", "password123")
				tokenString = tokenPair.AccessToken
			case "invalid token format":
				tokenString = "invalid"
			default:
				tokenString = "blacklisted-token"
			}

			tt.setupMocks(mockTokenRepo)

			claims, err := authService.ValidateToken(context.Background(), tokenString)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, "test@example.com", claims.Email)
			}

			mockTokenRepo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	tests := []struct {
		name          string
		accessToken   string
		refreshToken  string
		setupMocks    func(*mocks.MockTokenRepositoryInterface)
		expectedError error
	}{
		{
			name:         "successful logout with empty tokens",
			accessToken:  "",
			refreshToken: "",
			setupMocks: func(mockTokenRepo *mocks.MockTokenRepositoryInterface) {
				// No tokens to invalidate
			},
			expectedError: nil,
		},
		{
			name:         "logout with only refresh token",
			accessToken:  "",
			refreshToken: "valid-refresh-token",
			setupMocks: func(mockTokenRepo *mocks.MockTokenRepositoryInterface) {
				mockTokenRepo.On("DeleteByToken", mock.Anything, "valid-refresh-token").Return(nil)
			},
			expectedError: nil,
		},
		{
			name:         "logout with invalid access token format",
			accessToken:  "invalid-token",
			refreshToken: "valid-refresh-token",
			setupMocks: func(mockTokenRepo *mocks.MockTokenRepositoryInterface) {
				// InvalidateToken will fail due to invalid JWT format
				// But refresh token deletion should still succeed
				mockTokenRepo.On("DeleteByToken", mock.Anything, "valid-refresh-token").Return(nil)
			},
			expectedError: errors.New("invalidate access token"), // Now returns error for invalid tokens
		},
		{
			name:         "logout with refresh token deletion error",
			accessToken:  "",
			refreshToken: "valid-refresh-token",
			setupMocks: func(mockTokenRepo *mocks.MockTokenRepositoryInterface) {
				mockTokenRepo.On("DeleteByToken", mock.Anything, "valid-refresh-token").Return(errors.New("deletion failed"))
			},
			expectedError: errors.New("delete refresh token"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := new(mocks.MockUserRepositoryInterface)
			mockRoleRepo := new(mocks.MockRoleRepositoryInterface)
			mockTokenRepo := new(mocks.MockTokenRepositoryInterface)

			tt.setupMocks(mockTokenRepo)

			authService := service.NewAuthService(mockUserRepo, mockRoleRepo, mockTokenRepo, testAuthConfig())

			err := authService.Logout(context.Background(), tt.accessToken, tt.refreshToken)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			mockTokenRepo.AssertExpectations(t)
		})
	}
}
