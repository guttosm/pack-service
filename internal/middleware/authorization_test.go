//go:build !integration

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRequireAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		config         AuthorizationConfig
		setupMocks     func(*mocks.MockRoleService, *mocks.MockPermissionService)
		expectedStatus int
	}{
		{
			name: "no user claims returns unauthorized",
			setupContext: func(c *gin.Context) {
			},
			config: AuthorizationConfig{},
			setupMocks: func(roleService *mocks.MockRoleService, permService *mocks.MockPermissionService) {
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid claims type returns unauthorized",
			setupContext: func(c *gin.Context) {
				c.Set("user_claims", "invalid")
			},
			config: AuthorizationConfig{},
			setupMocks: func(roleService *mocks.MockRoleService, permService *mocks.MockPermissionService) {
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "no requirements allows access",
			setupContext: func(c *gin.Context) {
				c.Set("user_claims", &dto.Claims{
					UserID: primitive.NewObjectID(),
					Roles:  []string{},
				})
			},
			config: AuthorizationConfig{},
			setupMocks: func(roleService *mocks.MockRoleService, permService *mocks.MockPermissionService) {
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "user has required role",
			setupContext: func(c *gin.Context) {
				c.Set("user_claims", &dto.Claims{
					UserID: primitive.NewObjectID(),
					Roles:  []string{"role-123"},
				})
			},
			config: AuthorizationConfig{
				RequiredRoles: []string{"role-123"},
			},
			setupMocks: func(roleService *mocks.MockRoleService, permService *mocks.MockPermissionService) {
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "user missing required role",
			setupContext: func(c *gin.Context) {
				c.Set("user_claims", &dto.Claims{
					UserID: primitive.NewObjectID(),
					Roles:  []string{"role-456"},
				})
			},
			config: AuthorizationConfig{
				RequiredRoles: []string{"role-123"},
			},
			setupMocks: func(roleService *mocks.MockRoleService, permService *mocks.MockPermissionService) {
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "user has required permission",
			setupContext: func(c *gin.Context) {
				roleID := primitive.NewObjectID()
				c.Set("user_claims", &dto.Claims{
					UserID: primitive.NewObjectID(),
					Roles:  []string{roleID.Hex()},
				})
			},
			config: AuthorizationConfig{
				RequiredPermissions: []string{"perm-123"},
			},
			setupMocks: func(roleService *mocks.MockRoleService, permService *mocks.MockPermissionService) {
				role := &model.Role{
					Permissions: []string{"perm-123"},
				}
				roleService.On("FindByID", mock.Anything, mock.AnythingOfType("primitive.ObjectID")).Return(role, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "user missing required permission",
			setupContext: func(c *gin.Context) {
				roleID := primitive.NewObjectID()
				c.Set("user_claims", &dto.Claims{
					UserID: primitive.NewObjectID(),
					Roles:  []string{roleID.Hex()},
				})
			},
			config: AuthorizationConfig{
				RequiredPermissions: []string{"perm-123"},
			},
			setupMocks: func(roleService *mocks.MockRoleService, permService *mocks.MockPermissionService) {
				role := &model.Role{
					Permissions: []string{"perm-456"},
				}
				roleService.On("FindByID", mock.Anything, mock.AnythingOfType("primitive.ObjectID")).Return(role, nil).Once()
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "user has all required permissions",
			setupContext: func(c *gin.Context) {
				roleID := primitive.NewObjectID()
				c.Set("user_claims", &dto.Claims{
					UserID: primitive.NewObjectID(),
					Roles:  []string{roleID.Hex()},
				})
			},
			config: AuthorizationConfig{
				RequiredPermissions: []string{"perm-123", "perm-456"},
				RequireAllPermissions: true,
			},
			setupMocks: func(roleService *mocks.MockRoleService, permService *mocks.MockPermissionService) {
				role := &model.Role{
					Permissions: []string{"perm-123", "perm-456"},
				}
				roleService.On("FindByID", mock.Anything, mock.AnythingOfType("primitive.ObjectID")).Return(role, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "user missing one of required permissions",
			setupContext: func(c *gin.Context) {
				roleID := primitive.NewObjectID()
				c.Set("user_claims", &dto.Claims{
					UserID: primitive.NewObjectID(),
					Roles:  []string{roleID.Hex()},
				})
			},
			config: AuthorizationConfig{
				RequiredPermissions: []string{"perm-123", "perm-456"},
				RequireAllPermissions: true,
			},
			setupMocks: func(roleService *mocks.MockRoleService, permService *mocks.MockPermissionService) {
				role := &model.Role{
					Permissions: []string{"perm-123"},
				}
				roleService.On("FindByID", mock.Anything, mock.AnythingOfType("primitive.ObjectID")).Return(role, nil).Once()
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "invalid role ID skipped",
			setupContext: func(c *gin.Context) {
				c.Set("user_claims", &dto.Claims{
					UserID: primitive.NewObjectID(),
					Roles:  []string{"invalid-role-id"},
				})
			},
			config: AuthorizationConfig{
				RequiredPermissions: []string{"perm-123"},
			},
			setupMocks: func(roleService *mocks.MockRoleService, permService *mocks.MockPermissionService) {
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "role not found skipped",
			setupContext: func(c *gin.Context) {
				roleID := primitive.NewObjectID()
				c.Set("user_claims", &dto.Claims{
					UserID: primitive.NewObjectID(),
					Roles:  []string{roleID.Hex()},
				})
			},
			config: AuthorizationConfig{
				RequiredPermissions: []string{"perm-123"},
			},
			setupMocks: func(roleService *mocks.MockRoleService, permService *mocks.MockPermissionService) {
				roleService.On("FindByID", mock.Anything, mock.AnythingOfType("primitive.ObjectID")).Return(nil, nil).Once()
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "multiple roles with permissions",
			setupContext: func(c *gin.Context) {
				roleID1 := primitive.NewObjectID()
				roleID2 := primitive.NewObjectID()
				c.Set("user_claims", &dto.Claims{
					UserID: primitive.NewObjectID(),
					Roles:  []string{roleID1.Hex(), roleID2.Hex()},
				})
			},
			config: AuthorizationConfig{
				RequiredPermissions: []string{"perm-123"},
			},
			setupMocks: func(roleService *mocks.MockRoleService, permService *mocks.MockPermissionService) {
				role1 := &model.Role{
					Permissions: []string{"perm-456"},
				}
				role2 := &model.Role{
					Permissions: []string{"perm-123"},
				}
				roleService.On("FindByID", mock.Anything, mock.AnythingOfType("primitive.ObjectID")).Return(role1, nil).Once()
				roleService.On("FindByID", mock.Anything, mock.AnythingOfType("primitive.ObjectID")).Return(role2, nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roleService := mocks.NewMockRoleService(t)
			permService := mocks.NewMockPermissionService(t)
			tt.setupMocks(roleService, permService)

			router := gin.New()
			router.Use(RequestID())
			router.Use(func(c *gin.Context) {
				tt.setupContext(c)
				c.Next()
			})
			router.Use(RequireAuthorization(tt.config, roleService, permService))
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			roleService.AssertExpectations(t)
			permService.AssertExpectations(t)
		})
	}
}
