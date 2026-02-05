package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/middleware"
	"github.com/guttosm/pack-service/internal/service"
)

// PackSizesHandler provides HTTP handlers for pack sizes routes.
type PackSizesHandler struct {
	packSizesService service.PackSizesService
	calculator       service.PackCalculator
}

// NewPackSizesHandler creates a new PackSizesHandler instance.
func NewPackSizesHandler(packSizesService service.PackSizesService, calculator service.PackCalculator) *PackSizesHandler {
	return &PackSizesHandler{
		packSizesService: packSizesService,
		calculator:       calculator,
	}
}

// GetActivePackSizes handles GET /api/pack-sizes requests.
//
// @Summary      Get active pack sizes
// @Description  Returns the currently active pack size configuration
// @Tags         Pack Sizes
// @Accept       json
// @Produce      json
// @Param        Authorization header string false "Bearer token (required if auth enabled)"
// @Success      200 {object} dto.SuccessResponse "Active pack sizes"
// @Failure      401 {object} dto.ErrorResponse "Unauthorized - missing or invalid JWT token"
// @Failure      404 {object} dto.ErrorResponse "No active pack sizes found"
// @Failure      500 {object} dto.ErrorResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/pack-sizes [get]
func (h *PackSizesHandler) GetActivePackSizes(c *gin.Context) {
	builder := NewResponseBuilder(c)

	config, err := h.packSizesService.GetActive(c.Request.Context())
	if err != nil {
		builder.Error(http.StatusInternalServerError, dto.ErrCodeInternal, err)
		return
	}

	if config == nil {
		builder.Error(http.StatusNotFound, dto.ErrCodeNotFound, nil)
		return
	}

	builder.SuccessOK(map[string]interface{}{
		"sizes":     config.Sizes,
		"version":   config.Version,
		"created_at": config.CreatedAt,
		"updated_at": config.UpdatedAt,
	})
}

// UpdatePackSizes handles PUT /api/pack-sizes requests.
//
// @Summary      Update pack sizes
// @Description  Updates the active pack size configuration
// @Tags         Pack Sizes
// @Accept       json
// @Produce      json
// @Param        Authorization header string false "Bearer token (required if auth enabled)"
// @Param        request body dto.UpdatePackSizesRequest true "Pack sizes configuration"
// @Success      200 {object} dto.SuccessResponse "Updated pack sizes"
// @Failure      400 {object} dto.ErrorResponse "Bad request"
// @Failure      401 {object} dto.ErrorResponse "Unauthorized - missing or invalid JWT token"
// @Failure      500 {object} dto.ErrorResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/pack-sizes [put]
func (h *PackSizesHandler) UpdatePackSizes(c *gin.Context) {
	builder := NewResponseBuilder(c)

	var req dto.UpdatePackSizesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		builder.Error(http.StatusBadRequest, dto.ErrCodeInvalidRequest, err)
		return
	}

	if len(req.Sizes) == 0 {
		builder.Error(http.StatusBadRequest, dto.ErrCodeInvalidRequest, nil)
		return
	}

	config, err := h.packSizesService.Create(c.Request.Context(), req.Sizes, req.CreatedBy)
	if err != nil {
		builder.Error(http.StatusInternalServerError, dto.ErrCodeInternal, err)
		return
	}

	if h.calculator != nil {
		h.calculator.InvalidateCache()
	}

	if loggingService, exists := c.Get("logging_service"); exists {
		if ls, ok := loggingService.(service.LoggingService); ok {
			middleware.AuditLog(ls, c, "update_pack_sizes", "Pack sizes configuration updated", map[string]interface{}{
				"pack_sizes": req.Sizes,
				"version":    config.Version,
			})
		}
	}

	builder.SuccessOK(map[string]interface{}{
		"sizes":      config.Sizes,
		"version":    config.Version,
		"created_at": config.CreatedAt,
		"updated_at": config.UpdatedAt,
	})
}

// ListPackSizes handles GET /api/pack-sizes/history requests.
//
// @Summary      List pack sizes history
// @Description  Returns all pack size configurations (history)
// @Tags         Pack Sizes
// @Accept       json
// @Produce      json
// @Param        Authorization header string false "Bearer token (required if auth enabled)"
// @Param        limit query int false "Limit number of results"
// @Success      200 {object} dto.SuccessResponse "Pack sizes history"
// @Failure      401 {object} dto.ErrorResponse "Unauthorized - missing or invalid JWT token"
// @Failure      500 {object} dto.ErrorResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/pack-sizes/history [get]
func (h *PackSizesHandler) ListPackSizes(c *gin.Context) {
	builder := NewResponseBuilder(c)

	limit := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := parseInt(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	configs, err := h.packSizesService.List(c.Request.Context(), limit)
	if err != nil {
		builder.Error(http.StatusInternalServerError, dto.ErrCodeInternal, err)
		return
	}

	builder.SuccessOK(configs)
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

