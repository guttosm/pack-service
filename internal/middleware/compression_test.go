package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCompression(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		acceptEncoding string
		expectCompressed bool
	}{
		{
			name:            "compresses when Accept-Encoding includes gzip",
			acceptEncoding:  "gzip",
			expectCompressed: true,
		},
		{
			name:            "compresses when Accept-Encoding includes gzip, deflate",
			acceptEncoding:  "gzip, deflate",
			expectCompressed: true,
		},
		{
			name:            "does not compress when no Accept-Encoding",
			acceptEncoding:  "",
			expectCompressed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(Compression())
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "test response")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			if tt.expectCompressed {
				assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
			}
		})
	}
}
