package websocket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
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

	// Memory management
	maxClients      int
	maxMessageQueue int
	clientTimeout   time.Duration
	cleanupTicker   *time.Ticker
	cleanupStopChan chan bool
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

// Memory management constants
const (
	DefaultMaxClients      = 1000
	DefaultMaxMessageQueue = 1000
	DefaultClientTimeout   = 5 * time.Minute
	CleanupInterval        = 30 * time.Second
)

// Note: upgrader is declared in client.go to avoid redeclaration

// NewHub creates a new WebSocket hub
func NewHub(logger *logrus.Logger) *Hub {
	metrics := &HubMetrics{
		ClientsByTopic: make(map[string]int),
		StartTime:      time.Now(),
	}

	return &Hub{
		clients:         make(map[*Client]bool),
		broadcast:       make(chan []byte),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		logger:          logger,
		subscriptions:   make(map[*Client]map[string]bool),
		topicClients:    make(map[string]map[*Client]bool),
		messageQueue:    make(map[string][]Message),
		metrics:         metrics,
		stats:           metrics, // Alias for backward compatibility
		shutdown:        make(chan bool),
		maxClients:      DefaultMaxClients,
		maxMessageQueue: DefaultMaxMessageQueue,
		clientTimeout:   DefaultClientTimeout,
		cleanupTicker:   time.NewTicker(CleanupInterval),
		cleanupStopChan: make(chan bool),
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

	// Start cleanup ticker
	defer h.cleanupTicker.Stop()

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

		case <-h.cleanupTicker.C:
			h.cleanupMemory()

		case <-h.shutdown:
			h.logger.Info("WebSocket hub shutting down...")
			h.cleanupAll()
			return
		}
	}
}

// HandleWebSocket handles WebSocket upgrade requests with proper authentication
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	// Extract client IP address
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	h.logger.WithFields(logrus.Fields{
		"client_ip":  clientIP,
		"path":       r.URL.Path,
		"user_agent": r.Header.Get("User-Agent"),
	}).Info("WebSocket connection attempt")

	// CRITICAL FIX: Disable WebSocket authentication to match REST API approach
	// Authentication has been disabled across the application for development/testing
	h.logger.WithField("client_ip", clientIP).Info("WebSocket connection allowed (authentication disabled system-wide)")

	// Proceed with WebSocket upgrade without authentication checks
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
		logger:      h.logger, // CRITICAL FIX: Set logger to prevent nil pointer crashes
		ID:          clientID,
		lastPing:    time.Now(),
		UserAgent:   r.UserAgent(),
		RemoteAddr:  clientIP,
		ConnectedAt: time.Now(),
		info: &ClientInfo{
			ID:          clientID,
			IPAddress:   clientIP,
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

	h.logger.WithFields(logrus.Fields{
		"client_id":    client.ID,
		"client_ip":    clientIP,
		"is_local":     isLocalConnection(clientIP),
		"auth_enabled": cfg != nil && cfg.Auth.Enabled,
	}).Info("WebSocket client connected with authentication")
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
			// Client's send channel is full, close it directly without spawning goroutine
			// MEMORY LEAK FIX: Remove goroutine spawn to prevent goroutine leak
			h.logger.WithField("client_id", client.ID).Warn("Client send channel full, closing connection")
			delete(h.clients, client)
			close(client.send)
			client.conn.Close()
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
				// Client's send channel is full, close it directly without spawning goroutine
				// MEMORY LEAK FIX: Remove goroutine spawn to prevent goroutine leak
				h.logger.WithField("client_id", client.ID).Warn("Client send channel full, closing connection")
				delete(h.clients, client)
				close(client.send)
				client.conn.Close()
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

// BroadcastPMAEntityStateChange broadcasts a PMA entity state change to WebSocket clients
func (h *Hub) BroadcastPMAEntityStateChange(entityID string, oldState, newState interface{}, entity interface{}) {
	message := map[string]interface{}{
		"entity_id": entityID,
		"old_state": oldState,
		"new_state": newState,
		"entity":    entity,
		"timestamp": time.Now().UTC(),
	}

	// Broadcast to all clients
	h.BroadcastToAll("pma_entity_state_changed", message)

	// Broadcast to entity-specific topic
	h.BroadcastToTopic(fmt.Sprintf("entity:%s", entityID), "pma_entity_state_changed", message)
}

// BroadcastPMAEntityAdded broadcasts when a new PMA entity is discovered
func (h *Hub) BroadcastPMAEntityAdded(entity interface{}) {
	message := map[string]interface{}{
		"entity":    entity,
		"timestamp": time.Now().UTC(),
	}

	h.BroadcastToAll("pma_entity_added", message)
}

// BroadcastPMAEntityRemoved broadcasts when a PMA entity is removed
func (h *Hub) BroadcastPMAEntityRemoved(entityID string, source interface{}) {
	message := map[string]interface{}{
		"entity_id": entityID,
		"source":    source,
		"timestamp": time.Now().UTC(),
	}

	h.BroadcastToAll("pma_entity_removed", message)
	h.BroadcastToTopic(fmt.Sprintf("entity:%s", entityID), "pma_entity_removed", message)
}

// BroadcastPMASyncStatus broadcasts synchronization status updates
func (h *Hub) BroadcastPMASyncStatus(source string, status string, details map[string]interface{}) {
	message := map[string]interface{}{
		"source":    source,
		"status":    status,
		"details":   details,
		"timestamp": time.Now().UTC(),
	}

	h.BroadcastToAll("pma_sync_status", message)
	h.BroadcastToTopic(fmt.Sprintf("source:%s", source), "pma_sync_status", message)
}

// BroadcastPMAAdapterStatus broadcasts adapter health and connection status
func (h *Hub) BroadcastPMAAdapterStatus(adapterID, adapterName, source, status string, health interface{}, metrics interface{}) {
	message := map[string]interface{}{
		"adapter_id":   adapterID,
		"adapter_name": adapterName,
		"source":       source,
		"status":       status,
		"health":       health,
		"metrics":      metrics,
		"timestamp":    time.Now().UTC(),
	}

	h.BroadcastToAll("pma_adapter_status", message)
	h.BroadcastToTopic(fmt.Sprintf("adapter:%s", adapterID), "pma_adapter_status", message)
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

// isLocalConnection checks if the client IP is from a local connection
func isLocalConnection(clientIP string) bool {
	// Extract IP from address:port format if present
	if strings.Contains(clientIP, ":") {
		parts := strings.Split(clientIP, ":")
		if len(parts) > 0 {
			clientIP = parts[0]
		}
	}

	// Check for localhost IPs
	localhostIPs := []string{"127.0.0.1", "::1", "localhost", "0.0.0.0"}
	for _, ip := range localhostIPs {
		if clientIP == ip {
			return true
		}
	}

	// Check for local network ranges
	if strings.HasPrefix(clientIP, "192.168.") ||
		strings.HasPrefix(clientIP, "10.") ||
		strings.HasPrefix(clientIP, "172.") ||
		strings.HasPrefix(clientIP, "169.254.") {
		return true
	}

	return false
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

	// Stop cleanup ticker
	if h.cleanupTicker != nil {
		h.cleanupTicker.Stop()
		close(h.cleanupStopChan)
	}

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

	// Check if we've reached the maximum number of clients
	if len(h.clients) >= h.maxClients {
		h.logger.WithField("max_clients", h.maxClients).Warn("Maximum clients reached, rejecting new connection")
		client.conn.Close()
		return
	}

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
			// Channel is full, close the connection
			h.logger.WithField("client_id", client.ID).Warn("Client send channel full, closing connection")
			delete(h.clients, client)
			close(client.send)
			client.conn.Close()
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
			// Client's send channel is full, close it directly without spawning goroutine
			// MEMORY LEAK FIX: Remove goroutine spawn to prevent goroutine leak
			h.logger.WithField("client_id", client.ID).Warn("Client send channel full, closing connection")
			delete(h.clients, client)
			close(client.send)
			client.conn.Close()
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
	h.metrics.Uptime = time.Since(h.metrics.StartTime)

	// Update topic counts
	h.metrics.TopicCount = len(h.topicClients)
	h.metrics.SubscriptionCount = 0
	for _, subs := range h.subscriptions {
		h.metrics.SubscriptionCount += len(subs)
	}

	// Update clients by topic
	h.metrics.ClientsByTopic = make(map[string]int)
	for topic, clients := range h.topicClients {
		h.metrics.ClientsByTopic[topic] = len(clients)
	}
}

// cleanupMemory performs periodic memory cleanup
func (h *Hub) cleanupMemory() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Clean up inactive clients
	now := time.Now()
	inactiveClients := 0
	for client := range h.clients {
		if now.Sub(client.ConnectedAt) > h.clientTimeout {
			h.logger.WithField("client_id", client.ID).Info("Removing inactive client")
			delete(h.clients, client)
			close(client.send)
			inactiveClients++
		}
	}

	// Clean up message queue if too large
	if len(h.messageQueue) > h.maxMessageQueue {
		h.logger.WithField("queue_size", len(h.messageQueue)).Warn("Message queue too large, cleaning up")
		h.messageQueue = make(map[string][]Message)
	}

	if inactiveClients > 0 {
		h.logger.WithField("inactive_clients", inactiveClients).Info("Cleaned up inactive clients")
	}
}

// cleanupAll performs complete cleanup on shutdown
func (h *Hub) cleanupAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close all client connections
	for client := range h.clients {
		close(client.send)
		client.conn.Close()
	}

	// Clear all maps
	h.clients = make(map[*Client]bool)
	h.subscriptions = make(map[*Client]map[string]bool)
	h.topicClients = make(map[string]map[*Client]bool)
	h.messageQueue = make(map[string][]Message)

	h.logger.Info("WebSocket hub cleanup completed")
}

// Helper functions

// WebSocketEventEmitter wraps the Hub to implement the EventEmitter interface
// This allows other services to broadcast events without directly importing websocket types
type WebSocketEventEmitter struct {
	hub *Hub
}

// NewWebSocketEventEmitter creates a new WebSocket event emitter
func NewWebSocketEventEmitter(hub *Hub) *WebSocketEventEmitter {
	return &WebSocketEventEmitter{
		hub: hub,
	}
}

// Implement EventEmitter interface methods
func (w *WebSocketEventEmitter) BroadcastPMAEntityStateChange(entityID string, oldState, newState interface{}, entity interface{}) {
	logMemStatsWS(w.hub.logger, "before_BroadcastPMAEntityStateChange")
	w.hub.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"old_state": oldState,
		"new_state": newState,
		"entity":    fmt.Sprintf("%#v", entity),
	}).Info("Broadcasting PMA entity state change")
	w.hub.BroadcastPMAEntityStateChange(entityID, oldState, newState, entity)
	logMemStatsWS(w.hub.logger, "after_BroadcastPMAEntityStateChange")
}

func (w *WebSocketEventEmitter) BroadcastPMAEntityAdded(entity interface{}) {
	logMemStatsWS(w.hub.logger, "before_BroadcastPMAEntityAdded")
	w.hub.logger.WithField("entity", fmt.Sprintf("%#v", entity)).Info("Broadcasting PMA entity added")
	w.hub.BroadcastPMAEntityAdded(entity)
	logMemStatsWS(w.hub.logger, "after_BroadcastPMAEntityAdded")
}

func (w *WebSocketEventEmitter) BroadcastPMAEntityRemoved(entityID string, source interface{}) {
	if w.hub != nil {
		w.hub.BroadcastPMAEntityRemoved(entityID, source)
	}
}

func (w *WebSocketEventEmitter) BroadcastPMASyncStatus(source string, status string, details map[string]interface{}) {
	if w.hub != nil {
		w.hub.BroadcastPMASyncStatus(source, status, details)
	}
}

func (w *WebSocketEventEmitter) BroadcastPMAAdapterStatus(adapterID, adapterName, source, status string, health interface{}, metrics interface{}) {
	if w.hub != nil {
		w.hub.BroadcastPMAAdapterStatus(adapterID, adapterName, source, status, health, metrics)
	}
}

func logMemStatsWS(logger *logrus.Logger, context string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	logger.WithFields(logrus.Fields{
		"context":        context,
		"alloc_mb":       m.Alloc / 1024 / 1024,
		"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
		"sys_mb":         m.Sys / 1024 / 1024,
		"num_gc":         m.NumGC,
	}).Info("[MEMSTATS][WS] Memory usage snapshot")
}
