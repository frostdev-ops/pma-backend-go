package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// HAEntity represents a HomeAssistant entity
type HAEntity struct {
	EntityID     string                 `json:"entity_id"`
	State        string                 `json:"state"`
	FriendlyName string                 `json:"friendly_name"`
	Domain       string                 `json:"domain"`
	Attributes   map[string]interface{} `json:"attributes"`
	LastChanged  time.Time              `json:"last_changed"`
	LastUpdated  time.Time              `json:"last_updated"`
	Context      map[string]interface{} `json:"context"`
}

// HAArea represents a HomeAssistant area
type HAArea struct {
	ID      string   `json:"area_id"`
	Name    string   `json:"name"`
	Icon    string   `json:"icon,omitempty"`
	Aliases []string `json:"aliases,omitempty"`
}

// HADevice represents a HomeAssistant device
type HADevice struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	NameByUser    *string         `json:"name_by_user"`
	Model         *string         `json:"model"`
	Manufacturer  *string         `json:"manufacturer"`
	SWVersion     *string         `json:"sw_version"`
	HWVersion     *string         `json:"hw_version"`
	ViaDeviceID   *string         `json:"via_device_id"`
	AreaID        *string         `json:"area_id"`
	ConfigEntries []string        `json:"config_entries"`
	Connections   [][]interface{} `json:"connections"`
	Identifiers   [][]interface{} `json:"identifiers"`
	DisabledBy    *string         `json:"disabled_by"`
	EntryType     *string         `json:"entry_type"`
}

// HAServiceCall represents a service call to HomeAssistant
type HAServiceCall struct {
	Domain      string                 `json:"domain"`
	Service     string                 `json:"service"`
	ServiceData map[string]interface{} `json:"service_data,omitempty"`
	Target      *HAServiceTarget       `json:"target,omitempty"`
}

// HAServiceTarget represents the target for a service call
type HAServiceTarget struct {
	EntityID []string `json:"entity_id,omitempty"`
	DeviceID []string `json:"device_id,omitempty"`
	AreaID   []string `json:"area_id,omitempty"`
}

// HAWebSocketMessage represents a WebSocket message
type HAWebSocketMessage struct {
	ID      int               `json:"id,omitempty"`
	Type    string            `json:"type"`
	Success *bool             `json:"success,omitempty"`
	Result  interface{}       `json:"result,omitempty"`
	Error   *HAWebSocketError `json:"error,omitempty"`
	Event   *HAEvent          `json:"event,omitempty"`
}

// HAWebSocketError represents an error in WebSocket communication
type HAWebSocketError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HAEvent represents an event from HomeAssistant
type HAEvent struct {
	EventType string                 `json:"event_type"`
	Data      map[string]interface{} `json:"data"`
	Origin    string                 `json:"origin"`
	TimeFired time.Time              `json:"time_fired"`
	Context   map[string]interface{} `json:"context"`
}

// HAStateChangeEvent represents a Home Assistant state change event
type HAStateChangeEvent struct {
	EventType string            `json:"event_type"`
	Data      HAStateChangeData `json:"data"`
	Origin    string            `json:"origin"`
	TimeFired string            `json:"time_fired"`
}

// HAStateChangeData represents the data in a state change event
type HAStateChangeData struct {
	EntityID string                 `json:"entity_id"`
	NewState map[string]interface{} `json:"new_state"`
	OldState map[string]interface{} `json:"old_state"`
}

// HAClientWrapper wraps HomeAssistant API interactions
type HAClientWrapper struct {
	baseURL      string
	token        string
	httpClient   *http.Client // For regular API calls
	wsHttpClient *http.Client // Separate client for WebSocket-related HTTP calls
	wsConn       *websocket.Conn
	wsConnected  bool
	wsMessageID  int
	eventChan    chan HAStateChangeEvent // Channel for state change events
	stopChan     chan bool               // Channel to stop WebSocket listening
	logger       *logrus.Logger
	mutex        sync.RWMutex
}

// NewHAClientWrapper creates a new Home Assistant client wrapper
func NewHAClientWrapper(config *config.Config, logger *logrus.Logger) *HAClientWrapper {
	return &HAClientWrapper{
		baseURL:      getHABaseURL(config),
		token:        getHAToken(config),
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		wsHttpClient: &http.Client{Timeout: 10 * time.Second}, // Shorter timeout for WebSocket auth
		wsConnected:  false,
		wsMessageID:  1,
		eventChan:    make(chan HAStateChangeEvent, 100), // Buffer for events
		stopChan:     make(chan bool, 1),
		logger:       logger,
	}
}

// Connect establishes connection to Home Assistant
func (c *HAClientWrapper) Connect(ctx context.Context) error {
	c.logger.Info("🔴 CLIENT Connect method starting...")

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.logger.Info("🔴 CLIENT mutex acquired, connecting to Home Assistant API and WebSocket...")

	// Create a context with timeout for the entire connection process
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Test HTTP API connection first
	c.logger.Info("🔴 CLIENT testing Home Assistant HTTP API connection...")
	if err := c.testConnection(connectCtx); err != nil {
		c.wsConnected = false
		c.logger.WithError(err).Error("❌ CLIENT Home Assistant HTTP API connection failed")
		return fmt.Errorf("HTTP API connection failed: %w", err)
	}
	c.logger.Info("✅ CLIENT Home Assistant HTTP API connection successful")

	// Establish WebSocket connection
	c.logger.Info("🔴 CLIENT establishing Home Assistant WebSocket connection...")
	if err := c.connectWebSocket(connectCtx); err != nil {
		c.wsConnected = false
		c.logger.WithError(err).Error("❌ CLIENT Home Assistant WebSocket connection failed")
		return fmt.Errorf("WebSocket connection failed: %w", err)
	}
	c.logger.Info("✅ CLIENT Home Assistant WebSocket connection successful")

	c.wsConnected = true
	c.logger.Info("Successfully connected to Home Assistant")

	// Start WebSocket message handling
	c.logger.Info("🔴 CLIENT starting WebSocket message handler...")

	// Stop any existing WebSocket handler goroutine
	if c.stopChan != nil {
		select {
		case c.stopChan <- true:
			c.logger.Info("🔴 CLIENT stopping existing WebSocket handler")
		default:
		}
		// Wait for the goroutine to stop
		time.Sleep(1 * time.Second)
	}

	// Create a new stopChan for the new goroutine
	c.stopChan = make(chan bool, 1)
	go c.handleWebSocketMessages()

	// Subscribe to state change events
	c.logger.Info("🔴 CLIENT subscribing to Home Assistant state change events...")
	if err := c.subscribeToStateChanges(); err != nil {
		c.logger.WithError(err).Warn("❌ CLIENT failed to subscribe to state changes, will retry")
		// Don't fail completely - we can still work with HTTP API
	} else {
		c.logger.Info("✅ CLIENT successfully subscribed to Home Assistant state change events")
	}

	c.logger.Info("🎉 CLIENT Home Assistant connection setup completed successfully")
	return nil
}

// IsConnected returns true if connected to Home Assistant
func (c *HAClientWrapper) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.wsConnected
}

// Disconnect closes the connection to HomeAssistant
func (c *HAClientWrapper) Disconnect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Signal stop to WebSocket handler
	select {
	case c.stopChan <- true:
	default:
	}

	// Wait a moment for the goroutine to stop
	time.Sleep(500 * time.Millisecond)

	if c.wsConn != nil {
		c.wsConn.Close()
		c.wsConn = nil
	}

	c.wsConnected = false
	c.logger.Info("Disconnected from Home Assistant")
	return nil
}

// GetAllEntities fetches all entities from HomeAssistant
func (c *HAClientWrapper) GetAllEntities(ctx context.Context) ([]*HAEntity, error) {
	url := fmt.Sprintf("%s/api/states", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var states []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&states); err != nil {
		return nil, err
	}

	var entities []*HAEntity
	for _, state := range states {
		entity := c.convertStateToEntity(state)
		if entity != nil {
			entities = append(entities, entity)
		}
	}

	return entities, nil
}

// GetAllAreas fetches all areas from HomeAssistant
func (c *HAClientWrapper) GetAllAreas(ctx context.Context) ([]*HAArea, error) {
	url := fmt.Sprintf("%s/api/config/area_registry", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var areas []*HAArea
	if err := json.NewDecoder(resp.Body).Decode(&areas); err != nil {
		return nil, err
	}

	return areas, nil
}

// CallService calls a HomeAssistant service
func (c *HAClientWrapper) CallService(ctx context.Context, domain, service, entityID string, serviceData map[string]interface{}) error {
	url := fmt.Sprintf("%s/api/services/%s/%s", c.baseURL, domain, service)

	payload := map[string]interface{}{
		"entity_id": entityID,
	}

	// Merge service data
	for k, v := range serviceData {
		payload[k] = v
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("service call failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetAllEntitiesHTTPOnly fetches entities using only HTTP API, no WebSocket required
func (c *HAClientWrapper) GetAllEntitiesHTTPOnly(ctx context.Context) ([]*HAEntity, error) {
	url := fmt.Sprintf("%s/api/states", c.baseURL)
	c.logger.WithField("url", url).Debug("Fetching entities via HTTP-only mode")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var states []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&states); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var entities []*HAEntity
	for _, state := range states {
		entity := c.convertStateToEntity(state)
		if entity != nil {
			entities = append(entities, entity)
		}
	}

	c.logger.WithField("entity_count", len(entities)).Debug("Successfully fetched entities via HTTP-only mode")
	return entities, nil
}

// Helper methods

func (c *HAClientWrapper) testConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/", c.baseURL)
	c.logger.WithField("url", url).Debug("Testing Home Assistant HTTP API connection")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	// Use the dedicated WebSocket HTTP client for auth testing to avoid conflicts
	resp, err := c.wsHttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	c.logger.WithField("status_code", resp.StatusCode).Debug("Home Assistant HTTP API response received")

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Debug("Home Assistant HTTP API authentication successful")
	return nil
}

func (c *HAClientWrapper) connectWebSocket(ctx context.Context) error {
	wsURL := strings.Replace(c.baseURL, "http", "ws", 1) + "/api/websocket"
	c.logger.WithField("url", wsURL).Debug("Connecting to Home Assistant WebSocket")

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}

	// Set a deadline for the WebSocket connection
	deadline, ok := ctx.Deadline()
	if ok {
		dialer.HandshakeTimeout = time.Until(deadline)
		if dialer.HandshakeTimeout <= 0 {
			return fmt.Errorf("context deadline exceeded before WebSocket connection")
		}
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to dial WebSocket: %w", err)
	}

	c.wsConn = conn
	c.logger.Debug("WebSocket connection established, starting authentication")

	// Handle auth flow with timeout
	authCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Set connection deadline for auth
	if deadline, ok := authCtx.Deadline(); ok {
		c.wsConn.SetReadDeadline(deadline)
		c.wsConn.SetWriteDeadline(deadline)
	}

	if err := c.authenticateWebSocket(); err != nil {
		c.wsConn.Close()
		c.wsConn = nil
		return fmt.Errorf("WebSocket authentication failed: %w", err)
	}

	// Reset deadlines after auth
	c.wsConn.SetReadDeadline(time.Time{})
	c.wsConn.SetWriteDeadline(time.Time{})

	c.logger.Debug("WebSocket authentication completed successfully")
	return nil
}

func (c *HAClientWrapper) authenticateWebSocket() error {
	// Read auth_required message
	var authMsg HAWebSocketMessage
	if err := c.wsConn.ReadJSON(&authMsg); err != nil {
		return err
	}

	if authMsg.Type != "auth_required" {
		return fmt.Errorf("expected auth_required, got %s", authMsg.Type)
	}

	// Send auth message (this is a simplified version)
	if err := c.wsConn.WriteJSON(map[string]interface{}{
		"type":         "auth",
		"access_token": c.token,
	}); err != nil {
		return err
	}

	// Read auth result
	var authResult HAWebSocketMessage
	if err := c.wsConn.ReadJSON(&authResult); err != nil {
		return err
	}

	if authResult.Type != "auth_ok" {
		return fmt.Errorf("authentication failed: %s", authResult.Type)
	}

	return nil
}

func (c *HAClientWrapper) convertStateToEntity(state map[string]interface{}) *HAEntity {
	entityID, ok := state["entity_id"].(string)
	if !ok {
		return nil
	}

	parts := strings.Split(entityID, ".")
	if len(parts) != 2 {
		return nil
	}

	domain := parts[0]
	stateValue, _ := state["state"].(string)

	attributes, _ := state["attributes"].(map[string]interface{})
	if attributes == nil {
		attributes = make(map[string]interface{})
	}

	friendlyName, _ := attributes["friendly_name"].(string)
	if friendlyName == "" {
		friendlyName = entityID
	}

	entity := &HAEntity{
		EntityID:     entityID,
		Domain:       domain,
		State:        stateValue,
		FriendlyName: friendlyName,
		Attributes:   attributes,
		LastUpdated:  time.Now(), // Simplified - should parse from API
		LastChanged:  time.Now(), // Simplified - should parse from API
	}

	return entity
}

func getHABaseURL(config *config.Config) string {
	// Try to get from config, fallback to default
	if config.HomeAssistant.URL != "" {
		return config.HomeAssistant.URL
	}
	return "http://homeassistant.local:8123"
}

func getHAToken(config *config.Config) string {
	// Try to get from config
	return config.HomeAssistant.Token
}

// subscribeToStateChanges subscribes to Home Assistant state change events
// Note: Caller must hold mutex
func (c *HAClientWrapper) subscribeToStateChanges() error {
	if c.wsConn == nil {
		return fmt.Errorf("WebSocket connection not established")
	}

	// Subscribe to state_changed events
	subscribeMsg := map[string]interface{}{
		"id":         c.wsMessageID,
		"type":       "subscribe_events",
		"event_type": "state_changed",
	}
	c.wsMessageID++

	if err := c.wsConn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("failed to send subscribe message: %w", err)
	}

	c.logger.Info("Subscribed to Home Assistant state change events")
	return nil
}

// handleWebSocketMessages handles incoming WebSocket messages
func (c *HAClientWrapper) handleWebSocketMessages() {
	defer func() {
		if r := recover(); r != nil {
			c.logger.WithField("panic", r).Error("WebSocket message handler panicked")
		}
		// Always mark as disconnected on exit
		c.mutex.Lock()
		c.wsConnected = false
		c.wsConn = nil
		c.mutex.Unlock()
		c.logger.Info("WebSocket message handler has shut down.")
	}()

	c.logger.Info("Starting WebSocket message handler")

	consecutiveErrors := 0
	maxConsecutiveErrors := 10 // Increased tolerance
	errorBackoff := time.Second

	for {
		select {
		case <-c.stopChan:
			c.logger.Info("WebSocket message handler stopping via stopChan")
			return
		default:
			// Non-blocking check for stop signal
		}

		c.mutex.RLock()
		wsConn := c.wsConn
		c.mutex.RUnlock()

		if wsConn == nil {
			c.logger.Warn("WebSocket connection is nil, stopping message handler")
			return
		}

		// Set read deadline to prevent blocking forever
		wsConn.SetReadDeadline(time.Now().Add(60 * time.Second)) // Longer deadline

		var message map[string]interface{}
		if err := wsConn.ReadJSON(&message); err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) ||
				strings.Contains(err.Error(), "use of closed network connection") {
				c.logger.Info("WebSocket connection closed gracefully")
				return
			}

			consecutiveErrors++
			c.logger.WithError(err).Warnf("Failed to read WebSocket message (error #%d)", consecutiveErrors)

			if consecutiveErrors >= maxConsecutiveErrors {
				c.logger.Error("Too many consecutive WebSocket read errors, closing connection and stopping handler")
				wsConn.Close() // Attempt to close the faulty connection
				return         // Exit the goroutine
			}

			// Exponential backoff
			time.Sleep(errorBackoff)
			errorBackoff *= 2
			if errorBackoff > 30*time.Second {
				errorBackoff = 30 * time.Second
			}
			continue
		}

		consecutiveErrors = 0
		errorBackoff = time.Second

		c.processWebSocketMessage(message)
	}
}

// processWebSocketMessage processes an incoming WebSocket message
func (c *HAClientWrapper) processWebSocketMessage(message map[string]interface{}) {
	msgType, ok := message["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "event":
		c.handleEventMessage(message)
	case "result":
		// Handle subscription confirmation or other results
		id, hasID := message["id"].(float64)
		success, hasSuccess := message["success"].(bool)
		if hasID && hasSuccess {
			if success {
				c.logger.WithField("message_id", int(id)).Debug("WebSocket command successful")
			} else {
				c.logger.WithField("message_id", int(id)).Warn("WebSocket command failed")
			}
		}
	case "auth_required", "auth_ok", "auth_invalid":
		// Authentication messages are handled separately
		c.logger.WithField("auth_type", msgType).Debug("WebSocket auth message received")
	default:
		c.logger.WithField("message_type", msgType).Debug("Unknown WebSocket message type")
	}
}

// handleEventMessage handles WebSocket event messages
func (c *HAClientWrapper) handleEventMessage(message map[string]interface{}) {
	event, ok := message["event"].(map[string]interface{})
	if !ok {
		return
	}

	eventType, ok := event["event_type"].(string)
	if !ok || eventType != "state_changed" {
		return
	}

	// Extract event data
	data, ok := event["data"].(map[string]interface{})
	if !ok {
		return
	}

	entityID, ok := data["entity_id"].(string)
	if !ok {
		return
	}

	newState, hasNewState := data["new_state"].(map[string]interface{})
	oldState, hasOldState := data["old_state"].(map[string]interface{})

	if !hasNewState {
		return // Entity might have been removed
	}

	// Create state change event
	stateChangeEvent := HAStateChangeEvent{
		EventType: eventType,
		Data: HAStateChangeData{
			EntityID: entityID,
			NewState: newState,
			OldState: oldState,
		},
	}

	if origin, ok := event["origin"].(string); ok {
		stateChangeEvent.Origin = origin
	}
	if timeFired, ok := event["time_fired"].(string); ok {
		stateChangeEvent.TimeFired = timeFired
	}

	// Send event to channel (non-blocking)
	select {
	case c.eventChan <- stateChangeEvent:
		c.logger.WithFields(logrus.Fields{
			"entity_id": entityID,
			"new_state": newState["state"],
			"old_state": func() interface{} {
				if hasOldState {
					return oldState["state"]
				}
				return nil
			}(),
		}).Debug("State change event queued")
	default:
		c.logger.WithField("entity_id", entityID).Warn("State change event channel full, dropping event")
	}
}

// GetStateChangeEvents returns the channel for state change events
func (c *HAClientWrapper) GetStateChangeEvents() <-chan HAStateChangeEvent {
	return c.eventChan
}
