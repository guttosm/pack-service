//go:build integration

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/circuitbreaker"
	"github.com/guttosm/pack-service/internal/domain/dto"
	"github.com/guttosm/pack-service/internal/domain/model"
	"github.com/guttosm/pack-service/internal/repository"
	"github.com/guttosm/pack-service/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
func init() {
	gin.SetMode(gin.TestMode)
}

func setupIntegrationRouter() *gin.Engine {
	calculator := service.NewPackCalculatorService(
		service.WithCache(100, 5*time.Minute),
	)
	handler := NewHandler(calculator, nil) // nil means pack sizes from MongoDB are disabled
	healthHandler := NewHealthHandler()

	cfg := RouterConfig{
		RateLimit:  10,
		RateWindow: time.Second,
		EnableAuth: false,
	}

	return NewRouter(handler, healthHandler, cfg)
}

func TestIntegration_CalculatePacks_AllScenarios(t *testing.T) {
	router := setupIntegrationRouter()

	testCases := []struct {
		name           string
		itemsOrdered   int
		expectedTotal  int
		expectedPacks  []model.Pack
	}{
		{
			name:          "single item",
			itemsOrdered:  1,
			expectedTotal: 250,
			expectedPacks: []model.Pack{{Size: 250, Quantity: 1}},
		},
		{
			name:          "exact pack size",
			itemsOrdered:  250,
			expectedTotal: 250,
			expectedPacks: []model.Pack{{Size: 250, Quantity: 1}},
		},
		{
			name:          "just over pack size",
			itemsOrdered:  251,
			expectedTotal: 500,
			expectedPacks: []model.Pack{{Size: 500, Quantity: 1}},
		},
		{
			name:          "multiple packs",
			itemsOrdered:  501,
			expectedTotal: 750,
			expectedPacks: []model.Pack{{Size: 500, Quantity: 1}, {Size: 250, Quantity: 1}},
		},
		{
			name:          "large order",
			itemsOrdered:  12001,
			expectedTotal: 12250,
			expectedPacks: []model.Pack{{Size: 5000, Quantity: 2}, {Size: 2000, Quantity: 1}, {Size: 250, Quantity: 1}},
		},
		{
			name:          "very large order",
			itemsOrdered:  100000,
			expectedTotal: 100000,
			expectedPacks: []model.Pack{{Size: 5000, Quantity: 20}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := bytes.NewBufferString(`{"items_ordered": ` + strconv.Itoa(tc.itemsOrdered) + `}`)

			req := httptest.NewRequest(http.MethodPost, "/api/calculate", body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)

			var response dto.SuccessResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			dataBytes, _ := json.Marshal(response.Data)
			var resp model.PackResult
			err = json.Unmarshal(dataBytes, &resp)
			require.NoError(t, err)

			assert.Equal(t, tc.itemsOrdered, resp.OrderedItems)
			assert.Equal(t, tc.expectedTotal, resp.TotalItems)
			assert.Equal(t, tc.expectedPacks, resp.Packs)

			// Verify total items equals sum of packs
			var sum int
			for _, p := range resp.Packs {
				sum += p.Size * p.Quantity
			}
			assert.Equal(t, resp.TotalItems, sum)
		})
	}
}

func TestIntegration_RateLimiting(t *testing.T) {
	calculator := service.NewPackCalculatorService()
	handler := NewHandler(calculator, nil) // nil means pack sizes from MongoDB are disabled
	healthHandler := NewHealthHandler()

	cfg := RouterConfig{
		RateLimit:  5,
		RateWindow: time.Second,
	}

	router := NewRouter(handler, healthHandler, cfg)

	body := []byte(`{"items_ordered": 100}`)

	// Make requests up to rate limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "Request %d", i+1)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestIntegration_APIKeyAuth(t *testing.T) {
	calculator := service.NewPackCalculatorService()
	handler := NewHandler(calculator, nil) // nil means pack sizes from MongoDB are disabled
	healthHandler := NewHealthHandler()

	cfg := RouterConfig{
		RateLimit:  100,
		RateWindow: time.Minute,
		EnableAuth: true,
		APIKeys:    map[string]bool{"valid-key": true},
	}

	router := NewRouter(handler, healthHandler, cfg)

	body := []byte(`{"items_ordered": 100}`)

	t.Run("missing API key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid API key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "invalid-key")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("valid API key in header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "valid-key")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("valid API key in query param", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/calculate?api_key=valid-key", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("health endpoints bypass auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestIntegration_CacheEffectiveness(t *testing.T) {
	router := setupIntegrationRouter()

	body := []byte(`{"items_ordered": 12001}`)

	// First request - cache miss
	start := time.Now()
	req1 := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	firstDuration := time.Since(start)

	require.Equal(t, http.StatusOK, w1.Code)

	start = time.Now()
	req2 := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	secondDuration := time.Since(start)

	require.Equal(t, http.StatusOK, w2.Code)

	var resp1 dto.SuccessResponse
	var resp2 dto.SuccessResponse
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)

	dataBytes1, _ := json.Marshal(resp1.Data)
	dataBytes2, _ := json.Marshal(resp2.Data)
	assert.Equal(t, string(dataBytes1), string(dataBytes2))

	t.Logf("First request (cache miss): %v", firstDuration)
	t.Logf("Second request (cache hit): %v", secondDuration)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	if negative {
		result = "-" + result
	}
	return result
}

func setupHandlerWithMongoDBIntegrationRouter(dbName string) (*gin.Engine, *repository.MongoDB) {
	gin.SetMode(gin.TestMode)

	uri := getSharedContainerURI()
	db, err := repository.NewMongoDB(uri, dbName)
	if err != nil {
		panic(err)
	}

	calculator := service.NewPackCalculatorService()

	logsRepo := repository.NewLogsRepository(db)
	logsCB := circuitbreaker.New(circuitbreaker.DefaultConfig())
	logsRepoWithCB := repository.NewLogsRepositoryWithCircuitBreaker(logsRepo, logsCB)
	loggingService := service.NewLoggingService(logsRepoWithCB)

	packSizesRepo := repository.NewPackSizesRepository(db)
	packSizesCB := circuitbreaker.New(circuitbreaker.DefaultConfig())
	packSizesRepoWithCB := repository.NewPackSizesRepositoryWithCircuitBreaker(packSizesRepo, packSizesCB)
	packSizesService := service.NewPackSizesService(packSizesRepoWithCB)

	handler := NewHandler(calculator, packSizesService)
	healthHandler := NewHealthHandler()

	cfg := RouterConfig{
		RateLimit:       100,
		RateWindow:     time.Minute,
		EnableAuth:     false,
		LoggingService: loggingService,
	}

	return NewRouter(handler, healthHandler, cfg), db
}

func TestHandler_CalculatePacks_WithMongoDB_Integration(t *testing.T) {
	ctx := context.Background()

	// Use shared container with unique database name
	dbName := sanitizeDBNameForHTTP(t.Name())
	router, db := setupHandlerWithMongoDBIntegrationRouter(dbName)
	defer func() {
		_ = db.Close(ctx)
	}()

	t.Run("calculate with pack sizes from MongoDB", func(t *testing.T) {
		repo := repository.NewPackSizesRepository(db)
		_, createErr := repo.Create(ctx, []int{100, 200, 500}, "test")
		require.NoError(t, createErr)

		body := []byte(`{"items_ordered": 150}`)
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		dataBytes, _ := json.Marshal(response.Data)
		var packResult model.PackResult
		err = json.Unmarshal(dataBytes, &packResult)
		require.NoError(t, err)
		assert.Equal(t, 150, packResult.OrderedItems)
		assert.GreaterOrEqual(t, packResult.TotalItems, 150)
	})

	t.Run("calculate falls back to default when no MongoDB config", func(t *testing.T) {
		active, _ := repository.NewPackSizesRepository(db).GetActive(ctx)
		if active != nil {
			_ = db.Database.Collection("pack_sizes").Drop(ctx)
		}

		body := []byte(`{"items_ordered": 251}`)
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		dataBytes, _ := json.Marshal(response.Data)
		var packResult model.PackResult
		err = json.Unmarshal(dataBytes, &packResult)
		require.NoError(t, err)
		assert.Equal(t, 251, packResult.OrderedItems)
		assert.GreaterOrEqual(t, packResult.TotalItems, 251)
	})

	t.Run("calculate with custom pack sizes overrides MongoDB", func(t *testing.T) {
		repo := repository.NewPackSizesRepository(db)
		_, createErr := repo.Create(ctx, []int{100, 200}, "test")
		require.NoError(t, createErr)

		body := []byte(`{"items_ordered": 150, "pack_sizes": [50, 100, 200]}`)
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response dto.SuccessResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		dataBytes, _ := json.Marshal(response.Data)
		var packResult model.PackResult
		err = json.Unmarshal(dataBytes, &packResult)
		require.NoError(t, err)
		assert.Equal(t, 150, packResult.OrderedItems)
	})
}

func TestHandler_CalculatePacks_WithLogging_Integration(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Use shared container with unique database name
	dbName := sanitizeDBNameForHTTP(t.Name())
	router, db := setupHandlerWithMongoDBIntegrationRouter(dbName)
	defer func() {
		_ = db.Close(ctx)
	}()

	t.Run("request creates log entry", func(t *testing.T) {
		body := []byte(`{"items_ordered": 100}`)
		req := httptest.NewRequest(http.MethodPost, "/api/calculate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		time.Sleep(100 * time.Millisecond)

		logsRepo := repository.NewLogsRepository(db)
		opts := repository.LogQueryOptions{
			Path: "/api/calculate",
		}
		logs, err := logsRepo.Query(ctx, opts)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(logs), 1)
	})
}
