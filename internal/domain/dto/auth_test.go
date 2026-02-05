package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		request   LoginRequest
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid request",
			request: LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantError: false,
		},
		{
			name: "empty email",
			request: LoginRequest{
				Email:    "",
				Password: "password123",
			},
			wantError: true,
			errorMsg:  "email is required",
		},
		{
			name: "password too short",
			request: LoginRequest{
				Email:    "test@example.com",
				Password: "12345",
			},
			wantError: true,
			errorMsg:  "password must be at least 6 characters",
		},
		{
			name: "empty password",
			request: LoginRequest{
				Email:    "test@example.com",
				Password: "",
			},
			wantError: true,
			errorMsg:  "password must be at least 6 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantError {
				assert.Error(t, err)
				if validationErr, ok := err.(*ValidationError); ok {
					assert.Equal(t, tt.errorMsg, validationErr.Message)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegisterRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		request   RegisterRequest
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid request",
			request: RegisterRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "password123",
				Name:     "Test User",
			},
			wantError: false,
		},
		{
			name: "valid request without name",
			request: RegisterRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "password123",
			},
			wantError: false,
		},
		{
			name: "empty email",
			request: RegisterRequest{
				Email:    "",
				Username: "testuser",
				Password: "password123",
			},
			wantError: true,
			errorMsg:  "email is required",
		},
		{
			name: "password too short",
			request: RegisterRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "12345",
			},
			wantError: true,
			errorMsg:  "password must be at least 6 characters",
		},
		{
			name: "username missing",
			request: RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			wantError: true,
			errorMsg:  "username is required",
		},
		{
			name: "username too short",
			request: RegisterRequest{
				Email:    "test@example.com",
				Username: "ab",
				Password: "password123",
			},
			wantError: true,
			errorMsg:  "username must be at least 3 characters",
		},
		{
			name: "username too long",
			request: RegisterRequest{
				Email:    "test@example.com",
				Username: "thisusernameistoolongandexceedsthelimit",
				Password: "password123",
			},
			wantError: true,
			errorMsg:  "username must be at most 30 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantError {
				assert.Error(t, err)
				if validationErr, ok := err.(*ValidationError); ok {
					assert.Equal(t, tt.errorMsg, validationErr.Message)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
