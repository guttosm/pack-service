package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORS(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		origin         string
		expectedStatus int
		checkHeaders   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "OPTIONS preflight request",
			method:         http.MethodOptions,
			origin:         "https://example.com",
			expectedStatus: http.StatusNoContent,
			checkHeaders: func(t *testing.T, w *httptest.ResponseRecorder) {
				// CORS middleware sets headers based on origin
				origin := w.Header().Get("Access-Control-Allow-Origin")
				assert.NotEmpty(t, origin)
			},
		},
		{
			name:           "GET request with origin",
			method:         http.MethodGet,
			origin:         "https://example.com",
			expectedStatus: http.StatusOK,
			checkHeaders: func(t *testing.T, w *httptest.ResponseRecorder) {
				// CORS middleware sets headers based on origin
				origin := w.Header().Get("Access-Control-Allow-Origin")
				assert.NotEmpty(t, origin)
			},
		},
		{
			name:           "POST request without origin",
			method:         http.MethodPost,
			origin:         "",
			expectedStatus: http.StatusOK,
			checkHeaders: func(t *testing.T, w *httptest.ResponseRecorder) {
				// CORS headers may or may not be present
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()

			router.Use(CORS())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})
			router.POST("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})
			router.OPTIONS("/test", func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkHeaders != nil {
				tt.checkHeaders(t, w)
			}
		})
	}
}
