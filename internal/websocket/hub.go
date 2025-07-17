package websocket

import (
	"sync"
	"time"

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

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Statistics
	stats *HubStats
}

// HubStats contains hub statistics
type HubStats struct {
	ConnectedClients int       `json:"connected_clients"`
	TotalConnections int64     `json:"total_connections"`
	MessagesSent     int64     `json:"messages_sent"`
	MessagesReceived int64     `json:"messages_received"`
	LastActivity     time.Time `json:"last_activity"`
}

// NewHub creates a new WebSocket hub
func NewHub(logger *logrus.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
		stats: &HubStats{
			LastActivity: time.Now(),
		},
	}
}

// Run starts the hub and handles client registration/unregistration and broadcasting
func (h *Hub) Run() {
	h.logger.Info("WebSocket hub started")

	// Start periodic cleanup and statistics update
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case <-ticker.C:
			h.updateStats()
			h.sendHeartbeat()
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true
	h.stats.TotalConnections++
	h.stats.ConnectedClients = len(h.clients)
	h.stats.LastActivity = time.Now()

	h.logger.WithFields(logrus.Fields{
		"client_id":         client.ID,
		"remote_addr":       client.conn.RemoteAddr().String(),
		"connected_clients": len(h.clients),
	}).Info("WebSocket client connected")

	// Send welcome message
	welcome := Message{
		Type: "connection",
		Data: map[string]interface{}{
			"status":    "connected",
			"client_id": client.ID,
			"timestamp": time.Now().UTC(),
		},
	}
	client.send <- welcome.ToJSON()
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
		h.stats.ConnectedClients = len(h.clients)
		h.stats.LastActivity = time.Now()

		h.logger.WithFields(logrus.Fields{
			"client_id":         client.ID,
			"connected_clients": len(h.clients),
		}).Info("WebSocket client disconnected")
	}
}

func (h *Hub) broadcastMessage(message []byte) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	h.stats.MessagesSent++
	h.stats.LastActivity = time.Now()

	for _, client := range clients {
		select {
		case client.send <- message:
		default:
			// Client's send channel is full, close it
			h.unregister <- client
		}
	}

	h.logger.WithFields(logrus.Fields{
		"message_size": len(message),
		"clients_sent": len(clients),
	}).Debug("Message broadcasted to WebSocket clients")
}

func (h *Hub) updateStats() {
	h.mu.Lock()
	h.stats.ConnectedClients = len(h.clients)
	h.mu.Unlock()
}

func (h *Hub) sendHeartbeat() {
	heartbeat := Message{
		Type: "heartbeat",
		Data: map[string]interface{}{
			"timestamp": time.Now().UTC(),
			"clients":   h.stats.ConnectedClients,
		},
	}

	h.BroadcastToAll(heartbeat)
}

// BroadcastToAll broadcasts a message to all connected clients
func (h *Hub) BroadcastToAll(message Message) {
	data := message.ToJSON()
	select {
	case h.broadcast <- data:
	default:
		h.logger.Warn("Broadcast channel is full, message dropped")
	}
}

// BroadcastToRoom broadcasts a message to clients in a specific room
func (h *Hub) BroadcastToRoom(roomID int, message Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	data := message.ToJSON()

	for client := range h.clients {
		if client.IsInRoom(roomID) {
			select {
			case client.send <- data:
				count++
			default:
				h.unregister <- client
			}
		}
	}

	h.logger.WithFields(logrus.Fields{
		"room_id":      roomID,
		"clients_sent": count,
		"message_type": message.Type,
	}).Debug("Message broadcasted to room clients")
}

// GetStats returns current hub statistics
func (h *Hub) GetStats() *HubStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	statsCopy := *h.stats
	statsCopy.ConnectedClients = len(h.clients)
	return &statsCopy
}

// GetClientCount returns the current number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetClientByID returns a client by its ID, or nil if not found
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

// GetAllClients returns a copy of all connected clients
func (h *Hub) GetAllClients() []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}

	return clients
}
