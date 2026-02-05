package http

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/i18n"
	"github.com/guttosm/pack-service/internal/middleware"
)

// Response DTO pools for reducing allocations.
var (
	successResponsePool = sync.Pool{
		New: func() interface{} {
			return &dto.SuccessResponse{}
		},
	}

	errorResponsePool = sync.Pool{
		New: func() interface{} {
			return &dto.ErrorResponse{}
		},
	}
)

// getSuccessResponse retrieves a SuccessResponse from the pool.
func getSuccessResponse() *dto.SuccessResponse {
	if resp, ok := successResponsePool.Get().(*dto.SuccessResponse); ok {
		return resp
	}
	return &dto.SuccessResponse{}
}

// putSuccessResponse returns a SuccessResponse to the pool.
func putSuccessResponse(resp *dto.SuccessResponse) {
	// Clear the response before returning to pool
	resp.Data = nil
	resp.RequestID = ""
	resp.Timestamp = time.Time{}
	successResponsePool.Put(resp)
}

// getErrorResponse retrieves an ErrorResponse from the pool.
func getErrorResponse() *dto.ErrorResponse {
	if resp, ok := errorResponsePool.Get().(*dto.ErrorResponse); ok {
		return resp
	}
	return &dto.ErrorResponse{}
}

// putErrorResponse returns an ErrorResponse to the pool.
func putErrorResponse(resp *dto.ErrorResponse) {
	// Clear the response before returning to pool
	resp.Error = ""
	resp.Message = ""
	resp.RequestID = ""
	resp.Timestamp = time.Time{}
	resp.Details = nil
	resp.TraceID = ""
	errorResponsePool.Put(resp)
}

// RequestBuilder provides generic request building and unmarshaling capabilities.
type RequestBuilder struct {
	c *gin.Context
}

// NewRequestBuilder creates a new request builder for the given context.
func NewRequestBuilder(c *gin.Context) *RequestBuilder {
	return &RequestBuilder{c: c}
}

// Bind unmarshals the request body into the provided type.
func (b *RequestBuilder) Bind(v interface{}) error {
	if err := b.c.ShouldBindJSON(v); err != nil {
		return err
	}
	return nil
}

// UnmarshalFromReader unmarshals JSON from an io.Reader into the provided type.
func UnmarshalFromReader[T any](reader io.Reader) (*T, error) {
	var v T
	if err := json.NewDecoder(reader).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

// UnmarshalFromBytes unmarshals JSON bytes into the provided type.
func UnmarshalFromBytes[T any](data []byte) (*T, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// ResponseBuilder provides generic response building and marshaling capabilities.
// Uses sync.Pool for DTO reuse to reduce allocations.
type ResponseBuilder struct {
	c *gin.Context
}

// NewResponseBuilder creates a new response builder for the given context.
func NewResponseBuilder(c *gin.Context) *ResponseBuilder {
	return &ResponseBuilder{c: c}
}

// Success sends a successful response with the given data.
// Uses pooled SuccessResponse to reduce allocations.
func (b *ResponseBuilder) Success(statusCode int, data interface{}) {
	requestID := middleware.GetRequestID(b.c)

	// Get pooled response
	resp := getSuccessResponse()

	// Set values
	resp.Data = data
	resp.RequestID = requestID
	resp.Timestamp = time.Now()

	// Send response (this copies the data)
	b.c.JSON(statusCode, resp)

	// Return to pool after response is sent
	// Note: Gin's JSON serialization happens synchronously, so this is safe
	putSuccessResponse(resp)
}

// SuccessOK sends a 200 OK response with the given data.
func (b *ResponseBuilder) SuccessOK(data interface{}) {
	b.Success(http.StatusOK, data)
}

// SuccessCreated sends a 201 Created response with the given data.
func (b *ResponseBuilder) SuccessCreated(data interface{}) {
	b.Success(http.StatusCreated, data)
}

// SuccessAccepted sends a 202 Accepted response with the given data.
func (b *ResponseBuilder) SuccessAccepted(data interface{}) {
	b.Success(http.StatusAccepted, data)
}

// Error sends an error response with the given status code and message key.
// Uses pooled ErrorResponse to reduce allocations.
func (b *ResponseBuilder) Error(statusCode int, messageKey string, err error) {
	requestID := middleware.GetRequestID(b.c)
	locale := i18n.GetLocale(b.c)

	translatedMessage := i18n.GetTranslator().Translate(messageKey, locale)

	// Get pooled response
	resp := getErrorResponse()

	// Set values
	resp.Error = dto.ErrCodeFromStatus(statusCode)
	resp.Message = translatedMessage
	resp.RequestID = requestID
	resp.Timestamp = time.Now()

	// Add error to context for error handler middleware to log
	if err != nil {
		_ = b.c.Error(err)
	}

	b.c.AbortWithStatusJSON(statusCode, resp)

	// Return to pool after response is sent
	putErrorResponse(resp)
}

// ErrorWithMessage sends an error response with a custom message.
// Uses pooled ErrorResponse to reduce allocations.
func (b *ResponseBuilder) ErrorWithMessage(statusCode int, message string, err error) {
	requestID := middleware.GetRequestID(b.c)

	// Get pooled response
	resp := getErrorResponse()

	// Set values
	resp.Error = dto.ErrCodeFromStatus(statusCode)
	resp.Message = message
	resp.RequestID = requestID
	resp.Timestamp = time.Now()

	if err != nil {
		_ = b.c.Error(err)
	}

	b.c.AbortWithStatusJSON(statusCode, resp)

	// Return to pool after response is sent
	putErrorResponse(resp)
}

// MarshalJSON marshals the provided value to JSON bytes.
func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// MarshalToWriter marshals the provided value to JSON and writes it to the writer.
func MarshalToWriter(w io.Writer, v interface{}) error {
	return json.NewEncoder(w).Encode(v)
}

// BuildRequest is a generic helper to build and validate a request from gin context.
func BuildRequest[T any](c *gin.Context) (*T, error) {
	builder := NewRequestBuilder(c)
	var req T
	if err := builder.Bind(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

// Validator interface for types that can validate themselves.
type Validator interface {
	Validate() error
}

// BuildRequestAndValidate builds a request and validates it if it implements Validator.
func BuildRequestAndValidate[T any](c *gin.Context) (*T, error) {
	req, err := BuildRequest[T](c)
	if err != nil {
		return nil, err
	}
	if validator, ok := any(req).(Validator); ok {
		if err := validator.Validate(); err != nil {
			return nil, err
		}
	}
	return req, nil
}
