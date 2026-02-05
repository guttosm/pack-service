//go:build contract

package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/middleware"
	"github.com/guttosm/pack-service/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestAPI_ContractCompliance validates that API responses match the documented contract.
func TestAPI_ContractCompliance(t *testing.T) {
	calculator := service.NewPackCalculatorService()
	handler := NewHandler(calculator, nil) // nil means pack sizes from MongoDB are disabled
	healthHandler := NewHealthHandler()

	router := gin.New()
	router.Use(middleware.RequestID(), middleware.Recovery(), middleware.ErrorHandler())
	healthHandler.Register(router)
	api := router.Group("/api")
	api.POST("/calculate", handler.CalculatePacks)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		headers        map[string]string
		expectedStatus int
		validateResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "POST /api/calculate - Success 200",
			method:         http.MethodPost,
			path:           "/api/calculate",
			body:           `{"items_ordered": 251}`,
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)

				// Validate dto.SuccessResponse structure
				assert.NotEmpty(t, resp.RequestID, "Response must include request_id")
				assert.NotZero(t, resp.Timestamp, "Response must include timestamp")
				assert.NotNil(t, resp.Data, "Response must include data")

				// Validate PackResult structure
				packResult, ok := resp.Data.(map[string]interface{})
				require.True(t, ok, "Data must be PackResult")

				assert.Contains(t, packResult, "ordered_items")
				assert.Contains(t, packResult, "total_items")
				assert.Contains(t, packResult, "packs")

				orderedItems, ok := packResult["ordered_items"].(float64)
				require.True(t, ok)
				assert.Equal(t, float64(251), orderedItems)

				totalItems, ok := packResult["total_items"].(float64)
				require.True(t, ok)
				assert.GreaterOrEqual(t, totalItems, orderedItems)

				// Validate packs array
				packs, ok := packResult["packs"].([]interface{})
				require.True(t, ok)
				assert.NotEmpty(t, packs)

				// Validate each pack structure
				for _, packInterface := range packs {
					pack, ok := packInterface.(map[string]interface{})
					require.True(t, ok)
					assert.Contains(t, pack, "size")
					assert.Contains(t, pack, "quantity")
					assert.NotNil(t, pack["size"])
					assert.NotNil(t, pack["quantity"])
				}
			},
		},
		{
			name:           "POST /api/calculate - Error 400 Invalid JSON",
			method:         http.MethodPost,
			path:           "/api/calculate",
			body:           `invalid json`,
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)

				assert.Equal(t, dto.ErrCodeInvalidRequest, resp.Error)
				assert.NotEmpty(t, resp.Message)
				assert.NotEmpty(t, resp.RequestID)
				assert.NotZero(t, resp.Timestamp)
			},
		},
		{
			name:           "POST /api/calculate - Error 400 Invalid Input",
			method:         http.MethodPost,
			path:           "/api/calculate",
			body:           `{"items_ordered": 0}`,
			expectedStatus: http.StatusBadRequest,
			validateResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)

				assert.Equal(t, dto.ErrCodeInvalidRequest, resp.Error)
				assert.NotEmpty(t, resp.Message)
				assert.NotEmpty(t, resp.RequestID)
				assert.NotZero(t, resp.Timestamp)
			},
		},
		{
			name:           "GET /healthz - Success 200",
			method:         http.MethodGet,
			path:           "/healthz",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)

				assert.Contains(t, resp, "status")
				assert.Equal(t, "ok", resp["status"])
			},
		},
		{
			name:           "GET /readyz - Success 200",
			method:         http.MethodGet,
			path:           "/readyz",
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)

				assert.Contains(t, resp, "status")
				assert.Contains(t, resp, "checks")
				assert.Equal(t, "ok", resp["status"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewReader([]byte(tt.body)))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

			// Validate X-Request-ID header
			assert.NotEmpty(t, w.Header().Get("X-Request-ID"), "Response must include X-Request-ID header")

			if tt.validateResponse != nil {
				tt.validateResponse(t, w)
			}
		})
	}
}

// TestAPI_ResponseSchema validates response schemas match the contract.
func TestAPI_ResponseSchema(t *testing.T) {
	calculator := service.NewPackCalculatorService()
	handler := NewHandler(calculator, nil) // nil means pack sizes from MongoDB are disabled

	router := gin.New()
	router.Use(middleware.RequestID())
	api := router.Group("/api")
	api.POST("/calculate", handler.CalculatePacks)

	t.Run("SuccessResponse schema validation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader([]byte(`{"items_ordered": 100}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp dto.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		// Validate all required fields
		assert.NotEmpty(t, resp.RequestID)
		assert.NotZero(t, resp.Timestamp)
		assert.NotNil(t, resp.Data)

		// Validate data is PackResult
		dataBytes, _ := json.Marshal(resp.Data)
		var packResult model.PackResult
		err = json.Unmarshal(dataBytes, &packResult)
		require.NoError(t, err)

		assert.Greater(t, packResult.OrderedItems, 0)
		assert.GreaterOrEqual(t, packResult.TotalItems, packResult.OrderedItems)
		assert.NotNil(t, packResult.Packs)
	})

	t.Run("ErrorResponse schema validation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader([]byte(`{"items_ordered": -1}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)

		var resp dto.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		// Validate error response structure
		assert.NotEmpty(t, resp.Error)
		assert.NotEmpty(t, resp.Message)
		assert.NotEmpty(t, resp.RequestID)
		assert.NotZero(t, resp.Timestamp)
	})
}

// TestAPI_Headers validates required headers are present.
func TestAPI_Headers(t *testing.T) {
	calculator := service.NewPackCalculatorService()
	handler := NewHandler(calculator, nil) // nil means pack sizes from MongoDB are disabled
	healthHandler := NewHealthHandler()

	router := gin.New()
	router.Use(middleware.RequestID())
	healthHandler.Register(router)
	api := router.Group("/api")
	api.POST("/calculate", handler.CalculatePacks)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedHeaders map[string]string
	}{
		{
			name:   "X-Request-ID header present",
			method: http.MethodPost,
			path:   "/api/calculate",
			body:   `{"items_ordered": 100}`,
			expectedHeaders: map[string]string{
				"X-Request-ID": "",
			},
		},
		{
			name:   "Health endpoint headers",
			method: http.MethodGet,
			path:   "/healthz",
			expectedHeaders: map[string]string{
				"X-Request-ID": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewReader([]byte(tt.body)))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			for headerName, expectedValue := range tt.expectedHeaders {
				actualValue := w.Header().Get(headerName)
				if expectedValue == "" {
					assert.NotEmpty(t, actualValue, "Header %s must be present", headerName)
				} else {
					assert.Equal(t, expectedValue, actualValue, "Header %s mismatch", headerName)
				}
			}
		})
	}
}
