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
	"github.com/guttosm/pack-service/internal/mocks"
	"github.com/guttosm/pack-service/internal/service"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter() *gin.Engine {
	calculator := service.NewPackCalculatorService()
	handler := NewHandler(calculator, nil) // nil means pack sizes from MongoDB are disabled
	healthHandler := NewHealthHandler()
	return NewRouter(handler, healthHandler, DefaultRouterConfig())
}

func setupRouterWithMock(t *testing.T) (*gin.Engine, *mocks.MockPackCalculator) {
	mockCalc := mocks.NewMockPackCalculator(t)
	handler := NewHandler(mockCalc, nil) // nil means pack sizes from MongoDB are disabled
	healthHandler := NewHealthHandler()
	return NewRouter(handler, healthHandler, DefaultRouterConfig()), mockCalc
}

func TestCalculatePacks(t *testing.T) {
	router := setupRouter()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "valid request",
			body:           `{"items_ordered": 251}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.RequestID)
				assert.NotZero(t, resp.Timestamp)
				
				// Unmarshal data field
				dataBytes, _ := json.Marshal(resp.Data)
				var packResult model.PackResult
				err = json.Unmarshal(dataBytes, &packResult)
				assert.NoError(t, err)
				assert.Equal(t, 251, packResult.OrderedItems)
				assert.Equal(t, 500, packResult.TotalItems)
				assert.Len(t, packResult.Packs, 1)
			},
		},
		{
			name:           "large order",
			body:           `{"items_ordered": 12001}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				
				// Unmarshal data field
				dataBytes, _ := json.Marshal(resp.Data)
				var packResult model.PackResult
				err = json.Unmarshal(dataBytes, &packResult)
				assert.NoError(t, err)
				assert.Equal(t, 12001, packResult.OrderedItems)
				assert.Equal(t, 12250, packResult.TotalItems)
			},
		},
		{
			name:           "invalid JSON",
			body:           `invalid`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "zero items",
			body:           `{"items_ordered": 0}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative items",
			body:           `{"items_ordered": -10}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing field",
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "request with custom pack sizes",
			body:           `{"items_ordered": 100, "pack_sizes": [50, 100, 200]}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp dto.SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				dataBytes, _ := json.Marshal(resp.Data)
				var packResult model.PackResult
				err = json.Unmarshal(dataBytes, &packResult)
				assert.NoError(t, err)
				assert.Equal(t, 100, packResult.OrderedItems)
			},
		},
		{
			name:           "request with invalid pack sizes (negative)",
			body:           `{"items_ordered": 100, "pack_sizes": [-10, 50]}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should filter out negative sizes and use default
				var resp dto.SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
			},
		},
		{
			name:           "request with zero pack sizes",
			body:           `{"items_ordered": 100, "pack_sizes": [0, 0]}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should filter out zero sizes and use default
				var resp dto.SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestCalculatePacks_WithMock(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		setupMock      func(*mocks.MockPackCalculator) model.PackResult
		expectedStatus int
		validateResult func(*testing.T, *httptest.ResponseRecorder, model.PackResult)
	}{
		{
			name: "calculate with mock returns expected result",
			body: `{"items_ordered": 100}`,
			setupMock: func(mockCalc *mocks.MockPackCalculator) model.PackResult {
				expectedResult := model.PackResult{
					OrderedItems: 100,
					TotalItems:   250,
					Packs:        []model.Pack{{Size: 250, Quantity: 1}},
				}
				mockCalc.EXPECT().Calculate(100).Return(expectedResult)
				return expectedResult
			},
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, w *httptest.ResponseRecorder, expectedResult model.PackResult) {
				var resp dto.SuccessResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				
				dataBytes, _ := json.Marshal(resp.Data)
				var packResult model.PackResult
				err = json.Unmarshal(dataBytes, &packResult)
				assert.NoError(t, err)
				assert.Equal(t, expectedResult, packResult)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mockCalc := setupRouterWithMock(t)
			expectedResult := tt.setupMock(mockCalc)

			req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateResult != nil {
				tt.validateResult(t, w, expectedResult)
			}
		})
	}
}

func TestCalculatePacks_WithCustomPackSizes(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		setupMock      func(*mocks.MockPackCalculator)
		expectedStatus int
	}{
		{
			name: "calculate with custom pack sizes",
			body: `{"items_ordered": 100, "pack_sizes": [100, 50]}`,
			setupMock: func(mockCalc *mocks.MockPackCalculator) {
				expectedResult := model.PackResult{
					OrderedItems: 100,
					TotalItems:   100,
					Packs:        []model.Pack{{Size: 100, Quantity: 1}},
				}
				mockCalc.EXPECT().CalculateWithPackSizes(100, []int{100, 50}).Return(expectedResult)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mockCalc := setupRouterWithMock(t)
			tt.setupMock(mockCalc)

			req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCalculatePacks_WithInvalidPackSizes(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		setupMock      func(*mocks.MockPackCalculator)
		expectedStatus int
	}{
		{
			name: "invalid pack sizes fallback to default",
			body: `{"items_ordered": 100, "pack_sizes": [0, -1]}`,
			setupMock: func(mockCalc *mocks.MockPackCalculator) {
				expectedResult := model.PackResult{
					OrderedItems: 100,
					TotalItems:   250,
					Packs:        []model.Pack{{Size: 250, Quantity: 1}},
				}
				mockCalc.EXPECT().Calculate(100).Return(expectedResult)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, mockCalc := setupRouterWithMock(t)
			tt.setupMock(mockCalc)

			req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHealthEndpoints(t *testing.T) {
	router := setupRouter()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "liveness probe",
			path:           "/healthz",
			expectedStatus: http.StatusOK,
			expectedBody:   `"status":"ok"`,
		},
		{
			name:           "readiness probe",
			path:           "/readyz",
			expectedStatus: http.StatusOK,
			expectedBody:   `"status":"ok"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
		})
	}
}

func BenchmarkHandler(b *testing.B) {
	router := setupRouter()
	body := []byte(`{"items_ordered": 12001}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
