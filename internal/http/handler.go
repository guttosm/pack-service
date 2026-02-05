package http

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/i18n"
	"github.com/guttosm/pack-service/internal/metrics"
	"github.com/guttosm/pack-service/internal/middleware"
	"github.com/guttosm/pack-service/internal/service"
)

// packSizesCache provides thread-safe caching of pack sizes.
type packSizesCache struct {
	sizes     atomic.Value // holds []int
	expiresAt atomic.Value // holds time.Time
	mu        sync.Mutex
	ttl       time.Duration
}

// newPackSizesCache creates a new pack sizes cache with the given TTL.
func newPackSizesCache(ttl time.Duration) *packSizesCache {
	c := &packSizesCache{ttl: ttl}
	c.expiresAt.Store(time.Time{})
	return c
}

// get returns cached pack sizes if valid, or nil if cache is expired/empty.
func (c *packSizesCache) get() []int {
	if exp := c.expiresAt.Load(); exp != nil {
		if expiresAt, ok := exp.(time.Time); ok && time.Now().Before(expiresAt) {
			if sizes := c.sizes.Load(); sizes != nil {
				if s, ok := sizes.([]int); ok {
					return s
				}
			}
		}
	}
	return nil
}

// set stores pack sizes in the cache with TTL.
func (c *packSizesCache) set(sizes []int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring lock
	if exp := c.expiresAt.Load(); exp != nil {
		if expiresAt, ok := exp.(time.Time); ok && time.Now().Before(expiresAt) {
			return // Already cached by another goroutine
		}
	}

	c.sizes.Store(sizes)
	c.expiresAt.Store(time.Now().Add(c.ttl))
}

// invalidate clears the cache.
func (c *packSizesCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.expiresAt.Store(time.Time{})
}

// Handler provides HTTP handlers for pack calculation routes.
type Handler struct {
	calculator       service.PackCalculator
	packSizesService service.PackSizesService
	packSizesCache   *packSizesCache
}

// HandlerOption configures a Handler.
type HandlerOption func(*Handler)

// WithPackSizesCacheTTL sets the TTL for pack sizes caching.
func WithPackSizesCacheTTL(ttl time.Duration) HandlerOption {
	return func(h *Handler) {
		h.packSizesCache = newPackSizesCache(ttl)
	}
}

// NewHandler creates a new Handler instance.
func NewHandler(calculator service.PackCalculator, packSizesService service.PackSizesService, opts ...HandlerOption) *Handler {
	h := &Handler{
		calculator:       calculator,
		packSizesService: packSizesService,
		packSizesCache:   newPackSizesCache(30 * time.Second), // Default 30s cache
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// getPackSizes retrieves pack sizes from cache or database.
func (h *Handler) getPackSizes(ctx context.Context) []int {
	// Check cache first
	if sizes := h.packSizesCache.get(); sizes != nil {
		return sizes
	}

	// Cache miss - fetch from database
	if h.packSizesService == nil {
		return nil
	}

	// Use a timeout for database fetch
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	config, err := h.packSizesService.GetActive(ctx)
	if err != nil || config == nil || len(config.Sizes) == 0 {
		return nil
	}

	// Cache the result
	h.packSizesCache.set(config.Sizes)
	return config.Sizes
}

// InvalidatePackSizesCache invalidates the pack sizes cache.
// Call this when pack sizes are updated.
func (h *Handler) InvalidatePackSizesCache() {
	h.packSizesCache.invalidate()
}

// CalculatePacks handles POST /api/calculate requests.
//
// @Summary      Calculate packs for order
// @Description  Calculates the optimal number of packs needed to fulfill an order. The service uses dynamic programming to find the combination that minimizes total items while using the fewest number of packs. Supports idempotency via Idempotency-Key header.
// @Tags         Packs
// @Accept       json
// @Produce      json
// @Param        Idempotency-Key header string false "Idempotency key for request deduplication"
// @Param        request body dto.CalculatePacksRequest true "Order information"
// @Success      200 {object} dto.SuccessResponse "Successful calculation"
// @Failure      400 {object} dto.ErrorResponse "Bad request - invalid input"
// @Param        Authorization header string false "Bearer token (required if auth enabled)"
// @Failure      401 {object} dto.ErrorResponse "Unauthorized - missing or invalid JWT token"
// @Failure      403 {object} dto.ErrorResponse "Forbidden - insufficient permissions"
// @Failure      404 {object} dto.ErrorResponse "Not found - resource not found"
// @Failure      429 {object} dto.ErrorResponse "Too many requests - rate limit exceeded"
// @Failure      500 {object} dto.ErrorResponse "Internal server error"
// @Failure      502 {object} dto.ErrorResponse "Bad gateway"
// @Failure      503 {object} dto.ErrorResponse "Service unavailable"
// @Security     BearerAuth
// @Router       /api/calculate [post]
func (h *Handler) CalculatePacks(c *gin.Context) {
	builder := NewResponseBuilder(c)

	var req dto.CalculatePacksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		builder.Error(http.StatusBadRequest, i18n.ErrKeyInvalidRequestBody, err)
		return
	}

	if err := req.Validate(); err != nil {
		if _, ok := err.(*dto.ValidationError); ok {
			metrics.RecordPackCalculation(0, "validation_error")
			builder.Error(http.StatusBadRequest, i18n.ErrKeyValidationItemsOrdered, err)
		} else {
			builder.Error(http.StatusBadRequest, i18n.ErrKeyInvalidRequestBody, err)
		}
		return
	}

	start := time.Now()
	var result model.PackResult

	// Audit log (async)
	if loggingService, exists := c.Get("logging_service"); exists {
		if ls, ok := loggingService.(service.LoggingService); ok {
			middleware.AuditLog(ls, c, "calculate", "Pack calculation requested", map[string]interface{}{
				"items_ordered":    req.ItemsOrdered,
				"has_custom_sizes": len(req.PackSizes) > 0,
			})
		}
	}

	if len(req.PackSizes) > 0 {
		// Use custom pack sizes from request
		validPackSizes := make([]int, 0, len(req.PackSizes))
		for _, size := range req.PackSizes {
			if size > 0 {
				validPackSizes = append(validPackSizes, size)
			}
		}
		if len(validPackSizes) > 0 {
			result = h.calculator.CalculateWithPackSizes(req.ItemsOrdered, validPackSizes)
		} else {
			result = h.calculator.Calculate(req.ItemsOrdered)
		}
	} else {
		// Use cached pack sizes from database or defaults
		packSizes := h.getPackSizes(c.Request.Context())

		if len(packSizes) > 0 {
			result = h.calculator.CalculateWithPackSizes(req.ItemsOrdered, packSizes)
		} else {
			result = h.calculator.Calculate(req.ItemsOrdered)
		}
	}

	duration := time.Since(start)

	metrics.RecordPackCalculation(duration, "success")
	builder.SuccessOK(result)
}
