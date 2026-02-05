package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		headerValue    string
		validate       func(*testing.T, string)
	}{
		{
			name:        "generates new request ID when not provided",
			headerValue: "",
			validate: func(t *testing.T, id string) {
				assert.NotEmpty(t, id)
				_, err := uuid.Parse(id)
				assert.NoError(t, err)
			},
		},
		{
			name:        "uses provided request ID from header",
			headerValue: "custom-request-id-123",
			validate: func(t *testing.T, id string) {
				assert.Equal(t, "custom-request-id-123", id)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(RequestID())
			router.GET("/test", func(c *gin.Context) {
				id := GetRequestID(c)
				c.String(http.StatusOK, id)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.headerValue != "" {
				req.Header.Set(RequestIDHeader, tt.headerValue)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			requestID := w.Body.String()
			assert.NotEmpty(t, requestID)
			assert.Equal(t, requestID, w.Header().Get(RequestIDHeader))

			if tt.validate != nil {
				tt.validate(t, requestID)
			}
		})
	}
}

func TestGetRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedID     string
		expectedEmpty  bool
	}{
		{
			name: "returns empty string when not set",
			setupContext: func(c *gin.Context) {
				// No setup - request ID not set
			},
			expectedEmpty: true,
		},
		{
			name: "returns request ID when set",
			setupContext: func(c *gin.Context) {
				c.Set(string(RequestIDKey), "test-id-123")
			},
			expectedID: "test-id-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

			tt.setupContext(c)

			id := GetRequestID(c)
			if tt.expectedEmpty {
				assert.Empty(t, id)
			} else {
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}
