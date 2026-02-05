package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestNewRouter(t *testing.T) {
	calculator := service.NewPackCalculatorService()
	handler := NewHandler(calculator, nil) // nil means pack sizes from MongoDB are disabled
	healthHandler := NewHealthHandler()

	tests := []struct {
		name string
		cfg  RouterConfig
		test func(*testing.T, *gin.Engine)
	}{
		{
			name: "creates router with default config",
			cfg:  DefaultRouterConfig(),
			test: func(t *testing.T, router *gin.Engine) {
				assert.NotNil(t, router)
			},
		},
		{
			name: "creates router with auth enabled",
			cfg: RouterConfig{
				RateLimit:  100,
				RateWindow: time.Minute,
				EnableAuth: true,
				APIKeys:    map[string]bool{"test-key": true},
			},
			test: func(t *testing.T, router *gin.Engine) {
				assert.NotNil(t, router)
			},
		},
		{
			name: "creates router with idempotency enabled",
			cfg: RouterConfig{
				RateLimit:       100,
				RateWindow:      time.Minute,
				EnableIdempotency: true,
			},
			test: func(t *testing.T, router *gin.Engine) {
				assert.NotNil(t, router)
			},
		},
		{
			name: "creates router with rate limiting",
			cfg: RouterConfig{
				RateLimit:  5,
				RateWindow: time.Second,
			},
			test: func(t *testing.T, router *gin.Engine) {
				assert.NotNil(t, router)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(handler, healthHandler, tt.cfg)
			if tt.test != nil {
				tt.test(t, router)
			}
		})
	}
}

func TestRouter_Endpoints(t *testing.T) {
	calculator := service.NewPackCalculatorService()
	handler := NewHandler(calculator, nil) // nil means pack sizes from MongoDB are disabled
	healthHandler := NewHealthHandler()
	router := NewRouter(handler, healthHandler, DefaultRouterConfig())

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "healthz endpoint",
			method:         http.MethodGet,
			path:           "/healthz",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "readyz endpoint",
			method:         http.MethodGet,
			path:           "/readyz",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "metrics endpoint",
			method:         http.MethodGet,
			path:           "/metrics",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "swagger endpoint",
			method:         http.MethodGet,
			path:           "/swagger/index.html",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "calculate endpoint",
			method:         http.MethodPost,
			path:           "/api/calculate",
			expectedStatus: http.StatusBadRequest, // Missing body
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
