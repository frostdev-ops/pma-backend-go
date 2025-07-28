package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteEntityActionEndpoint(t *testing.T) {
	// Setup test environment
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	config := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
			Mode: "test",
		},
	}

	typeRegistry := types.NewPMATypeRegistry(logger)
	unifiedService := unified.NewUnifiedEntityService(typeRegistry, config, logger)
	router := setupTestRouter(config, unifiedService, logger)

	t.Run("ExecuteEntityAction endpoint exists", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"action": "turn_on",
		}
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/entities/test_entity/action", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should not return 404 (endpoint exists)
		assert.NotEqual(t, http.StatusNotFound, w.Code, "ExecuteEntityAction endpoint should be registered")
	})

	t.Run("ExecuteEntityAction validates JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/entities/test_entity/action", bytes.NewBufferString(`{invalid json`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 for invalid JSON
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ExecuteEntityAction requires action field", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"parameters": map[string]interface{}{
				"brightness": 128,
			},
		}
		body, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/entities/test_entity/action", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 for missing action field
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestNewEntityEndpoints(t *testing.T) {
	// Setup test environment
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	config := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
			Mode: "test",
		},
	}

	typeRegistry := types.NewPMATypeRegistry(logger)
	unifiedService := unified.NewUnifiedEntityService(typeRegistry, config, logger)
	router := setupTestRouter(config, unifiedService, logger)

	// Test that all new entity endpoints are registered
	testEndpoints := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/entities/test_entity/action"},
		{"GET", "/api/v1/entities/search"},
		{"GET", "/api/v1/entities/types"},
		{"GET", "/api/v1/entities/capabilities"},
		{"GET", "/api/v1/entities/type/light"},
		{"GET", "/api/v1/entities/source/homeassistant"},
		{"GET", "/api/v1/entities/room/living_room"},
	}

	for _, endpoint := range testEndpoints {
		t.Run(endpoint.method+" "+endpoint.path, func(t *testing.T) {
			var req *http.Request
			if endpoint.method == "POST" {
				requestBody := map[string]interface{}{
					"action": "turn_on",
				}
				body, _ := json.Marshal(requestBody)
				req = httptest.NewRequest(endpoint.method, endpoint.path, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(endpoint.method, endpoint.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Verify that the endpoint exists (not 404)
			assert.NotEqual(t, http.StatusNotFound, w.Code, "Endpoint should be registered: %s %s", endpoint.method, endpoint.path)
		})
	}
}

func TestExecuteEntityActionRequestFormat(t *testing.T) {
	// Setup test environment
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	config := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
			Mode: "test",
		},
	}

	typeRegistry := types.NewPMATypeRegistry(logger)
	unifiedService := unified.NewUnifiedEntityService(typeRegistry, config, logger)
	router := setupTestRouter(config, unifiedService, logger)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "Valid action request",
			requestBody:    `{"action": "turn_on"}`,
			expectedStatus: http.StatusInternalServerError, // Will fail due to missing entity, but validates JSON
		},
		{
			name:           "Valid action with parameters",
			requestBody:    `{"action": "set_brightness", "parameters": {"brightness": 128}}`,
			expectedStatus: http.StatusInternalServerError, // Will fail due to missing entity, but validates JSON
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{"action": "turn_on"`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty action",
			requestBody:    `{"action": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing action field",
			requestBody:    `{"parameters": {"brightness": 100}}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/entities/test_entity/action", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
