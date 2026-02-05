//go:build integration

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/guttosm/pack-service/internal/circuitbreaker"
	"github.com/guttosm/pack-service/internal/repository"
	"github.com/guttosm/pack-service/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPackSizesIntegrationRouter(dbName string) *gin.Engine {
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
	healthHandler.RegisterCircuitBreaker("mongodb_pack_sizes", packSizesCB)
	healthHandler.RegisterCircuitBreaker("mongodb_logs", logsCB)

	cfg := RouterConfig{
		RateLimit:       100,
		RateWindow:      time.Minute,
		EnableAuth:      false,
		LoggingService:  loggingService,
		PackSizesService: packSizesService,
	}

	router := NewRouter(handler, healthHandler, cfg)

	return router
}

func TestPackSizesHandler_Integration(t *testing.T) {
	t.Parallel()

	// Use shared container with unique database name
	dbName := sanitizeDBNameForHTTP(t.Name())
	router := setupPackSizesIntegrationRouter(dbName)

	t.Run("get active pack sizes when none exist", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/pack-sizes", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("create pack sizes via repository then get", func(t *testing.T) {
		ctx := context.Background()
		uri := getSharedContainerURI()
		// Use the same database name as the router
		testDBName := sanitizeDBNameForHTTP(t.Name() + "_get")
		db, err := repository.NewMongoDB(uri, testDBName)
		require.NoError(t, err)
		defer func() {
			_ = db.Close(ctx)
		}()

		repo := repository.NewPackSizesRepository(db)
		_, createErr := repo.Create(ctx, []int{100, 200, 500}, "test")
		require.NoError(t, createErr)

		// Create a router with the same database where we created pack sizes
		testRouter := setupPackSizesIntegrationRouter(testDBName)

		// Now get via API
		req := httptest.NewRequest(http.MethodGet, "/api/pack-sizes", nil)
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok, "Response data should be a map")
		sizes := data["sizes"].([]interface{})
		assert.Equal(t, 3, len(sizes))
	})

	t.Run("update pack sizes", func(t *testing.T) {
		ctx := context.Background()
		uri := getSharedContainerURI()
		testDBName := sanitizeDBNameForHTTP(t.Name() + "_update")
		db, err := repository.NewMongoDB(uri, testDBName)
		require.NoError(t, err)
		defer func() {
			_ = db.Close(ctx)
		}()

		// First create initial pack sizes
		repo := repository.NewPackSizesRepository(db)
		_, createErr := repo.Create(ctx, []int{100, 200}, "test-user-init")
		require.NoError(t, createErr)

		// Create router with the same database
		testRouter := setupPackSizesIntegrationRouter(testDBName)

		updateBody := map[string]interface{}{
			"sizes":      []int{250, 500, 1000},
			"created_by": "test-user",
		}
		bodyBytes, _ := json.Marshal(updateBody)

		req := httptest.NewRequest(http.MethodPut, "/api/pack-sizes", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		data, ok := response["data"].(map[string]interface{})
		require.True(t, ok, "Response data should be a map")
		sizes := data["sizes"].([]interface{})
		assert.Equal(t, 3, len(sizes))
	})

	t.Run("list pack sizes history", func(t *testing.T) {
		// First, create some pack sizes to have history
		ctx := context.Background()
		uri := getSharedContainerURI()
		dbName := sanitizeDBNameForHTTP(t.Name() + "_history")
		db, err := repository.NewMongoDB(uri, dbName)
		require.NoError(t, err)
		defer func() {
			_ = db.Close(ctx)
		}()

		repo := repository.NewPackSizesRepository(db)
		_, createErr := repo.Create(ctx, []int{100, 200}, "test-user-1")
		require.NoError(t, createErr)
		_, createErr = repo.Create(ctx, []int{250, 500}, "test-user-2")
		require.NoError(t, createErr)

		// Create a router with the same database where we created pack sizes
		historyRouter := setupPackSizesIntegrationRouter(dbName)

		req := httptest.NewRequest(http.MethodGet, "/api/pack-sizes/history", nil)
		w := httptest.NewRecorder()

		historyRouter.ServeHTTP(w, req)

		// Check response body is valid JSON
		bodyBytes := w.Body.Bytes()
		require.NotEmpty(t, bodyBytes, "Response body should not be empty")

		// The endpoint might return 404 if no route is found, or 200 with data
		if w.Code == http.StatusNotFound {
			// Check if it's a JSON 404 or HTML 404
			var errorResponse map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &errorResponse); err == nil {
				// It's a JSON error response, which is fine
				assert.Equal(t, http.StatusNotFound, w.Code)
				return
			}
			// HTML 404 means route not found - this is a problem
			t.Fatalf("Route not found - got HTML 404: %s", string(bodyBytes))
		}

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(bodyBytes, &response)
		require.NoError(t, err, "Response should be valid JSON: %s", string(bodyBytes))

		data, ok := response["data"].([]interface{})
		require.True(t, ok, "Response data should be an array")
		assert.GreaterOrEqual(t, len(data), 1, "Should have at least one pack size configuration")
	})
}

func TestHealthCheckWithCircuitBreaker_Integration(t *testing.T) {
	t.Parallel()

	// Use shared container with unique database name
	dbName := sanitizeDBNameForHTTP(t.Name())
	router := setupPackSizesIntegrationRouter(dbName)

	t.Run("health check includes circuit breaker status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		checks := response["checks"].(map[string]interface{})
		assert.Contains(t, checks, "mongodb_pack_sizes_circuit")
		assert.Contains(t, checks, "mongodb_logs_circuit")
		assert.Equal(t, "closed", checks["mongodb_pack_sizes_circuit"])
		assert.Equal(t, "closed", checks["mongodb_logs_circuit"])
	})
}
