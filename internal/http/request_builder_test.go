package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/i18n"
	"github.com/guttosm/pack-service/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestBuilder_Bind(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		body          string
		expectedItems int
		expectError   bool
	}{
		{
			name:          "valid request",
			body:          `{"items_ordered": 251}`,
			expectedItems: 251,
			expectError:   false,
		},
		{
			name:        "invalid JSON",
			body:        `{"items_ordered": invalid}`,
			expectError: true,
		},
		{
			name:        "empty body",
			body:        ``,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			builder := NewRequestBuilder(c)
			var request dto.CalculatePacksRequest
			err := builder.Bind(&request)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedItems, request.ItemsOrdered)
			}
		})
	}
}

func TestUnmarshalFromBytes(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectError bool
	}{
		{
			name:        "valid JSON",
			data:        []byte(`{"items_ordered": 251}`),
			expectError: false,
		},
		{
			name:        "invalid JSON",
			data:        []byte(`{"items_ordered": invalid}`),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UnmarshalFromBytes[dto.CalculatePacksRequest](tt.data)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 251, result.ItemsOrdered)
			}
		})
	}
}

func TestUnmarshalFromReader(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		expectError bool
	}{
		{
			name:        "valid JSON",
			body:        `{"items_ordered": 251}`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			body:        `{"items_ordered": invalid}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewBufferString(tt.body)
			result, err := UnmarshalFromReader[dto.CalculatePacksRequest](reader)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 251, result.ItemsOrdered)
			}
		})
	}
}

func TestBuildRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		body        string
		expectError bool
	}{
		{
			name:        "valid request",
			body:        `{"items_ordered": 251}`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			body:        `{"items_ordered": invalid}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			result, err := BuildRequest[dto.CalculatePacksRequest](c)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 251, result.ItemsOrdered)
			}
		})
	}
}

func TestBuildRequestAndValidate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		body        string
		expectError bool
	}{
		{
			name:        "valid request",
			body:        `{"items_ordered": 251}`,
			expectError: false,
		},
		{
			name:        "invalid request - zero items",
			body:        `{"items_ordered": 0}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			result, err := BuildRequestAndValidate[dto.CalculatePacksRequest](c)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, 251, result.ItemsOrdered)
			}
		})
	}
}

func TestResponseBuilder_ErrorWithKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	middleware.RequestID()(c)
	builder := NewResponseBuilder(c)

	builder.Error(http.StatusBadRequest, i18n.ErrKeyInvalidRequest, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errorResp dto.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Equal(t, dto.ErrCodeInvalidRequest, errorResp.Error)
	assert.NotEmpty(t, errorResp.Message)
}

func TestResponseBuilder_ErrorWithCustomMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	middleware.RequestID()(c)
	builder := NewResponseBuilder(c)

	customMessage := "Custom error message"
	builder.ErrorWithMessage(http.StatusBadRequest, customMessage, nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errorResp dto.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	assert.NoError(t, err)
	assert.Equal(t, customMessage, errorResp.Message)
}

func TestMarshalJSON(t *testing.T) {
	data := dto.CalculatePacksRequest{ItemsOrdered: 251}
	result, err := MarshalJSON(data)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	var unmarshaled dto.CalculatePacksRequest
	err = json.Unmarshal(result, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, 251, unmarshaled.ItemsOrdered)
}

func TestMarshalToWriter(t *testing.T) {
	data := dto.CalculatePacksRequest{ItemsOrdered: 251}
	var buf bytes.Buffer

	err := MarshalToWriter(&buf, data)
	assert.NoError(t, err)

	var result dto.CalculatePacksRequest
	err = json.Unmarshal(buf.Bytes(), &result)
	assert.NoError(t, err)
	assert.Equal(t, 251, result.ItemsOrdered)
}
