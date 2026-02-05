// Package metrics provides Prometheus metrics collection for the pack service.
package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestDuration tracks HTTP request duration by method, path, and status code.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status_code"},
	)

	// HTTPRequestTotal tracks total HTTP requests by method, path, and status code.
	HTTPRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status_code"},
	)

	// PackCalculationsTotal tracks total pack calculations.
	PackCalculationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pack_calculations_total",
			Help: "Total number of pack calculations",
		},
		[]string{"status"},
	)

	// PackCalculationDuration tracks pack calculation duration.
	PackCalculationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "pack_calculation_duration_seconds",
			Help:    "Pack calculation duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
	)

	// CacheOperationsTotal tracks cache operations.
	CacheOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation", "result"},
	)

	// CacheSize tracks current cache size.
	CacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cache_size",
			Help: "Current cache size",
		},
	)

	// CacheCapacity tracks cache capacity.
	CacheCapacity = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cache_capacity",
			Help: "Cache capacity",
		},
	)
)

// PrometheusMiddleware returns a Gin middleware that collects HTTP metrics.
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		c.Next()

		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		HTTPRequestDuration.WithLabelValues(method, path, statusCode).Observe(duration)
		HTTPRequestTotal.WithLabelValues(method, path, statusCode).Inc()
	}
}

// RecordPackCalculation records metrics for a pack calculation.
func RecordPackCalculation(duration time.Duration, status string) {
	PackCalculationDuration.Observe(duration.Seconds())
	PackCalculationsTotal.WithLabelValues(status).Inc()
}

// RecordCacheOperation records metrics for a cache operation.
func RecordCacheOperation(operation, result string) {
	CacheOperationsTotal.WithLabelValues(operation, result).Inc()
}

// UpdateCacheMetrics updates cache size and capacity metrics.
func UpdateCacheMetrics(size, capacity int) {
	CacheSize.Set(float64(size))
	CacheCapacity.Set(float64(capacity))
}
