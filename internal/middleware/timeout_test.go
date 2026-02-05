package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDefaultTimeoutConfig(t *testing.T) {
	cfg := DefaultTimeoutConfig()

	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, "Request timeout", cfg.ErrorMessage)
}

func TestTimeout_RequestCompletesInTime(t *testing.T) {
	tests := []struct {
		name          string
		timeout       time.Duration
		handlerDelay  time.Duration
		wantStatus    int
	}{
		{
			name:         "fast request completes",
			timeout:      time.Second,
			handlerDelay: 10 * time.Millisecond,
			wantStatus:   http.StatusOK,
		},
		{
			name:         "zero delay request completes",
			timeout:      time.Second,
			handlerDelay: 0,
			wantStatus:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()

			cfg := TimeoutConfig{
				Timeout:      tt.timeout,
				ErrorMessage: "timeout",
			}

			router.Use(Timeout(cfg))
			router.GET("/test", func(c *gin.Context) {
				if tt.handlerDelay > 0 {
					time.Sleep(tt.handlerDelay)
				}
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestTimeoutWithDuration(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "1 second timeout",
			timeout: time.Second,
		},
		{
			name:    "5 second timeout",
			timeout: 5 * time.Second,
		},
		{
			name:    "100ms timeout",
			timeout: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()

			router.Use(TimeoutWithDuration(tt.timeout))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestTimeout_ContextHasDeadline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	cfg := TimeoutConfig{
		Timeout:      time.Second,
		ErrorMessage: "timeout",
	}

	hasDeadline := false
	router.Use(Timeout(cfg))
	router.GET("/test", func(c *gin.Context) {
		_, hasDeadline = c.Request.Context().Deadline()
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.True(t, hasDeadline, "context should have deadline set")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTimeout_FastRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	cfg := TimeoutConfig{
		Timeout:      100 * time.Millisecond,
		ErrorMessage: "timeout",
	}

	router.Use(Timeout(cfg))
	router.GET("/fast", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Multiple fast requests should all succeed
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/fast", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}
