//go:build integration

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/config"
	"github.com/guttosm/pack-service/internal/circuitbreaker"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/repository"
	"github.com/guttosm/pack-service/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// dbConnections stores MongoDB connections to prevent garbage collection
var dbConnections = make(map[string]*repository.MongoDB)
var dbConnectionsMutex sync.Mutex

func setupAuthIntegrationRouter(dbName string) *gin.Engine {
	gin.SetMode(gin.TestMode)

	uri := getSharedContainerURI()
	
	// Check if we already have a connection for this database
	dbConnectionsMutex.Lock()
	db, exists := dbConnections[dbName]
	dbConnectionsMutex.Unlock()
	
	if !exists {
		var err error
		db, err = repository.NewMongoDB(uri, dbName)
		if err != nil {
			panic(err)
		}
		// Store the connection to prevent garbage collection
		dbConnectionsMutex.Lock()
		dbConnections[dbName] = db
		dbConnectionsMutex.Unlock()
	}

	userRepo := repository.NewUserRepository(db.Database)
	roleRepo := repository.NewRoleRepository(db.Database)
	permissionRepo := repository.NewPermissionRepository(db.Database)
	tokenRepo := repository.NewTokenRepository(db.Database)

	initCtx, initCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer initCancel()
	
	permissions := []*model.Permission{
		{Name: "packs:read", Description: "Read pack calculations", Resource: "packs", Action: "read", Active: true},
		{Name: "packs:write", Description: "Create pack calculations", Resource: "packs", Action: "write", Active: true},
		{Name: "users:read", Description: "Read users", Resource: "users", Action: "read", Active: true},
		{Name: "users:write", Description: "Create/update users", Resource: "users", Action: "write", Active: true},
		{Name: "users:delete", Description: "Delete users", Resource: "users", Action: "delete", Active: true},
		{Name: "roles:read", Description: "Read roles", Resource: "roles", Action: "read", Active: true},
		{Name: "roles:write", Description: "Create/update roles", Resource: "roles", Action: "write", Active: true},
	}
	
	permissionIDs := make([]string, 0, len(permissions))
	for _, perm := range permissions {
		existing, _ := permissionRepo.FindByResourceAndAction(initCtx, perm.Resource, perm.Action)
		if existing == nil {
			if err := permissionRepo.Create(initCtx, perm); err != nil {
				panic("failed to create permission: " + err.Error())
			}
			// After Create, perm.ID should be set by the repository
			if perm.ID.IsZero() {
				// If ID still not set, fetch the created permission
				created, _ := permissionRepo.FindByResourceAndAction(initCtx, perm.Resource, perm.Action)
				if created != nil {
					perm.ID = created.ID
				} else {
					perm.ID = primitive.NewObjectID()
				}
			}
		} else {
			perm.ID = existing.ID
		}
		permissionIDs = append(permissionIDs, perm.ID.Hex())
	}
	
	roles := []*model.Role{
		{
			Name:        "user",
			Description: "Standard user role",
			Permissions: []string{permissionIDs[0], permissionIDs[1]},
			Active:      true,
		},
		{
			Name:        "admin",
			Description: "Administrator role with full access",
			Permissions: permissionIDs,
			Active:      true,
		},
	}
	
	for _, role := range roles {
		existing, _ := roleRepo.FindByName(initCtx, role.Name)
		if existing == nil {
			if err := roleRepo.Create(initCtx, role); err != nil {
				panic("failed to create role: " + err.Error())
			}
		}
	}

	authConfig := config.AuthConfig{
		JWTSecretKey:     "test-secret-key",
		JWTRefreshSecret: "test-refresh-secret-key",
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  7 * 24 * time.Hour,
	}
	authService := service.NewAuthService(userRepo, roleRepo, tokenRepo, authConfig)

	logsRepo := repository.NewLogsRepository(db)
	logsCB := circuitbreaker.New(circuitbreaker.DefaultConfig())
	logsRepoWithCB := repository.NewLogsRepositoryWithCircuitBreaker(logsRepo, logsCB)
	loggingService := service.NewLoggingService(logsRepoWithCB)

	authHandler := NewAuthHandler(authService)
	healthHandler := NewHealthHandler()

	cfg := RouterConfig{
		RateLimit:      100,
		RateWindow:     time.Minute,
		EnableAuth:     false,
		LoggingService: loggingService,
	}

	router := NewRouter(nil, healthHandler, cfg)

	api := router.Group("/api")
	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.RefreshToken)
	auth.POST("/logout", authHandler.Logout)

	return router
}

func TestAuthHandler_Login_Integration(t *testing.T) {
	t.Parallel()

	t.Run("register then login", func(t *testing.T) {
		// Use shared container with unique database name for this subtest
		dbName := sanitizeDBNameForHTTP(t.Name())
		router := setupAuthIntegrationRouter(dbName)
		registerBody := dto.RegisterRequest{
			Email:    "test@example.com",
			Username: "testuser",
			Password: "password123",
			Name:     "Test User",
		}
		registerBodyBytes, _ := json.Marshal(registerBody)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(registerBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		loginBody := dto.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}
		loginBodyBytes, _ := json.Marshal(loginBody)

		req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			// Log the error response for debugging
			t.Logf("Login failed with status %d, body: %s", w.Code, w.Body.String())
			var errorResponse dto.ErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &errorResponse); err == nil {
				t.Logf("Error response: %+v", errorResponse)
			}
		}
		require.Equal(t, http.StatusOK, w.Code, "Login should succeed after registration")

		var response dto.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		dataBytes, _ := json.Marshal(response.Data)
		var loginResponse dto.LoginResponse
		err = json.Unmarshal(dataBytes, &loginResponse)
		require.NoError(t, err)
		assert.NotEmpty(t, loginResponse.Token)
		assert.NotEmpty(t, loginResponse.RefreshToken)
		assert.Equal(t, "test@example.com", loginResponse.User.Email)
	})

	t.Run("login with invalid credentials", func(t *testing.T) {
		// Use shared container with unique database name for this subtest
		dbName := sanitizeDBNameForHTTP(t.Name())
		router := setupAuthIntegrationRouter(dbName)
		
		loginBody := dto.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "wrongpassword",
		}
		loginBodyBytes, _ := json.Marshal(loginBody)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusUnauthorized || w.Code == http.StatusInternalServerError)
	})
}

func TestAuthHandler_Register_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful registration", func(t *testing.T) {
		// Use shared container with unique database name for this subtest
		dbName := sanitizeDBNameForHTTP(t.Name())
		router := setupAuthIntegrationRouter(dbName)
		registerBody := dto.RegisterRequest{
			Email:    "newuser@example.com",
			Username: "newuser",
			Password: "password123",
			Name:     "New User",
		}
		bodyBytes, _ := json.Marshal(registerBody)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response dto.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		dataBytes, _ := json.Marshal(response.Data)
		var loginResponse dto.LoginResponse
		err = json.Unmarshal(dataBytes, &loginResponse)
		require.NoError(t, err)
		assert.NotEmpty(t, loginResponse.Token)
		assert.NotEmpty(t, loginResponse.RefreshToken)
	})

	t.Run("duplicate email registration", func(t *testing.T) {
		// Use shared container with unique database name for this subtest
		dbName := sanitizeDBNameForHTTP(t.Name())
		router := setupAuthIntegrationRouter(dbName)
		registerBody := dto.RegisterRequest{
			Email:    "duplicate@example.com",
			Username: "duplicateuser",
			Password: "password123",
			Name:     "First User",
		}
		bodyBytes, _ := json.Marshal(registerBody)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		req = httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

func TestAuthHandler_RefreshToken_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful token refresh", func(t *testing.T) {
		// Use shared container with unique database name for this subtest
		dbName := sanitizeDBNameForHTTP(t.Name())
		router := setupAuthIntegrationRouter(dbName)
		registerBody := dto.RegisterRequest{
			Email:    "refreshtest@example.com",
			Username: "refreshtest",
			Password: "password123",
			Name:     "Refresh Test",
		}
		registerBodyBytes, _ := json.Marshal(registerBody)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(registerBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		var registerResponse dto.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &registerResponse)
		require.NoError(t, err)

		dataBytes, _ := json.Marshal(registerResponse.Data)
		var loginResponse dto.LoginResponse
		err = json.Unmarshal(dataBytes, &loginResponse)
		require.NoError(t, err)

		// Wait for at least 1 second to ensure JWT timestamps differ
		time.Sleep(time.Second)

		// Refresh token is passed in X-Refresh-Token header, not body
		req = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Refresh-Token", loginResponse.RefreshToken)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var refreshResponse dto.SuccessResponse
		err = json.Unmarshal(w.Body.Bytes(), &refreshResponse)
		require.NoError(t, err)

		dataBytes, _ = json.Marshal(refreshResponse.Data)
		var newTokenPair dto.LoginResponse
		err = json.Unmarshal(dataBytes, &newTokenPair)
		require.NoError(t, err)
		assert.NotEmpty(t, newTokenPair.Token)
		assert.NotEmpty(t, newTokenPair.RefreshToken)
		assert.NotEqual(t, loginResponse.Token, newTokenPair.Token)
	})

	t.Run("refresh with invalid token", func(t *testing.T) {
		// Use shared container with unique database name for this subtest
		dbName := sanitizeDBNameForHTTP(t.Name())
		router := setupAuthIntegrationRouter(dbName)
		
		// Refresh token is passed in X-Refresh-Token header, not body
		req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Refresh-Token", "invalid-refresh-token")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestAuthHandler_Logout_Integration(t *testing.T) {
	t.Parallel()

	t.Run("successful logout", func(t *testing.T) {
		// Use shared container with unique database name for this subtest
		dbName := sanitizeDBNameForHTTP(t.Name())
		router := setupAuthIntegrationRouter(dbName)
		registerBody := dto.RegisterRequest{
			Email:    "logouttest@example.com",
			Username: "logouttest",
			Password: "password123",
			Name:     "Logout Test",
		}
		registerBodyBytes, _ := json.Marshal(registerBody)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(registerBodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		var registerResponse dto.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &registerResponse)
		require.NoError(t, err)

		dataBytes, _ := json.Marshal(registerResponse.Data)
		var loginResponse dto.LoginResponse
		err = json.Unmarshal(dataBytes, &loginResponse)
		require.NoError(t, err)

		// JWT tokens are passed in headers, not body - access token in Authorization header, refresh token in X-Refresh-Token header
		req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+loginResponse.Token)
		req.Header.Set("X-Refresh-Token", loginResponse.RefreshToken)
		w = httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
