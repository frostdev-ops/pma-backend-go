package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// HAClientWrapper wraps HomeAssistant API interactions
type HAClientWrapper struct {
	baseURL    string
	token      string
	httpClient *http.Client
	wsConn     *websocket.Conn
	logger     *logrus.Logger
	config     *config.Config
	connected  bool
	msgID      int
}

// NewHAClientWrapper creates a new HomeAssistant client wrapper
func NewHAClientWrapper(config *config.Config, logger *logrus.Logger) *HAClientWrapper {
	return &HAClientWrapper{
		baseURL: getHABaseURL(config),
		token:   getHAToken(config),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
		config: config,
		msgID:  1,
	}
}

// Connect establishes connection to HomeAssistant
func (c *HAClientWrapper) Connect(ctx context.Context) error {
	// Test HTTP connection first
	if err := c.testConnection(ctx); err != nil {
		return fmt.Errorf("HTTP connection test failed: %w", err)
	}

	// Establish WebSocket connection
	if err := c.connectWebSocket(ctx); err != nil {
		return fmt.Errorf("WebSocket connection failed: %w", err)
	}

	c.connected = true
	c.logger.Info("Successfully connected to HomeAssistant")
	return nil
}

// Disconnect closes the connection to HomeAssistant
func (c *HAClientWrapper) Disconnect(ctx context.Context) error {
	if c.wsConn != nil {
		err := c.wsConn.Close()
		c.wsConn = nil
		if err != nil {
			return err
		}
	}
	c.connected = false
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

// Helper methods

func (c *HAClientWrapper) testConnection(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	return nil
}

func (c *HAClientWrapper) connectWebSocket(ctx context.Context) error {
	wsURL := strings.Replace(c.baseURL, "http", "ws", 1) + "/api/websocket"

	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return err
	}

	c.wsConn = conn

	// Handle auth flow
	return c.authenticateWebSocket()
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
