package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAPIKeyAuth(t *testing.T) {
	validKeys := map[string]bool{"valid-key-123": true, "another-valid-key": true}

	tests := []struct {
		name           string
		validKeys      map[string]bool
		setupRequest   func(*http.Request)
		expectedStatus int
		expectedBody    string
	}{
		{
			name:           "allows request with valid API key in header",
			validKeys:      validKeys,
			setupRequest:   func(req *http.Request) { req.Header.Set(APIKeyHeader, "valid-key-123") },
			expectedStatus: http.StatusOK,
			expectedBody:   "ok",
		},
		{
			name:           "allows request with valid API key in query",
			validKeys:      validKeys,
			setupRequest:   func(req *http.Request) { req.URL.RawQuery = "api_key=valid-key-123" },
			expectedStatus: http.StatusOK,
			expectedBody:   "ok",
		},
		{
			name:           "rejects request without API key",
			validKeys:      validKeys,
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "API key is required",
		},
		{
			name:           "rejects request with invalid API key",
			validKeys:      validKeys,
			setupRequest:   func(req *http.Request) { req.Header.Set(APIKeyHeader, "invalid-key") },
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid API key",
		},
		{
			name:           "allows all requests when no keys configured",
			validKeys:      nil,
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusOK,
			expectedBody:   "ok",
		},
		{
			name:           "allows all requests when empty keys map",
			validKeys:      map[string]bool{},
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusOK,
			expectedBody:   "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(APIKeyAuth(tt.validKeys))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}
