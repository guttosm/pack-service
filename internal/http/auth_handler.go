package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/i18n"
	"github.com/guttosm/pack-service/internal/middleware"
	"github.com/guttosm/pack-service/internal/service"
)

// AuthHandler provides HTTP handlers for authentication routes.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler creates a new authentication handler.
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login handles POST /api/auth/login requests.
//
// @Summary      Login user
// @Description  Authenticates a user and returns a JWT token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "Login credentials"
// @Success      200 {object} dto.LoginResponse "Successful login"
// @Failure      400 {object} dto.ErrorResponse "Bad request - invalid input"
// @Failure      401 {object} dto.ErrorResponse "Unauthorized - invalid credentials"
// @Failure      500 {object} dto.ErrorResponse "Internal server error"
// @Router       /api/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	builder := NewResponseBuilder(c)
	locale := i18n.GetLocale(c)

	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		builder.Error(http.StatusBadRequest, i18n.ErrKeyInvalidRequestBody, err)
		return
	}

	if err := req.Validate(); err != nil {
		if validationErr, ok := err.(*dto.ValidationError); ok {
			message := i18n.GetTranslator().Translate(i18n.ErrKeyValidationItemsOrdered, locale)
			// Override with specific validation message
			switch validationErr.Field {
			case "email":
				message = "email: email is required"
			case "password":
				message = "password: password must be at least 6 characters"
			}
			builder.Error(http.StatusBadRequest, dto.ErrCodeInvalidRequest, errors.New(message))
		} else {
			builder.Error(http.StatusBadRequest, i18n.ErrKeyInvalidRequestBody, err)
		}
		return
	}

	tokenPair, user, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			if loggingService, exists := c.Get("logging_service"); exists {
				if ls, ok := loggingService.(service.LoggingService); ok {
					middleware.AuditLogError(ls, c, "login_failed", "Failed login attempt", err, map[string]interface{}{
						"email": req.Email,
					})
				}
			}
			message := i18n.GetTranslator().Translate(i18n.ErrKeyInvalidCredentials, locale)
			builder.Error(http.StatusUnauthorized, dto.ErrCodeUnauthorized, errors.New(message))
		} else {
			// Log the actual error for debugging
			if loggingService, exists := c.Get("logging_service"); exists {
				if ls, ok := loggingService.(service.LoggingService); ok {
					middleware.AuditLogError(ls, c, "login_error", "Login internal error", err, map[string]interface{}{
						"email": req.Email,
						"error": err.Error(),
					})
				}
			}
			builder.Error(http.StatusInternalServerError, i18n.ErrKeyInternalError, err)
		}
		return
	}

	c.Set("user_id", user.ID)
	c.Set("user_email", user.Email)

	if loggingService, exists := c.Get("logging_service"); exists {
		if ls, ok := loggingService.(service.LoggingService); ok {
			middleware.AuditLog(ls, c, "login", "User logged in successfully", map[string]interface{}{
				"email": user.Email,
			})
		}
	}

	response := dto.LoginResponse{
		Token: tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User: dto.UserResponse{
			Email: user.Email,
			Name:  user.Name,
		},
	}
	builder.SuccessOK(response)
}

// Register handles POST /api/auth/register requests.
//
// @Summary      Register new user
// @Description  Creates a new user account and returns a JWT token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterRequest true "Registration information"
// @Success      201 {object} dto.LoginResponse "Successful registration"
// @Failure      400 {object} dto.ErrorResponse "Bad request - invalid input"
// @Failure      409 {object} dto.ErrorResponse "Conflict - user already exists"
// @Failure      500 {object} dto.ErrorResponse "Internal server error"
// @Router       /api/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	builder := NewResponseBuilder(c)
	locale := i18n.GetLocale(c)

	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		builder.Error(http.StatusBadRequest, i18n.ErrKeyInvalidRequestBody, err)
		return
	}

		if err := req.Validate(); err != nil {
		if validationErr, ok := err.(*dto.ValidationError); ok {
			message := "validation error: " + validationErr.Message
			builder.Error(http.StatusBadRequest, dto.ErrCodeInvalidRequest, errors.New(message))
		} else {
			builder.Error(http.StatusBadRequest, i18n.ErrKeyInvalidRequestBody, err)
		}
		return
	}

	tokenPair, user, err := h.authService.Register(c.Request.Context(), req.Email, req.Username, req.Password, req.Name)
	if err != nil {
		if err == service.ErrUserExists {
			if loggingService, exists := c.Get("logging_service"); exists {
				if ls, ok := loggingService.(service.LoggingService); ok {
					middleware.AuditLogError(ls, c, "register_failed", "Failed registration attempt - user already exists", err, map[string]interface{}{
						"email": req.Email,
					})
				}
			}
			message := i18n.GetTranslator().Translate(i18n.ErrKeyConflict, locale)
			builder.Error(http.StatusConflict, dto.ErrCodeConflict, errors.New(message))
		} else {
			builder.Error(http.StatusInternalServerError, i18n.ErrKeyInternalError, err)
		}
		return
	}

	c.Set("user_id", user.ID)
	c.Set("user_email", user.Email)

	if loggingService, exists := c.Get("logging_service"); exists {
		if ls, ok := loggingService.(service.LoggingService); ok {
			middleware.AuditLog(ls, c, "register", "New user registered successfully", map[string]interface{}{
				"email": user.Email,
				"name":  user.Name,
			})
		}
	}

	response := dto.LoginResponse{
		Token: tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User: dto.UserResponse{
			Email: user.Email,
			Name:  user.Name,
		},
	}
	builder.SuccessCreated(response)
}

// RefreshToken handles POST /api/auth/refresh requests.
//
// @Summary      Refresh access token
// @Description  Generates a new access token using a refresh token. Refresh token is extracted from X-Refresh-Token header.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        X-Refresh-Token header string true "Refresh token"
// @Success      200 {object} dto.LoginResponse "Successful token refresh"
// @Failure      400 {object} dto.ErrorResponse "Bad request - missing refresh token"
// @Failure      401 {object} dto.ErrorResponse "Unauthorized - invalid refresh token"
// @Failure      500 {object} dto.ErrorResponse "Internal server error"
// @Router       /api/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	builder := NewResponseBuilder(c)
	locale := i18n.GetLocale(c)

	// Extract refresh token from X-Refresh-Token header
	refreshToken := c.GetHeader("X-Refresh-Token")
	if refreshToken == "" {
		builder.Error(http.StatusBadRequest, dto.ErrCodeInvalidRequest, errors.New("X-Refresh-Token header is required"))
		return
	}

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		if err == service.ErrInvalidToken {
			message := i18n.GetTranslator().Translate(i18n.ErrKeyInvalidToken, locale)
			builder.Error(http.StatusUnauthorized, dto.ErrCodeUnauthorized, errors.New(message))
		} else {
			builder.Error(http.StatusInternalServerError, i18n.ErrKeyInternalError, err)
		}
		return
	}

	response := dto.LoginResponse{
		Token: tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}
	builder.SuccessOK(response)
}

// Logout handles POST /api/auth/logout requests.
//
// @Summary      Logout user
// @Description  Invalidates access and refresh tokens. Access token is extracted from Authorization header, refresh token from X-Refresh-Token header.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        Authorization header string true "Bearer token" default(Bearer )
// @Param        X-Refresh-Token header string true "Refresh token"
// @Success      200 {object} dto.SuccessResponse "Successful logout"
// @Failure      400 {object} dto.ErrorResponse "Bad request - missing refresh token"
// @Failure      401 {object} dto.ErrorResponse "Unauthorized"
// @Failure      500 {object} dto.ErrorResponse "Internal server error"
// @Router       /api/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	builder := NewResponseBuilder(c)

	// Extract access token from Authorization header (already validated by JWTAuth middleware)
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		builder.Error(http.StatusUnauthorized, dto.ErrCodeUnauthorized, errors.New("authorization header required"))
		return
	}

	// Extract token from "Bearer <token>"
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		builder.Error(http.StatusUnauthorized, dto.ErrCodeUnauthorized, errors.New("invalid authorization header format"))
		return
	}

	accessToken := strings.TrimPrefix(authHeader, bearerPrefix)
	if accessToken == "" {
		builder.Error(http.StatusUnauthorized, dto.ErrCodeUnauthorized, errors.New("access token required"))
		return
	}

	// Extract refresh token from X-Refresh-Token header
	refreshToken := c.GetHeader("X-Refresh-Token")
	if refreshToken == "" {
		builder.Error(http.StatusBadRequest, dto.ErrCodeInvalidRequest, errors.New("X-Refresh-Token header is required"))
		return
	}

	err := h.authService.Logout(c.Request.Context(), accessToken, refreshToken)
	if err != nil {
		builder.Error(http.StatusInternalServerError, i18n.ErrKeyInternalError, err)
		return
	}

	if loggingService, exists := c.Get("logging_service"); exists {
		if ls, ok := loggingService.(service.LoggingService); ok {
			middleware.AuditLog(ls, c, "logout", "User logged out successfully", nil)
		}
	}

	builder.SuccessOK(map[string]string{"message": "Logged out successfully"})
}
