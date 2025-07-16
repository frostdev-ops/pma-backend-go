package homeassistant

import (
	"context"
	"testing"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/sirupsen/logrus"
)

// Mock config repository for testing
type mockConfigRepo struct {
	configs map[string]*models.SystemConfig
}

func (m *mockConfigRepo) Get(ctx context.Context, key string) (*models.SystemConfig, error) {
	if config, exists := m.configs[key]; exists {
		return config, nil
	}
	return nil, nil
}

func (m *mockConfigRepo) Set(ctx context.Context, config *models.SystemConfig) error {
	m.configs[config.Key] = config
	return nil
}

func (m *mockConfigRepo) GetAll(ctx context.Context) ([]*models.SystemConfig, error) {
	var configs []*models.SystemConfig
	for _, config := range m.configs {
		configs = append(configs, config)
	}
	return configs, nil
}

func (m *mockConfigRepo) Delete(ctx context.Context, key string) error {
	delete(m.configs, key)
	return nil
}

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}

	mockRepo := &mockConfigRepo{configs: make(map[string]*models.SystemConfig)}
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	client, err := NewClient(cfg, mockRepo, logger)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be created")
	}
}

func TestClientInitialize(t *testing.T) {
	cfg := &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}

	mockRepo := &mockConfigRepo{configs: make(map[string]*models.SystemConfig)}
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	client, err := NewClient(cfg, mockRepo, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// This will fail because we don't have a real HA instance,
	// but it should get past the initial configuration steps
	ctx := context.Background()
	err = client.Initialize(ctx)

	// We expect this to fail on health check since there's no real HA instance
	if err == nil {
		t.Log("Initialization succeeded (unexpected with mock)")
	} else {
		t.Logf("Initialization failed as expected: %v", err)
	}

	// Test that configuration was loaded
	if client.baseURL == "" {
		t.Error("Expected baseURL to be set")
	}

	if client.token == "" {
		t.Error("Expected token to be set")
	}
}

func TestClientWithDatabaseToken(t *testing.T) {
	cfg := &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL: "http://localhost:8123",
			// No token in config
		},
	}

	mockRepo := &mockConfigRepo{
		configs: map[string]*models.SystemConfig{
			"home_assistant_token": {
				Key:         "home_assistant_token",
				Value:       "database-token",
				Encrypted:   false,
				Description: "HA Token from DB",
				UpdatedAt:   time.Now(),
			},
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	client, err := NewClient(cfg, mockRepo, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	token, err := client.getAccessToken(ctx)
	if err != nil {
		t.Fatalf("Failed to get token: %v", err)
	}

	if token != "database-token" {
		t.Errorf("Expected 'database-token', got '%s'", token)
	}
}

func TestClientConnectionInfo(t *testing.T) {
	cfg := &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}

	mockRepo := &mockConfigRepo{configs: make(map[string]*models.SystemConfig)}
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	client, err := NewClient(cfg, mockRepo, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	info := client.GetConnectionInfo()

	// Before initialization
	if info["rest_available"].(bool) {
		t.Error("Expected rest_available to be false before initialization")
	}

	if info["websocket_available"].(bool) {
		t.Error("Expected websocket_available to be false before initialization")
	}
}

func TestErrorTypes(t *testing.T) {
	// Test that our error types work correctly
	err := ErrUnauthorized
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}

	if !IsAuthError(err) {
		t.Error("Expected IsAuthError to return true for ErrUnauthorized")
	}

	if IsConnectionError(err) {
		t.Error("Expected IsConnectionError to return false for ErrUnauthorized")
	}

	connectionErr := ErrConnectionFailed
	if !IsConnectionError(connectionErr) {
		t.Error("Expected IsConnectionError to return true for ErrConnectionFailed")
	}
}
