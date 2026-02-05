package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Tests for AuthRoutes

func TestNewAuthRoutes(t *testing.T) {
	mockAuthService := mocks.NewMockAuthService(t)

	routes := NewAuthRoutes(mockAuthService)

	assert.NotNil(t, routes)
	assert.NotNil(t, routes.handler)
}

func TestAuthRoutes_RegisterPublicRoutes(t *testing.T) {
	mockAuthService := mocks.NewMockAuthService(t)
	routes := NewAuthRoutes(mockAuthService)

	router := gin.New()
	api := router.Group("/api")
	routes.RegisterPublicRoutes(api)

	// Verify routes are registered by checking if they respond
	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/auth/login"},
		{http.MethodPost, "/api/auth/register"},
		{http.MethodPost, "/api/auth/refresh"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Should not return 404 - route exists
			assert.NotEqual(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestAuthRoutes_RegisterProtectedRoutes(t *testing.T) {
	mockAuthService := mocks.NewMockAuthService(t)
	routes := NewAuthRoutes(mockAuthService)

	router := gin.New()
	api := router.Group("/api")

	cfg := &RouterConfig{
		RateLimit:  100,
		RateWindow: time.Minute,
	}

	routes.RegisterProtectedRoutes(api, cfg)

	// Verify logout route is registered
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not return 404 - route exists (will fail auth but that's expected)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestAuthRoutes_GetProtectedGroup(t *testing.T) {
	tests := []struct {
		name       string
		rateLimit  int
		rateWindow time.Duration
	}{
		{
			name:       "with rate limiting",
			rateLimit:  100,
			rateWindow: time.Minute,
		},
		{
			name:       "without rate limiting",
			rateLimit:  0,
			rateWindow: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuthService := mocks.NewMockAuthService(t)
			routes := NewAuthRoutes(mockAuthService)

			router := gin.New()
			api := router.Group("/api")

			cfg := &RouterConfig{
				RateLimit:  tt.rateLimit,
				RateWindow: tt.rateWindow,
			}

			protected := routes.GetProtectedGroup(api, cfg)

			assert.NotNil(t, protected)
		})
	}
}

// Tests for PackRoutes

func TestNewPackRoutes(t *testing.T) {
	t.Run("with pack sizes service", func(t *testing.T) {
		mockCalc := mocks.NewMockPackCalculator(t)
		mockPackSizes := mocks.NewMockPackSizesService(t)

		routes := NewPackRoutes(mockCalc, mockPackSizes)

		assert.NotNil(t, routes)
		assert.NotNil(t, routes.handler)
		assert.NotNil(t, routes.packSizesHandler)
	})

	t.Run("without pack sizes service", func(t *testing.T) {
		mockCalc := mocks.NewMockPackCalculator(t)

		routes := NewPackRoutes(mockCalc, nil)

		assert.NotNil(t, routes)
		assert.NotNil(t, routes.handler)
		assert.Nil(t, routes.packSizesHandler)
	})
}

func TestPackRoutes_RegisterPublicRoutes(t *testing.T) {
	mockCalc := mocks.NewMockPackCalculator(t)

	// Test without pack sizes service to avoid mock setup complexity
	routes := NewPackRoutes(mockCalc, nil)

	router := gin.New()
	api := router.Group("/api")
	routes.RegisterPublicRoutes(api)

	// Verify calculate route is registered
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not return 404 - route exists
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestPackRoutes_RegisterPublicRoutes_WithoutPackSizesService(t *testing.T) {
	mockCalc := mocks.NewMockPackCalculator(t)

	routes := NewPackRoutes(mockCalc, nil)

	router := gin.New()
	api := router.Group("/api")
	routes.RegisterPublicRoutes(api)

	// Calculate route should exist
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)

	// Pack sizes routes should NOT exist
	req2 := httptest.NewRequest(http.MethodGet, "/api/pack-sizes", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusNotFound, w2.Code)
}

func TestPackRoutes_GetHandler(t *testing.T) {
	mockCalc := mocks.NewMockPackCalculator(t)
	routes := NewPackRoutes(mockCalc, nil)

	handler := routes.GetHandler()

	assert.NotNil(t, handler)
	assert.Equal(t, routes.handler, handler)
}

func TestPackRoutes_RegisterProtectedRoutes(t *testing.T) {
	mockCalc := mocks.NewMockPackCalculator(t)

	// Test without pack sizes service to avoid mock setup complexity
	routes := NewPackRoutes(mockCalc, nil)

	router := gin.New()
	api := router.Group("/api")

	cfg := &RouterConfig{}

	routes.RegisterProtectedRoutes(api, cfg)

	// Verify calculate route is registered
	req := httptest.NewRequest(http.MethodPost, "/api/calculate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not return 404 - route exists
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestPackRoutes_GetPermissionIDs(t *testing.T) {
	mockCalc := mocks.NewMockPackCalculator(t)
	routes := NewPackRoutes(mockCalc, nil)

	cfg := &RouterConfig{
		PermissionService: nil,
	}

	readID, writeID := routes.getPermissionIDs(cfg)

	assert.Equal(t, "", readID)
	assert.Equal(t, "", writeID)
}
