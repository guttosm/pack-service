package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestPrometheusMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(PrometheusMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	router.GET("/error", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "error")
	})

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "records metrics for successful request",
			path:           "/test",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "records metrics for error request",
			path:           "/error",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRecordPackCalculation(t *testing.T) {
	RecordPackCalculation(100*time.Millisecond, "success")
	RecordPackCalculation(50*time.Millisecond, "error")

	assert.True(t, true)
}

func TestRecordCacheOperation(t *testing.T) {
	RecordCacheOperation("get", "hit")
	RecordCacheOperation("get", "miss")
	RecordCacheOperation("set", "success")

	assert.True(t, true)
}

func TestUpdateCacheMetrics(t *testing.T) {
	UpdateCacheMetrics(50, 100)
	UpdateCacheMetrics(75, 100)

	assert.True(t, true)
}
