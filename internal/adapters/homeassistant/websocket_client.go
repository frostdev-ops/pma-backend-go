package homeassistant

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

// WebSocketClient interface defines WebSocket operations
type WebSocketClient interface {
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
	SubscribeToEvents(eventType string, handler EventHandler) (int, error)
	SubscribeToStateChanges(entityID string, handler StateChangeHandler) (int, error)
	Unsubscribe(subscriptionID int) error
	SendCommand(command interface{}) error
	Ping() error
	SetConnectionStateHandler(handler ConnectionStateHandler)
}

// wsClient implements WebSocketClient (stub for now)
type wsClient struct {
	baseURL   string
	token     string
	logger    *logrus.Logger
	connected bool
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(baseURL, token string, logger *logrus.Logger) WebSocketClient {
	return &wsClient{
		baseURL:   baseURL,
		token:     token,
		logger:    logger,
		connected: false,
	}
}

func (c *wsClient) Connect(ctx context.Context) error {
	c.logger.Info("WebSocket Connect (stub)")
	c.connected = true
	return nil
}

func (c *wsClient) Disconnect() error {
	c.logger.Info("WebSocket Disconnect (stub)")
	c.connected = false
	return nil
}

func (c *wsClient) IsConnected() bool {
	return c.connected
}

func (c *wsClient) SubscribeToEvents(eventType string, handler EventHandler) (int, error) {
	c.logger.WithField("event_type", eventType).Info("SubscribeToEvents (stub)")
	return 1, nil
}

func (c *wsClient) SubscribeToStateChanges(entityID string, handler StateChangeHandler) (int, error) {
	c.logger.WithField("entity_id", entityID).Info("SubscribeToStateChanges (stub)")
	return 1, nil
}

func (c *wsClient) Unsubscribe(subscriptionID int) error {
	c.logger.WithField("subscription_id", subscriptionID).Info("Unsubscribe (stub)")
	return nil
}

func (c *wsClient) SendCommand(command interface{}) error {
	return fmt.Errorf("not implemented")
}

func (c *wsClient) Ping() error {
	return nil
}

func (c *wsClient) SetConnectionStateHandler(handler ConnectionStateHandler) {
	c.logger.Info("SetConnectionStateHandler (stub)")
}
