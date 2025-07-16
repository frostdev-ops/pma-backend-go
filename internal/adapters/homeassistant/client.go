package homeassistant

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// Client represents the main Home Assistant client
type Client struct {
	rest       RESTClient
	websocket  WebSocketClient
	configRepo repositories.ConfigRepository
	logger     *logrus.Logger
	config     *config.Config

	baseURL   string
	token     string
	connected bool
	mu        sync.RWMutex
}

// NewClient creates a new Home Assistant client
func NewClient(cfg *config.Config, configRepo repositories.ConfigRepository, logger *logrus.Logger) (*Client, error) {
	if cfg == nil || configRepo == nil || logger == nil {
		return nil, fmt.Errorf("invalid parameters")
	}

	return &Client{
		configRepo: configRepo,
		logger:     logger,
		config:     cfg,
	}, nil
}

// Initialize sets up the client
func (c *Client) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("Initializing Home Assistant client")

	baseURL := c.config.HomeAssistant.URL
	if baseURL == "" {
		return ErrInvalidURL
	}
	c.baseURL = strings.TrimSuffix(baseURL, "/")

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return err
	}
	if token == "" {
		return ErrMissingToken
	}
	c.token = token

	c.rest = NewRESTClient(c.baseURL, c.token, c.logger)
	c.websocket = NewWebSocketClient(c.baseURL, c.token, c.logger)

	if err := c.HealthCheck(ctx); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	c.logger.Info("Home Assistant client initialized")
	return nil
}

// Shutdown shuts down the client
func (c *Client) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.websocket != nil {
		c.websocket.Disconnect()
	}
	c.connected = false
	return nil
}

// HealthCheck verifies connectivity
func (c *Client) HealthCheck(ctx context.Context) error {
	if c.rest == nil {
		return fmt.Errorf("REST client not initialized")
	}

	config, err := c.rest.GetConfig(ctx)
	if err != nil {
		return err
	}

	c.logger.WithField("version", config.Version).Debug("Health check passed")
	return nil
}

// REST API delegation methods
func (c *Client) GetConfig(ctx context.Context) (*HAConfig, error) {
	if c.rest == nil {
		return nil, fmt.Errorf("client not initialized")
	}
	return c.rest.GetConfig(ctx)
}

func (c *Client) GetStates(ctx context.Context) ([]EntityState, error) {
	if c.rest == nil {
		return nil, fmt.Errorf("client not initialized")
	}
	return c.rest.GetStates(ctx)
}

func (c *Client) GetState(ctx context.Context, entityID string) (*EntityState, error) {
	if c.rest == nil {
		return nil, fmt.Errorf("client not initialized")
	}
	return c.rest.GetState(ctx, entityID)
}

func (c *Client) SetState(ctx context.Context, entityID string, state interface{}, attributes map[string]interface{}) error {
	if c.rest == nil {
		return fmt.Errorf("client not initialized")
	}
	return c.rest.SetState(ctx, entityID, state, attributes)
}

func (c *Client) CallService(ctx context.Context, domain, service string, data map[string]interface{}) error {
	if c.rest == nil {
		return fmt.Errorf("client not initialized")
	}
	return c.rest.CallService(ctx, domain, service, data)
}

func (c *Client) GetAreas(ctx context.Context) ([]Area, error) {
	if c.rest == nil {
		return nil, fmt.Errorf("client not initialized")
	}
	return c.rest.GetAreas(ctx)
}

func (c *Client) GetArea(ctx context.Context, areaID string) (*Area, error) {
	if c.rest == nil {
		return nil, fmt.Errorf("client not initialized")
	}
	return c.rest.GetArea(ctx, areaID)
}

func (c *Client) GetDevices(ctx context.Context) ([]Device, error) {
	if c.rest == nil {
		return nil, fmt.Errorf("client not initialized")
	}
	return c.rest.GetDevices(ctx)
}

// WebSocket delegation methods
func (c *Client) SubscribeToEvents(eventType string, handler EventHandler) (int, error) {
	if c.websocket == nil {
		return 0, fmt.Errorf("WebSocket not enabled")
	}
	return c.websocket.SubscribeToEvents(eventType, handler)
}

func (c *Client) SubscribeToStateChanges(entityID string, handler StateChangeHandler) (int, error) {
	if c.websocket == nil {
		return 0, fmt.Errorf("WebSocket not enabled")
	}
	return c.websocket.SubscribeToStateChanges(entityID, handler)
}

func (c *Client) Unsubscribe(subscriptionID int) error {
	if c.websocket == nil {
		return fmt.Errorf("WebSocket not enabled")
	}
	return c.websocket.Unsubscribe(subscriptionID)
}

func (c *Client) IsWebSocketConnected() bool {
	if c.websocket == nil {
		return false
	}
	return c.websocket.IsConnected()
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rest != nil
}

// Helper methods
func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	configEntry, err := c.configRepo.Get(ctx, "home_assistant_token")
	if err == nil && configEntry != nil && configEntry.Value != "" {
		return configEntry.Value, nil
	}

	if c.config.HomeAssistant.Token != "" {
		return c.config.HomeAssistant.Token, nil
	}

	return "", nil
}

func (c *Client) GetConnectionInfo() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"base_url":            c.baseURL,
		"has_token":           c.token != "",
		"rest_available":      c.rest != nil,
		"websocket_available": c.websocket != nil,
		"websocket_connected": c.IsWebSocketConnected(),
		"overall_connected":   c.IsConnected(),
	}
}
