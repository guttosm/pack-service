package dto

import (
	"net/http"
	"time"
)

const (
	// ErrCodeInvalidRequest indicates an invalid request.
	ErrCodeInvalidRequest = "invalid_request"
	// ErrCodeInternal indicates an internal server error.
	ErrCodeInternal = "internal_error"
	// ErrCodeUnauthorized indicates missing or invalid authentication.
	ErrCodeUnauthorized = "unauthorized"
	// ErrCodeForbidden indicates insufficient permissions.
	ErrCodeForbidden = "forbidden"
	// ErrCodeNotFound indicates a resource was not found.
	ErrCodeNotFound = "not_found"
	// ErrCodeRateLimit indicates rate limit exceeded.
	ErrCodeRateLimit = "rate_limit_exceeded"
	// ErrCodeConflict indicates a conflict with current state.
	ErrCodeConflict = "conflict"
	// ErrCodeTimeout indicates a request timeout.
	ErrCodeTimeout = "timeout"
)

// SuccessResponse wraps successful API responses with metadata.
// @Description Successful API response wrapper
type SuccessResponse struct {
	// Data contains the actual response data (PackResult for calculate endpoint)
	// Example: {"ordered_items": 251, "total_items": 500, "packs": [{"size": 500, "quantity": 1}]}
	Data      interface{} `json:"data" swaggertype:"object"`
	// RequestID is the unique request identifier
	RequestID string       `json:"request_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	// Timestamp is when the response was generated
	Timestamp time.Time    `json:"timestamp" example:"2025-01-28T10:00:00Z"`
} // @name SuccessResponse

// ErrorResponse represents a standardized error response for the API.
// @Description Standardized error response
type ErrorResponse struct {
	Error     string            `json:"error" example:"invalid_request"`
	Message   string            `json:"message,omitempty" example:"items_ordered: must be a positive integer"`
	// Details contains additional error details (optional)
	// Example: {"field": "error message"}
	Details   map[string]string `json:"details,omitempty"`
	RequestID string            `json:"request_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Timestamp time.Time         `json:"timestamp" example:"2025-01-28T10:00:00Z"`
	TraceID   string            `json:"trace_id,omitempty" example:"trace-123"`
} // @name ErrorResponse

// NewError creates a new ErrorResponse with the given code and message.
func NewError(code, message string) ErrorResponse {
	return ErrorResponse{
		Error:     code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// WithRequestID adds a request ID to the error response.
func (e ErrorResponse) WithRequestID(requestID string) ErrorResponse {
	e.RequestID = requestID
	return e
}

// ErrCodeFromStatus returns the appropriate error code for an HTTP status.
func ErrCodeFromStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return ErrCodeInvalidRequest
	case http.StatusUnauthorized:
		return ErrCodeUnauthorized
	case http.StatusForbidden:
		return ErrCodeForbidden
	case http.StatusNotFound:
		return ErrCodeNotFound
	case http.StatusConflict:
		return ErrCodeConflict
	case http.StatusTooManyRequests:
		return ErrCodeRateLimit
	case http.StatusGatewayTimeout, http.StatusRequestTimeout:
		return ErrCodeTimeout
	default:
		return ErrCodeInternal
	}
}
