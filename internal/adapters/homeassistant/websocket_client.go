package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
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

// wsClient implements WebSocketClient
type wsClient struct {
	baseURL           string
	token             string
	logger            *logrus.Logger
	connected         bool
	conn              *websocket.Conn
	mu                sync.RWMutex
	messageID         int64
	subscriptions     map[int]interface{}
	connectionHandler ConnectionStateHandler
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(baseURL, token string, logger *logrus.Logger) WebSocketClient {
	return &wsClient{
		baseURL:       baseURL,
		token:         token,
		logger:        logger,
		connected:     false,
		subscriptions: make(map[int]interface{}),
	}
}

func (c *wsClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// Parse the base URL and create WebSocket URL
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	// Convert HTTP(S) to WS(S)
	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}

	wsURL := fmt.Sprintf("%s://%s/api/websocket", scheme, u.Host)

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	c.conn = conn
	c.connected = true

	// Authenticate
	if err := c.authenticate(); err != nil {
		c.conn.Close()
		c.connected = false
		return fmt.Errorf("authentication failed: %w", err)
	}

	c.logger.Info("WebSocket connected and authenticated")

	if c.connectionHandler != nil {
		c.connectionHandler(true)
	}

	return nil
}

func (c *wsClient) authenticate() error {
	// First, read the auth_required message
	var authRequired map[string]interface{}
	if err := c.conn.ReadJSON(&authRequired); err != nil {
		return fmt.Errorf("failed to read auth_required: %w", err)
	}

	// Send authentication
	auth := map[string]interface{}{
		"type":         "auth",
		"access_token": c.token,
	}

	if err := c.conn.WriteJSON(auth); err != nil {
		return fmt.Errorf("failed to send auth: %w", err)
	}

	// Read auth response
	var authResponse map[string]interface{}
	if err := c.conn.ReadJSON(&authResponse); err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	if authResponse["type"] != "auth_ok" {
		return fmt.Errorf("authentication failed: %v", authResponse)
	}

	return nil
}

func (c *wsClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.connected = false
	c.conn = nil

	if c.connectionHandler != nil {
		c.connectionHandler(false)
	}

	c.logger.Info("WebSocket disconnected")
	return err
}

func (c *wsClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *wsClient) SubscribeToEvents(eventType string, handler EventHandler) (int, error) {
	if !c.IsConnected() {
		return 0, fmt.Errorf("not connected")
	}

	id := c.nextMessageID()

	subscribeMsg := map[string]interface{}{
		"id":         id,
		"type":       "subscribe_events",
		"event_type": eventType,
	}

	c.mu.Lock()
	c.subscriptions[id] = handler
	c.mu.Unlock()

	return id, c.sendMessage(subscribeMsg)
}

func (c *wsClient) SubscribeToStateChanges(entityID string, handler StateChangeHandler) (int, error) {
	// Subscribe to state_changed events and filter by entity ID
	eventHandler := func(event Event) {
		if event.EventType == "state_changed" {
			if data, ok := event.Data["entity_id"]; ok && data == entityID {
				// Extract old and new states
				var oldState, newState *EntityState
				if old, ok := event.Data["old_state"]; ok {
					if oldStateData, err := json.Marshal(old); err == nil {
						json.Unmarshal(oldStateData, &oldState)
					}
				}
				if new, ok := event.Data["new_state"]; ok {
					if newStateData, err := json.Marshal(new); err == nil {
						json.Unmarshal(newStateData, &newState)
					}
				}
				handler(entityID, oldState, newState)
			}
		}
	}

	return c.SubscribeToEvents("state_changed", eventHandler)
}

func (c *wsClient) Unsubscribe(subscriptionID int) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}

	unsubscribeMsg := map[string]interface{}{
		"id":           c.nextMessageID(),
		"type":         "unsubscribe_events",
		"subscription": subscriptionID,
	}

	c.mu.Lock()
	delete(c.subscriptions, subscriptionID)
	c.mu.Unlock()

	return c.sendMessage(unsubscribeMsg)
}

func (c *wsClient) SendCommand(command interface{}) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}

	// Handle different command types
	switch cmd := command.(type) {
	case *CallServiceMessage:
		cmd.ID = c.nextMessageID()
		cmd.Type = "call_service"
		return c.sendMessage(cmd)

	case *GetStatesMessage:
		cmd.ID = c.nextMessageID()
		cmd.Type = "get_states"
		return c.sendMessage(cmd)

	case *PingMessage:
		cmd.ID = c.nextMessageID()
		cmd.Type = "ping"
		return c.sendMessage(cmd)

	case map[string]interface{}:
		// Generic command - ensure it has an ID and send as-is
		if _, hasID := cmd["id"]; !hasID {
			cmd["id"] = c.nextMessageID()
		}
		return c.sendMessage(cmd)

	default:
		return fmt.Errorf("unsupported command type: %T", command)
	}
}

func (c *wsClient) Ping() error {
	pingMsg := &PingMessage{
		ID:   c.nextMessageID(),
		Type: "ping",
	}
	return c.SendCommand(pingMsg)
}

func (c *wsClient) SetConnectionStateHandler(handler ConnectionStateHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connectionHandler = handler
}

// Helper methods

func (c *wsClient) nextMessageID() int {
	return int(atomic.AddInt64(&c.messageID, 1))
}

func (c *wsClient) sendMessage(message interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return fmt.Errorf("not connected")
	}

	return c.conn.WriteJSON(message)
}
