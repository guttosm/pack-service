package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/circuitbreaker"
)

// HealthChecker defines the interface for health check operations.
type HealthChecker interface {
	Check() error
}

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	checkers        map[string]HealthChecker
	circuitBreakers map[string]*circuitbreaker.CircuitBreaker
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{
		checkers:        make(map[string]HealthChecker),
		circuitBreakers: make(map[string]*circuitbreaker.CircuitBreaker),
	}
}

// RegisterCircuitBreaker registers a circuit breaker for health monitoring.
func (h *HealthHandler) RegisterCircuitBreaker(name string, cb *circuitbreaker.CircuitBreaker) {
	h.circuitBreakers[name] = cb
}

// Register registers health endpoints on the router.
func (h *HealthHandler) Register(router *gin.Engine) {
	router.GET("/healthz", h.Liveness)
	router.GET("/readyz", h.Readiness)
}

// Liveness handles the liveness probe endpoint.
// @Summary     Liveness probe
// @Description Returns OK if the service is running. Used by Kubernetes and other orchestration platforms to determine if the service should be restarted.
// @Tags        Health
// @Produce     json
// @Success     200 {object} map[string]string "Service is alive"
// @ExampleResponse 200 {"status": "ok"}
// @Router      /healthz [get]
//
// Metrics endpoint is available at /metrics for Prometheus scraping.
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readiness handles the readiness probe endpoint.
// @Summary     Readiness probe
// @Description Returns OK if all dependencies are healthy and the service is ready to accept traffic. Used by load balancers and orchestration platforms.
// @Tags        Health
// @Produce     json
// @Success     200 {object} map[string]interface{} "Service is ready"
// @Failure     503 {object} map[string]interface{} "Service is not ready"
// @ExampleResponse 200 {"status": "ok", "checks": {"service": "ok"}}
// @ExampleResponse 503 {"status": "degraded", "checks": {"database": "connection failed"}}
// @Router      /readyz [get]
func (h *HealthHandler) Readiness(c *gin.Context) {
	status := http.StatusOK
	checks := make(map[string]interface{})

	// Check registered health checkers
	for name, checker := range h.checkers {
		if err := checker.Check(); err != nil {
			checks[name] = err.Error()
			status = http.StatusServiceUnavailable
		} else {
			checks[name] = "ok"
		}
	}

	// Check circuit breakers
	for name, cb := range h.circuitBreakers {
		stats := cb.GetStats()
		checks[name+"_circuit"] = stats.State
		if !stats.IsHealthy {
			status = http.StatusServiceUnavailable
		}
	}

	if len(checks) == 0 {
		checks["service"] = "ok"
	}

	c.JSON(status, gin.H{
		"status": map[bool]string{true: "ok", false: "degraded"}[status == http.StatusOK],
		"checks": checks,
	})
}
