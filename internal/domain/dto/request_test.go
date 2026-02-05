package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculatePacksRequest_Validate(t *testing.T) {
	tests := []struct {
		name          string
		request       CalculatePacksRequest
		expectedError bool
	}{
		{
			name:          "valid request",
			request:       CalculatePacksRequest{ItemsOrdered: 100},
			expectedError: false,
		},
		{
			name:          "zero items",
			request:       CalculatePacksRequest{ItemsOrdered: 0},
			expectedError: true,
		},
		{
			name:          "negative items",
			request:       CalculatePacksRequest{ItemsOrdered: -10},
			expectedError: true,
		},
		{
			name:          "large valid number",
			request:       CalculatePacksRequest{ItemsOrdered: 100000},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, ErrInvalidItemsOrdered, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name          string
		validationErr *ValidationError
		expected      string
	}{
		{
			name: "validation error message format",
			validationErr: &ValidationError{
				Field:   "items_ordered",
				Message: "must be positive",
			},
			expected: "items_ordered: must be positive",
		},
		{
			name: "validation error with different field",
			validationErr: &ValidationError{
				Field:   "email",
				Message: "invalid format",
			},
			expected: "email: invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.validationErr.Error())
		})
	}
}
