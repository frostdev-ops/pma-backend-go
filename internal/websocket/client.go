package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now (configure based on your needs)
		return true
	},
}

// ClientInfo holds information about a connected client
type ClientInfo struct {
	ID            string                 `json:"id"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	IPAddress     string                 `json:"ip_address"`
	ConnectedAt   time.Time              `json:"connected_at"`
	LastSeen      time.Time              `json:"last_seen"`
	Authenticated bool                   `json:"authenticated"`
	Subscriptions []string               `json:"subscriptions"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Client is a middleman between the websocket connection and the hub
type Client struct {
	// Unique client identifier
	ID string

	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// Hub reference
	hub *Hub

	// Logger
	logger *logrus.Logger

	// Client metadata
	UserAgent   string    `json:"user_agent"`
	RemoteAddr  string    `json:"remote_addr"`
	ConnectedAt time.Time `json:"connected_at"`

	// Room subscriptions
	rooms map[int]bool

	// Home Assistant subscriptions
	haSubscriptions map[string]bool // event_type -> enabled
	roomFilters     map[string]bool // room_id -> subscribed
	entityFilters   map[string]bool // entity_id -> subscribed

	// Subscription mutex for thread-safe access
	subscriptionMu sync.RWMutex

	// Additional fields needed by hub
	lastPing      time.Time
	info          *ClientInfo
	authenticated bool
}

// HandleWebSocketWithAuth handles websocket requests from clients with authentication
func HandleWebSocketWithAuth(hub *Hub, w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	// Extract client IP address
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	hub.logger.WithFields(logrus.Fields{
		"client_ip":  clientIP,
		"path":       r.URL.Path,
		"user_agent": r.Header.Get("User-Agent"),
	}).Info("WebSocket connection attempt")

	// Check for localhost secret header first (for nginx-proxied local requests)
	localhostSecret := r.Header.Get("X-Localhost-Secret")
	if localhostSecret != "" && localhostSecret == cfg.Auth.LocalhostSecret {
		hub.logger.WithField("client_ip", clientIP).Info("WebSocket connection allowed (localhost secret bypass)")
		// Proceed with WebSocket upgrade for localhost secret bypass
		hub.HandleWebSocket(w, r, cfg)
		return
	}

	// Check if this is a local connection (localhost or local network)
	if isLocalConnection(clientIP) {
		hub.logger.WithField("client_ip", clientIP).Info("WebSocket connection allowed (local connection bypass)")
		// Proceed with WebSocket upgrade for local connections
		hub.HandleWebSocket(w, r, cfg)
		return
	}

	// For remote connections, require authentication
	// Check for API secret header first (preferred method)
	apiSecret := r.Header.Get("X-API-Secret")
	if apiSecret != "" {
		if apiSecret == cfg.Auth.APISecret {
			hub.logger.WithField("client_ip", clientIP).Info("WebSocket connection allowed (API secret)")
			hub.HandleWebSocket(w, r, cfg)
			return
		} else {
			hub.logger.WithField("client_ip", clientIP).Warn("WebSocket connection denied (invalid API secret)")
			http.Error(w, "Invalid API secret", http.StatusUnauthorized)
			return
		}
	}

	// Check for JWT token in Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		hub.logger.WithField("client_ip", clientIP).Warn("WebSocket connection denied (no authentication)")
		http.Error(w, "Authentication required for remote access", http.StatusUnauthorized)
		return
	}

	// Extract token from "Bearer <token>"
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		hub.logger.WithField("client_ip", clientIP).Warn("WebSocket connection denied (invalid authorization header)")
		http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
		return
	}

	tokenString := tokenParts[1]

	// Validate JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(cfg.Auth.JWTSecret), nil
	})

	if err != nil {
		hub.logger.WithField("client_ip", clientIP).WithError(err).Warn("WebSocket connection denied (invalid JWT token)")
		http.Error(w, "Invalid JWT token", http.StatusUnauthorized)
		return
	}

	if !token.Valid {
		hub.logger.WithField("client_ip", clientIP).Warn("WebSocket connection denied (invalid or expired JWT token)")
		http.Error(w, "Invalid or expired JWT token", http.StatusUnauthorized)
		return
	}

	// Extract claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		// Check if token is expired
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				hub.logger.WithField("client_ip", clientIP).Warn("WebSocket connection denied (expired JWT token)")
				http.Error(w, "JWT token has expired", http.StatusUnauthorized)
				return
			}
		}

		hub.logger.WithField("client_ip", clientIP).Info("WebSocket connection allowed (valid JWT token)")
		// Proceed with WebSocket upgrade for authenticated connections
		hub.HandleWebSocket(w, r, cfg)
	} else {
		hub.logger.WithField("client_ip", clientIP).Warn("WebSocket connection denied (invalid JWT claims)")
		http.Error(w, "Invalid JWT claims", http.StatusUnauthorized)
		return
	}
}

// HandleWebSocket handles websocket requests from clients
func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		hub.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	clientID := uuid.New().String()
	client := &Client{
		ID:              clientID,
		conn:            conn,
		send:            make(chan []byte, 256),
		hub:             hub,
		logger:          hub.logger,
		UserAgent:       r.Header.Get("User-Agent"),
		RemoteAddr:      r.RemoteAddr,
		ConnectedAt:     time.Now(),
		rooms:           make(map[int]bool),
		haSubscriptions: make(map[string]bool),
		roomFilters:     make(map[string]bool),
		entityFilters:   make(map[string]bool),
		info: &ClientInfo{
			ID:          clientID,
			IPAddress:   getClientIP(r),
			UserAgent:   r.Header.Get("User-Agent"),
			ConnectedAt: time.Now(),
			LastSeen:    time.Now(),
			Metadata:    make(map[string]interface{}),
		},
	}

	// Register the client with the hub
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines
	go client.writePump()
	go client.readPump()
}

// HandleWebSocketGin is a Gin-compatible wrapper for HandleWebSocket
func HandleWebSocketGin(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		HandleWebSocket(hub, c.Writer, c.Request)
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.WithError(err).Error("WebSocket connection error")
			}
			break
		}

		// Handle incoming message
		c.handleMessage(message)
		c.hub.metrics.MessagesReceived++
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				c.logger.WithError(err).Error("Failed to create writer")
				return
			}

			if _, err := w.Write(message); err != nil {
				c.logger.WithError(err).Error("Failed to write message")
				w.Close()
				return
			}

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				select {
				case queuedMessage := <-c.send:
					w.Write([]byte{'\n'})
					w.Write(queuedMessage)
				default:
					break
				}
			}

			if err := w.Close(); err != nil {
				c.logger.WithError(err).Error("Failed to close writer")
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.WithError(err).Error("Failed to write ping")
				return
			}
		}
	}
}

// handleMessage processes incoming messages from the client
func (c *Client) handleMessage(message []byte) {
	defer func() {
		if r := recover(); r != nil {
			c.logger.WithField("panic", r).Error("WebSocket message handler panicked")
		}
	}()

	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		c.logger.WithError(err).Error("Failed to unmarshal WebSocket message")
		return
	}

	switch msg.Type {
	case "subscribe":
		// Generic subscribe - subscribe to all entity state changes by default
		c.SubscribeToHAEvents([]string{"state_changed", "entity_updated"})
		// Send confirmation
		confirmMsg := Message{
			Type: "subscription_confirmed",
			Data: map[string]interface{}{
				"message": "Subscribed to entity state changes",
			},
		}
		c.send <- confirmMsg.ToJSON()
	case "subscribe_room":
		if roomID, ok := msg.Data["room_id"].(float64); ok {
			c.SubscribeToRoom(int(roomID))
		}
	case "unsubscribe_room":
		if roomID, ok := msg.Data["room_id"].(float64); ok {
			c.UnsubscribeFromRoom(int(roomID))
		}
	case "subscribe_ha_events":
		if eventTypes, ok := msg.Data["event_types"].([]interface{}); ok {
			var eventTypeStrs []string
			for _, et := range eventTypes {
				if etStr, ok := et.(string); ok {
					eventTypeStrs = append(eventTypeStrs, etStr)
				}
			}
			c.SubscribeToHAEvents(eventTypeStrs)
		}
	case "unsubscribe_ha_events":
		if eventTypes, ok := msg.Data["event_types"].([]interface{}); ok {
			var eventTypeStrs []string
			for _, et := range eventTypes {
				if etStr, ok := et.(string); ok {
					eventTypeStrs = append(eventTypeStrs, etStr)
				}
			}
			c.UnsubscribeFromHAEvents(eventTypeStrs)
		}
	case "subscribe_ha_entities":
		if entityIDs, ok := msg.Data["entity_ids"].([]interface{}); ok {
			var entityIDStrs []string
			for _, eid := range entityIDs {
				if eidStr, ok := eid.(string); ok {
					entityIDStrs = append(entityIDStrs, eidStr)
				}
			}
			c.SubscribeToHAEntities(entityIDStrs)
		}
	case "unsubscribe_ha_entities":
		if entityIDs, ok := msg.Data["entity_ids"].([]interface{}); ok {
			var entityIDStrs []string
			for _, eid := range entityIDs {
				if eidStr, ok := eid.(string); ok {
					entityIDStrs = append(entityIDStrs, eidStr)
				}
			}
			c.UnsubscribeFromHAEntities(entityIDStrs)
		}
	case "subscribe_ha_rooms":
		if roomIDs, ok := msg.Data["room_ids"].([]interface{}); ok {
			var roomIDStrs []string
			for _, rid := range roomIDs {
				if ridStr, ok := rid.(string); ok {
					roomIDStrs = append(roomIDStrs, ridStr)
				}
			}
			c.SubscribeToHARooms(roomIDStrs)
		}
	case "unsubscribe_ha_rooms":
		if roomIDs, ok := msg.Data["room_ids"].([]interface{}); ok {
			var roomIDStrs []string
			for _, rid := range roomIDs {
				if ridStr, ok := rid.(string); ok {
					roomIDStrs = append(roomIDStrs, ridStr)
				}
			}
			c.UnsubscribeFromHARooms(roomIDStrs)
		}
	case "ping":
		// Respond with pong
		pong := Message{
			Type: "pong",
			Data: map[string]interface{}{
				"timestamp": time.Now().UTC(),
			},
		}
		c.send <- pong.ToJSON()
	case "auth":
		// Handle authentication request
		c.handleAuthentication(msg)
	default:
		c.logger.WithField("message_type", msg.Type).Warn("Unknown WebSocket message type")
	}
}

// SubscribeToRoom subscribes the client to room updates
func (c *Client) SubscribeToRoom(roomID int) {
	// CRITICAL FIX: Add nil check to prevent "assignment to entry in nil map" panic
	if c.rooms == nil {
		c.rooms = make(map[int]bool)
		c.logger.WithField("client_id", c.ID).Warn("🔧 [WS-FIX] Recreated nil rooms map")
	}

	c.rooms[roomID] = true
	c.logger.WithFields(logrus.Fields{
		"client_id": c.ID,
		"room_id":   roomID,
	}).Info("Client subscribed to room")
}

// UnsubscribeFromRoom unsubscribes the client from room updates
func (c *Client) UnsubscribeFromRoom(roomID int) {
	delete(c.rooms, roomID)
	c.logger.WithFields(logrus.Fields{
		"client_id": c.ID,
		"room_id":   roomID,
	}).Info("Client unsubscribed from room")
}

// IsInRoom checks if the client is subscribed to a room
func (c *Client) IsInRoom(roomID int) bool {
	return c.rooms[roomID]
}

// SubscribeToHAEvents subscribes the client to specific HA event types
func (c *Client) SubscribeToHAEvents(eventTypes []string) error {
	c.subscriptionMu.Lock()
	defer c.subscriptionMu.Unlock()

	// CRITICAL FIX: Add nil check to prevent "assignment to entry in nil map" panic
	if c.haSubscriptions == nil {
		c.haSubscriptions = make(map[string]bool)
		c.logger.WithField("client_id", c.ID).Warn("🔧 [WS-FIX] Recreated nil haSubscriptions map")
	}

	for _, eventType := range eventTypes {
		c.haSubscriptions[eventType] = true
	}

	c.logger.WithFields(logrus.Fields{
		"client_id":   c.ID,
		"event_types": eventTypes,
	}).Info("Client subscribed to HA events")

	return nil
}

// UnsubscribeFromHAEvents unsubscribes the client from specific HA event types
func (c *Client) UnsubscribeFromHAEvents(eventTypes []string) error {
	c.subscriptionMu.Lock()
	defer c.subscriptionMu.Unlock()

	for _, eventType := range eventTypes {
		delete(c.haSubscriptions, eventType)
	}

	c.logger.WithFields(logrus.Fields{
		"client_id":   c.ID,
		"event_types": eventTypes,
	}).Info("Client unsubscribed from HA events")

	return nil
}

// SubscribeToHAEntities subscribes the client to specific HA entities
func (c *Client) SubscribeToHAEntities(entityIDs []string) error {
	c.subscriptionMu.Lock()
	defer c.subscriptionMu.Unlock()

	// CRITICAL FIX: Add nil check to prevent "assignment to entry in nil map" panic
	if c.entityFilters == nil {
		c.entityFilters = make(map[string]bool)
		c.logger.WithField("client_id", c.ID).Warn("🔧 [WS-FIX] Recreated nil entityFilters map")
	}

	for _, entityID := range entityIDs {
		c.entityFilters[entityID] = true
	}

	c.logger.WithFields(logrus.Fields{
		"client_id":  c.ID,
		"entity_ids": entityIDs,
	}).Info("Client subscribed to HA entities")

	return nil
}

// UnsubscribeFromHAEntities unsubscribes the client from specific HA entities
func (c *Client) UnsubscribeFromHAEntities(entityIDs []string) error {
	c.subscriptionMu.Lock()
	defer c.subscriptionMu.Unlock()

	for _, entityID := range entityIDs {
		delete(c.entityFilters, entityID)
	}

	c.logger.WithFields(logrus.Fields{
		"client_id":  c.ID,
		"entity_ids": entityIDs,
	}).Info("Client unsubscribed from HA entities")

	return nil
}

// SubscribeToHARooms subscribes the client to HA events for specific rooms
func (c *Client) SubscribeToHARooms(roomIDs []string) error {
	c.subscriptionMu.Lock()
	defer c.subscriptionMu.Unlock()

	// CRITICAL FIX: Add nil check to prevent "assignment to entry in nil map" panic
	if c.roomFilters == nil {
		c.roomFilters = make(map[string]bool)
		c.logger.WithField("client_id", c.ID).Warn("🔧 [WS-FIX] Recreated nil roomFilters map")
	}

	for _, roomID := range roomIDs {
		c.roomFilters[roomID] = true
	}

	c.logger.WithFields(logrus.Fields{
		"client_id": c.ID,
		"room_ids":  roomIDs,
	}).Info("Client subscribed to HA room filters")

	return nil
}

// UnsubscribeFromHARooms unsubscribes the client from HA events for specific rooms
func (c *Client) UnsubscribeFromHARooms(roomIDs []string) error {
	c.subscriptionMu.Lock()
	defer c.subscriptionMu.Unlock()

	for _, roomID := range roomIDs {
		delete(c.roomFilters, roomID)
	}

	c.logger.WithFields(logrus.Fields{
		"client_id": c.ID,
		"room_ids":  roomIDs,
	}).Info("Client unsubscribed from HA room filters")

	return nil
}

// GetHASubscriptions returns the client's current HA event subscriptions
func (c *Client) GetHASubscriptions() map[string]bool {
	c.subscriptionMu.RLock()
	defer c.subscriptionMu.RUnlock()

	// Create a copy to avoid race conditions
	subscriptions := make(map[string]bool)
	for eventType, enabled := range c.haSubscriptions {
		subscriptions[eventType] = enabled
	}

	return subscriptions
}

// GetHAEntityFilters returns the client's current HA entity filters
func (c *Client) GetHAEntityFilters() map[string]bool {
	c.subscriptionMu.RLock()
	defer c.subscriptionMu.RUnlock()

	// Create a copy to avoid race conditions
	filters := make(map[string]bool)
	for entityID, enabled := range c.entityFilters {
		filters[entityID] = enabled
	}

	return filters
}

// GetHARoomFilters returns the client's current HA room filters
func (c *Client) GetHARoomFilters() map[string]bool {
	c.subscriptionMu.RLock()
	defer c.subscriptionMu.RUnlock()

	// Create a copy to avoid race conditions
	filters := make(map[string]bool)
	for roomID, enabled := range c.roomFilters {
		filters[roomID] = enabled
	}

	return filters
}

// IsSubscribedToHAEvent checks if the client is subscribed to a specific HA event type
func (c *Client) IsSubscribedToHAEvent(eventType string) bool {
	c.subscriptionMu.RLock()
	defer c.subscriptionMu.RUnlock()

	return c.haSubscriptions[eventType]
}

// IsSubscribedToHAEntity checks if the client is subscribed to a specific HA entity
func (c *Client) IsSubscribedToHAEntity(entityID string) bool {
	c.subscriptionMu.RLock()
	defer c.subscriptionMu.RUnlock()

	// If no entity filters are set, consider all entities subscribed
	if len(c.entityFilters) == 0 {
		return true
	}

	return c.entityFilters[entityID]
}

// IsSubscribedToHARoom checks if the client is subscribed to HA events for a specific room
func (c *Client) IsSubscribedToHARoom(roomID string) bool {
	c.subscriptionMu.RLock()
	defer c.subscriptionMu.RUnlock()

	// If no room filters are set, consider all rooms subscribed
	if len(c.roomFilters) == 0 {
		return true
	}

	return c.roomFilters[roomID]
}

// handleAuthentication handles WebSocket authentication requests
func (c *Client) handleAuthentication(msg Message) {
	c.logger.WithFields(logrus.Fields{
		"client_id":      c.ID,
		"token_provided": msg.Token != "",
	}).Info("WebSocket authentication request received")

	var authSuccess bool
	var authMessage string
	var userInfo map[string]interface{}

	if msg.Token != "" {
		// Validate JWT token
		token, err := jwt.Parse(msg.Token, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			// Use the JWT secret from config - we need to get this from the hub or pass it in
			// For now, we'll use a default secret - in production this should be passed from the hub
			return []byte("Bwv3acVjr0RHkYNnXAsDAT7RYWXaQEZhm7xZzfccUMI="), nil
		})

		if err != nil {
			authSuccess = false
			authMessage = fmt.Sprintf("Invalid JWT token: %v", err)
			userInfo = map[string]interface{}{
				"authenticated": false,
				"error":         err.Error(),
			}
		} else if !token.Valid {
			authSuccess = false
			authMessage = "Invalid or expired JWT token"
			userInfo = map[string]interface{}{
				"authenticated": false,
				"error":         "token_invalid",
			}
		} else {
			// Extract claims
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				// Check if token is expired
				if exp, ok := claims["exp"].(float64); ok {
					if time.Now().Unix() > int64(exp) {
						authSuccess = false
						authMessage = "JWT token has expired"
						userInfo = map[string]interface{}{
							"authenticated": false,
							"error":         "token_expired",
						}
					} else {
						// Token is valid
						authSuccess = true
						authMessage = "Authentication successful"

						// Extract user info from claims
						userID := "1"
						username := "user"

						if userIDClaim, ok := claims["user_id"].(string); ok {
							userID = userIDClaim
						}
						if usernameClaim, ok := claims["username"].(string); ok {
							username = usernameClaim
						}

						userInfo = map[string]interface{}{
							"authenticated": true,
							"user_id":       userID,
							"username":      username,
							"permissions":   []string{"read", "write"},
							"auth_type":     claims["auth_type"],
						}

						// Mark client as authenticated
						c.authenticated = true
					}
				} else {
					authSuccess = false
					authMessage = "Invalid JWT claims"
					userInfo = map[string]interface{}{
						"authenticated": false,
						"error":         "invalid_claims",
					}
				}
			} else {
				authSuccess = false
				authMessage = "Invalid JWT claims"
				userInfo = map[string]interface{}{
					"authenticated": false,
					"error":         "invalid_claims",
				}
			}
		}
	} else {
		// No token provided
		authSuccess = false
		authMessage = "No authentication token provided"
		userInfo = map[string]interface{}{
			"authenticated": false,
			"error":         "no_token",
		}
	}

	authResponse := Message{
		Type: "auth_response",
		Data: map[string]interface{}{
			"success": authSuccess,
			"message": authMessage,
			"user":    userInfo,
		},
	}

	c.send <- authResponse.ToJSON()

	c.logger.WithFields(logrus.Fields{
		"client_id": c.ID,
		"success":   authSuccess,
		"message":   authMessage,
	}).Info("WebSocket authentication response sent")
}
