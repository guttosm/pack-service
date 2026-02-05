package dto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestErrorResponse_WithRequestID(t *testing.T) {
	tests := []struct {
		name        string
		errCode     string
		message     string
		requestID   string
		validate    func(*testing.T, ErrorResponse)
	}{
		{
			name:      "error response with request ID",
			errCode:    ErrCodeInternal,
			message:   "test error",
			requestID: "test-id",
			validate: func(t *testing.T, err ErrorResponse) {
				assert.Equal(t, "test-id", err.RequestID)
				assert.Equal(t, ErrCodeInternal, err.Error)
				assert.Equal(t, "test error", err.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.errCode, tt.message)
			err = err.WithRequestID(tt.requestID)
			if tt.validate != nil {
				tt.validate(t, err)
			}
		})
	}
}

func TestErrCodeFromStatus(t *testing.T) {
	tests := []struct {
		status     int
		expectedCode string
	}{
		{400, ErrCodeInvalidRequest},
		{401, ErrCodeUnauthorized},
		{403, ErrCodeForbidden},
		{404, ErrCodeNotFound},
		{409, ErrCodeConflict},
		{429, ErrCodeRateLimit},
		{500, ErrCodeInternal},
		{502, ErrCodeInternal},
		{503, ErrCodeInternal},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.status)), func(t *testing.T) {
			code := ErrCodeFromStatus(tt.status)
			assert.Equal(t, tt.expectedCode, code)
		})
	}
}

func TestNewError(t *testing.T) {
	tests := []struct {
		name      string
		errCode   string
		message   string
		validate  func(*testing.T, ErrorResponse)
	}{
		{
			name:    "new error with code and message",
			errCode: ErrCodeInvalidRequest,
			message: "test message",
			validate: func(t *testing.T, err ErrorResponse) {
				assert.Equal(t, ErrCodeInvalidRequest, err.Error)
				assert.Equal(t, "test message", err.Message)
				assert.NotZero(t, err.Timestamp)
				assert.WithinDuration(t, time.Now(), err.Timestamp, time.Second)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.errCode, tt.message)
			if tt.validate != nil {
				tt.validate(t, err)
			}
		})
	}
}
