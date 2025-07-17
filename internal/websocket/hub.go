package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/sirupsen/logrus"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Logger
	logger *logrus.Logger

	// Client subscriptions (client -> topics)
	subscriptions map[*Client]map[string]bool

	// Topic subscriptions (topic -> clients)
	topicClients map[string]map[*Client]bool

	// Message queue for offline clients
	messageQueue map[string][]Message

	// Metrics
	metrics *HubMetrics
	stats   *HubMetrics // Add stats field as alias to metrics

	// Mutex for thread safety
	mu sync.RWMutex

	// Shutdown channel
	shutdown chan bool
}

// ExtendedClientInfo holds additional information about a connected client
type ExtendedClientInfo struct {
	ID            string                 `json:"id"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	IPAddress     string                 `json:"ip_address"`
	ConnectedAt   time.Time              `json:"connected_at"`
	LastSeen      time.Time              `json:"last_seen"`
	Authenticated bool                   `json:"authenticated"`
	Subscriptions []string               `json:"subscriptions"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// HubMetrics holds metrics about the WebSocket hub
type HubMetrics struct {
	ConnectedClients    int            `json:"connected_clients"`
	TotalConnections    int            `json:"total_connections"`
	MessagesSent        int            `json:"messages_sent"`
	MessagesReceived    int            `json:"messages_received"`
	BytesSent           int64          `json:"bytes_sent"`
	BytesReceived       int64          `json:"bytes_received"`
	SubscriptionCount   int            `json:"subscription_count"`
	TopicCount          int            `json:"topic_count"`
	AverageResponseTime float64        `json:"average_response_time_ms"`
	ConnectionErrors    int            `json:"connection_errors"`
	LastMessageTime     time.Time      `json:"last_message_time"`
	ClientsByTopic      map[string]int `json:"clients_by_topic"`
	Uptime              time.Duration  `json:"uptime"`
	StartTime           time.Time      `json:"start_time"`
}

// Event types for broadcasting
const (
	EventTypeEntityUpdated    = "entity_updated"
	EventTypeEntityDeleted    = "entity_deleted"
	EventTypeRoomUpdated      = "room_updated"
	EventTypeRoomDeleted      = "room_deleted"
	EventTypeSystemStatus     = "system_status"
	EventTypeNotification     = "notification"
	EventTypeHeartbeat        = "heartbeat"
	EventTypeAuthentication   = "authentication"
	EventTypeError            = "error"
	EventTypeHAConnected      = "ha_connected"
	EventTypeHADisconnected   = "ha_disconnected"
	EventTypeHAStateChanged   = "ha_state_changed"
	EventTypeDeviceDiscovered = "device_discovered"
)

// Note: upgrader is declared in client.go to avoid redeclaration

// NewHub creates a new WebSocket hub
func NewHub(logger *logrus.Logger) *Hub {
	metrics := &HubMetrics{
		ClientsByTopic: make(map[string]int),
		StartTime:      time.Now(),
	}

	return &Hub{
		clients:       make(map[*Client]bool),
		broadcast:     make(chan []byte),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		logger:        logger,
		subscriptions: make(map[*Client]map[string]bool),
		topicClients:  make(map[string]map[*Client]bool),
		messageQueue:  make(map[string][]Message),
		metrics:       metrics,
		stats:         metrics, // Alias for backward compatibility
		shutdown:      make(chan bool),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	h.logger.Info("WebSocket hub starting...")

	// Start heartbeat ticker
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	// Start metrics ticker
	metricsTicker := time.NewTicker(5 * time.Minute)
	defer metricsTicker.Stop()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case <-heartbeatTicker.C:
			h.sendHeartbeat()

		case <-metricsTicker.C:
			h.updateMetrics()

		case <-h.shutdown:
			h.logger.Info("WebSocket hub shutting down...")
			return
		}
	}
}

// HandleWebSocket handles WebSocket upgrade requests
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		h.metrics.ConnectionErrors++
		return
	}

	clientID := generateClientID()

	// Create client
	client := &Client{
		conn:        conn,
		send:        make(chan []byte, 256),
		hub:         h,
		ID:          clientID,
		lastPing:    time.Now(),
		UserAgent:   r.UserAgent(),
		RemoteAddr:  getClientIP(r),
		ConnectedAt: time.Now(),
		info: &ClientInfo{
			ID:          clientID,
			IPAddress:   getClientIP(r),
			UserAgent:   r.UserAgent(),
			ConnectedAt: time.Now(),
			LastSeen:    time.Now(),
			Metadata:    make(map[string]interface{}),
		},
	}

	// Register the client
	h.register <- client

	// Allow collection of memory referenced by the caller
	go client.writePump()
	go client.readPump()

	h.logger.WithField("client_id", client.ID).Info("WebSocket client connected")
}

// BroadcastToAll sends a message to all connected clients
func (h *Hub) BroadcastToAll(messageType string, data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		// Convert to map if not already
		dataMap = map[string]interface{}{"data": data}
	}

	message := Message{
		Type:      messageType,
		Data:      dataMap,
		Timestamp: time.Now().UTC(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal broadcast message")
		return
	}

	h.broadcast <- messageBytes
}

// BroadcastToTopic sends a message to all clients subscribed to a topic
func (h *Hub) BroadcastToTopic(topic, messageType string, data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		// Convert to map if not already
		dataMap = map[string]interface{}{"data": data}
	}

	message := Message{
		Type:      messageType,
		Data:      dataMap,
		Timestamp: time.Now().UTC(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal topic message")
		return
	}

	h.mu.RLock()
	clients := h.topicClients[topic]
	h.mu.RUnlock()

	if clients == nil {
		return
	}

	for client := range clients {
		select {
		case client.send <- messageBytes:
			h.metrics.MessagesSent++
			h.metrics.BytesSent += int64(len(messageBytes))
		default:
			// Client's send channel is full, close it
			h.unregisterClient(client)
		}
	}
}

// SendToClient sends a message to a specific client
func (h *Hub) SendToClient(clientID, messageType string, data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		// Convert to map if not already
		dataMap = map[string]interface{}{"data": data}
	}

	message := Message{
		Type:      messageType,
		Data:      dataMap,
		Timestamp: time.Now().UTC(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal client message")
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.ID == clientID {
			select {
			case client.send <- messageBytes:
				h.metrics.MessagesSent++
				h.metrics.BytesSent += int64(len(messageBytes))
			default:
				h.unregisterClient(client)
			}
			break
		}
	}
}

// Convenience methods for common events

// BroadcastEntityUpdate broadcasts an entity update to all clients
func (h *Hub) BroadcastEntityUpdate(entity *models.Entity) {
	h.BroadcastToAll(EventTypeEntityUpdated, entity)
	h.BroadcastToTopic(fmt.Sprintf("entity:%s", entity.EntityID), EventTypeEntityUpdated, entity)
}

// BroadcastRoomUpdate broadcasts a room update to all clients
func (h *Hub) BroadcastRoomUpdate(room *models.Room) {
	h.BroadcastToAll(EventTypeRoomUpdated, room)
	h.BroadcastToTopic(fmt.Sprintf("room:%d", room.ID), EventTypeRoomUpdated, room)
}

// BroadcastSystemStatus broadcasts system status to all clients
func (h *Hub) BroadcastSystemStatus(status interface{}) {
	h.BroadcastToAll(EventTypeSystemStatus, status)
}

// BroadcastNotification broadcasts a notification to all clients
func (h *Hub) BroadcastNotification(notification interface{}) {
	h.BroadcastToAll(EventTypeNotification, notification)
}

// BroadcastHAStateChange broadcasts Home Assistant state changes
func (h *Hub) BroadcastHAStateChange(stateChange interface{}) {
	h.BroadcastToAll(EventTypeHAStateChanged, stateChange)
	h.BroadcastToTopic("homeassistant", EventTypeHAStateChanged, stateChange)
}

// GetMetrics returns current hub metrics
func (h *Hub) GetMetrics() *HubMetrics {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Update real-time metrics
	h.metrics.ConnectedClients = len(h.clients)
	h.metrics.SubscriptionCount = len(h.subscriptions)
	h.metrics.TopicCount = len(h.topicClients)
	h.metrics.Uptime = time.Since(h.metrics.StartTime)

	// Count clients by topic
	h.metrics.ClientsByTopic = make(map[string]int)
	for topic, clients := range h.topicClients {
		h.metrics.ClientsByTopic[topic] = len(clients)
	}

	return h.metrics
}

// GetStats returns current hub metrics (alias for GetMetrics)
func (h *Hub) GetStats() *HubMetrics {
	return h.GetMetrics()
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetClientByID returns a client by ID
func (h *Hub) GetClientByID(clientID string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.ID == clientID {
			return client
		}
	}
	return nil
}

// GetConnectedClients returns information about connected clients
func (h *Hub) GetConnectedClients() []*ClientInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*ClientInfo, 0, len(h.clients))
	for client := range h.clients {
		// Update subscriptions
		subs := make([]string, 0)
		if clientSubs, exists := h.subscriptions[client]; exists {
			for topic := range clientSubs {
				subs = append(subs, topic)
			}
		}
		client.info.Subscriptions = subs
		client.info.LastSeen = client.lastPing
		client.info.Authenticated = client.authenticated

		clients = append(clients, client.info)
	}

	return clients
}

// Shutdown gracefully shuts down the hub
func (h *Hub) Shutdown() {
	h.logger.Info("Shutting down WebSocket hub...")

	// Close all client connections
	h.mu.Lock()
	for client := range h.clients {
		client.conn.Close()
	}
	h.mu.Unlock()

	// Signal shutdown
	close(h.shutdown)
}

// Private methods

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true
	h.subscriptions[client] = make(map[string]bool)
	h.metrics.TotalConnections++

	h.logger.WithFields(logrus.Fields{
		"client_id":     client.ID,
		"ip_address":    client.info.IPAddress,
		"user_agent":    client.info.UserAgent,
		"total_clients": len(h.clients),
	}).Info("WebSocket client registered")

	// Send welcome message
	welcomeMsg := Message{
		Type: "welcome",
		Data: map[string]interface{}{
			"client_id":   client.ID,
			"server_time": time.Now().UTC().Format(time.RFC3339),
			"message":     "Connected to PMA WebSocket server",
		},
		Timestamp: time.Now().UTC(),
	}

	if msgBytes, err := json.Marshal(welcomeMsg); err == nil {
		select {
		case client.send <- msgBytes:
		default:
		}
	}
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)

		// Remove from subscriptions
		if subs, exists := h.subscriptions[client]; exists {
			for topic := range subs {
				if topicClients, exists := h.topicClients[topic]; exists {
					delete(topicClients, client)
					if len(topicClients) == 0 {
						delete(h.topicClients, topic)
					}
				}
			}
			delete(h.subscriptions, client)
		}

		h.logger.WithFields(logrus.Fields{
			"client_id":         client.ID,
			"remaining_clients": len(h.clients),
		}).Info("WebSocket client unregistered")
	}
}

func (h *Hub) broadcastMessage(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- message:
			h.metrics.MessagesSent++
			h.metrics.BytesSent += int64(len(message))
		default:
			// Client's send channel is full, close it
			go func(c *Client) {
				h.unregister <- c
			}(client)
		}
	}

	h.metrics.LastMessageTime = time.Now()
}

func (h *Hub) sendHeartbeat() {
	heartbeat := Message{
		Type: EventTypeHeartbeat,
		Data: map[string]interface{}{
			"server_time":       time.Now().UTC().Format(time.RFC3339),
			"connected_clients": len(h.clients),
			"uptime":            time.Since(h.metrics.StartTime).String(),
		},
		Timestamp: time.Now().UTC(),
	}

	if msgBytes, err := json.Marshal(heartbeat); err == nil {
		h.broadcast <- msgBytes
	}
}

func (h *Hub) updateMetrics() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.metrics.ConnectedClients = len(h.clients)
	h.metrics.SubscriptionCount = len(h.subscriptions)
	h.metrics.TopicCount = len(h.topicClients)

	h.logger.WithFields(logrus.Fields{
		"connected_clients": h.metrics.ConnectedClients,
		"messages_sent":     h.metrics.MessagesSent,
		"messages_received": h.metrics.MessagesReceived,
		"topics":            h.metrics.TopicCount,
	}).Debug("WebSocket metrics updated")
}

// Helper functions

func generateClientID() string {
	return fmt.Sprintf("client_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
}

func getClientIP(r *http.Request) string {
	// Try X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Try X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}
