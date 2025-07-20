package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frostdev-ops/pma-backend-go/internal/api"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/frostdev-ops/pma-backend-go/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ErrorResponse represents the enhanced error response structure
type ErrorResponse struct {
	Success   bool                   `json:"success"`
	Error     string                 `json:"error"`
	Code      int                    `json:"code"`
	Timestamp string                 `json:"timestamp"`
	Request   map[string]interface{} `json:"request"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

func TestEnhancedErrorHandling(t *testing.T) {
	// Setup test router
	cfg := &config.Config{
		Server: config.ServerConfig{
			Mode: "test",
		},
	}

	log := logger.New()
	wsHub := websocket.NewHub(log.Logger)

	// Create a mock repositories struct
	repos := &database.Repositories{}

	router := api.NewRouter(cfg, repos, log, wsHub, nil)

	t.Run("404 Not Found with suggestions", func(t *testing.T) {
		// Test a non-existent endpoint that should trigger suggestions
		req := httptest.NewRequest("GET", "/api/v1/entitie", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify response structure
		assert.False(t, response.Success)
		assert.Equal(t, 404, response.Code)
		assert.Contains(t, response.Error, "Endpoint not found")
		assert.Contains(t, response.Error, "GET /api/v1/entitie")
		assert.NotEmpty(t, response.Timestamp)

		// Verify request information
		assert.Equal(t, "GET", response.Request["method"])
		assert.Equal(t, "/api/v1/entitie", response.Request["path"])

		// Verify suggestions are provided
		if response.Details != nil {
			suggestions, exists := response.Details["suggestions"]
			if exists {
				suggestionsSlice := suggestions.([]interface{})
				assert.Greater(t, len(suggestionsSlice), 0)
				// Should suggest the correct endpoint
				found := false
				for _, suggestion := range suggestionsSlice {
					if suggestion == "/api/v1/entities" {
						found = true
						break
					}
				}
				assert.True(t, found, "Should suggest /api/v1/entities")
			}
		}
	})

	t.Run("405 Method Not Allowed", func(t *testing.T) {
		// Test an unsupported method on an existing endpoint
		req := httptest.NewRequest("POST", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify response structure
		assert.False(t, response.Success)
		assert.Equal(t, 405, response.Code)
		assert.Contains(t, response.Error, "Method POST not allowed")
		assert.Contains(t, response.Error, "/health")
		assert.NotEmpty(t, response.Timestamp)

		// Verify request information
		assert.Equal(t, "POST", response.Request["method"])
		assert.Equal(t, "/health", response.Request["path"])

		// Verify helpful message
		if response.Details != nil {
			message, exists := response.Details["message"]
			if exists {
				assert.Contains(t, message.(string), "HTTP method is not supported")
			}
		}
	})

	t.Run("Health endpoint includes error handling info", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify health response includes error handling information
		assert.True(t, response["success"].(bool))

		data := response["data"].(map[string]interface{})
		errorHandling := data["error_handling"].(map[string]interface{})

		assert.True(t, errorHandling["enhanced_404"].(bool))
		assert.True(t, errorHandling["enhanced_405"].(bool))
		assert.True(t, errorHandling["detailed_logging"].(bool))
		assert.True(t, errorHandling["error_suggestions"].(bool))
	})

	t.Run("404 for completely random path", func(t *testing.T) {
		// Test a completely random path that shouldn't get suggestions
		req := httptest.NewRequest("GET", "/completely/random/path", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify basic response structure
		assert.False(t, response.Success)
		assert.Equal(t, 404, response.Code)
		assert.Contains(t, response.Error, "Endpoint not found")
		assert.NotEmpty(t, response.Timestamp)

		// Verify request information
		assert.Equal(t, "GET", response.Request["method"])
		assert.Equal(t, "/completely/random/path", response.Request["path"])
	})
}
