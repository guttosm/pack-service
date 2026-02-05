package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/guttosm/pack-service/internal/circuitbreaker"
)

func TestHealthHandler_Readiness(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupHandler   func() *HealthHandler
		expectedStatus int
	}{
		{
			name: "readiness check no checkers",
			setupHandler: func() *HealthHandler {
				return NewHealthHandler()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "readiness check with healthy circuit breaker",
			setupHandler: func() *HealthHandler {
				handler := NewHealthHandler()
				cb := circuitbreaker.New(circuitbreaker.DefaultConfig())
				handler.RegisterCircuitBreaker("test", cb)
				return handler
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			handler := tt.setupHandler()
			handler.Register(router)

			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHealthHandler_RegisterCircuitBreaker(t *testing.T) {
	tests := []struct {
		name           string
		circuitBreaker *circuitbreaker.CircuitBreaker
	}{
		{
			name:           "register circuit breaker",
			circuitBreaker: circuitbreaker.New(circuitbreaker.DefaultConfig()),
		},
		{
			name:           "register nil circuit breaker",
			circuitBreaker: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			handler := NewHealthHandler()
			if tt.circuitBreaker != nil {
				handler.RegisterCircuitBreaker("test", tt.circuitBreaker)
			}

			router.GET("/readyz", handler.Readiness)

			req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}
