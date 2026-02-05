package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestErrorHandler(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		setupHandler   func(*gin.Engine)
		expectedStatus int
		expectedBody   string
		mustContain    []string
	}{
		{
			name: "handles gin context errors",
			path: "/error",
			setupHandler: func(router *gin.Engine) {
				router.GET("/error", func(c *gin.Context) {
					_ = c.Error(errors.New("test error"))
				})
			},
			expectedStatus: http.StatusInternalServerError,
			mustContain:    []string{"internal_error", "An unexpected error occurred"},
		},
		{
			name: "does nothing when no errors",
			path: "/ok",
			setupHandler: func(router *gin.Engine) {
				router.GET("/ok", func(c *gin.Context) {
					c.String(http.StatusOK, "ok")
				})
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(RequestID(), ErrorHandler())
			tt.setupHandler(router)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
			for _, substr := range tt.mustContain {
				assert.Contains(t, w.Body.String(), substr)
			}
		})
	}
}
