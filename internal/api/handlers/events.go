package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SSEMessage represents a Server-Sent Event message
type SSEMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
}

// SSEConnection represents an active SSE connection
type SSEConnection struct {
	ID          string
	Channel     chan SSEMessage
	ClientIP    string
	ConnectedAt time.Time
	LastPing    time.Time
}

// EventsHandler handles SSE event streaming
type EventsHandler struct {
	connections     map[string]*SSEConnection
	connectionsMux  sync.RWMutex
	heartbeatTicker *time.Ticker
	log             *log.Logger
}

// NewEventsHandler creates a new events handler
func NewEventsHandler(logger *log.Logger) *EventsHandler {
	handler := &EventsHandler{
		connections: make(map[string]*SSEConnection),
		log:         logger,
	}

	// Start heartbeat ticker
	handler.startHeartbeat()

	return handler
}

// GetEventStream handles SSE connection endpoint
func (h *EventsHandler) GetEventStream(c *gin.Context) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	// CORS headers are handled by middleware, not here
	c.Header("X-Accel-Buffering", "no") // Disable Nginx buffering

	// Generate connection ID
	connID := fmt.Sprintf("conn_%d_%s", time.Now().UnixNano(), c.ClientIP())

	// Create connection
	conn := &SSEConnection{
		ID:          connID,
		Channel:     make(chan SSEMessage, 100),
		ClientIP:    c.ClientIP(),
		ConnectedAt: time.Now(),
		LastPing:    time.Now(),
	}

	// Add to connections map
	h.connectionsMux.Lock()
	h.connections[connID] = conn
	h.connectionsMux.Unlock()

	h.log.Printf("SSE connection established: %s from %s (total: %d)",
		connID, c.ClientIP(), len(h.connections))

	// Send initial connection message
	initialMsg := SSEMessage{
		Type: "heartbeat",
		Data: map[string]interface{}{
			"message":      "Connected to PMA real-time updates",
			"connectionId": connID,
			"serverTime":   time.Now().Format(time.RFC3339),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Send initial message immediately
	h.writeSSEMessage(c.Writer, initialMsg)
	c.Writer.Flush()

	// Send initial system status
	go func() {
		time.Sleep(100 * time.Millisecond) // Small delay to ensure connection is stable
		statusMsg := SSEMessage{
			Type: "system_status",
			Data: map[string]interface{}{
				"status":    "online",
				"timestamp": time.Now().Format(time.RFC3339),
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		select {
		case conn.Channel <- statusMsg:
		default:
		}
	}()

	// Handle client disconnect
	ctx := c.Request.Context()
	defer func() {
		h.connectionsMux.Lock()
		delete(h.connections, connID)
		h.connectionsMux.Unlock()
		close(conn.Channel)
		h.log.Printf("SSE connection closed: %s (remaining: %d)", connID, len(h.connections))
	}()

	// Message loop
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-conn.Channel:
			if err := h.writeSSEMessage(c.Writer, msg); err != nil {
				h.log.Printf("Error writing SSE message to %s: %v", connID, err)
				return
			}
			c.Writer.Flush()
			conn.LastPing = time.Now()
		case <-time.After(30 * time.Second):
			// Send keepalive if no messages for 30 seconds
			if _, err := c.Writer.Write([]byte(":keepalive\n\n")); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}

// GetEventStatus returns the status of the events service
func (h *EventsHandler) GetEventStatus(c *gin.Context) {
	h.connectionsMux.RLock()
	connectionCount := len(h.connections)
	h.connectionsMux.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"status":            "Events service available",
			"activeConnections": connectionCount,
			"uptime":            time.Since(time.Now()).String(),
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// writeSSEMessage writes an SSE message to the response writer
func (h *EventsHandler) writeSSEMessage(w http.ResponseWriter, msg SSEMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}

// BroadcastMessage sends a message to all connected clients
func (h *EventsHandler) BroadcastMessage(msgType string, data interface{}) {
	msg := SSEMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	h.connectionsMux.RLock()
	defer h.connectionsMux.RUnlock()

	sentCount := 0
	for connID, conn := range h.connections {
		select {
		case conn.Channel <- msg:
			sentCount++
		default:
			// Channel is full or closed, skip this connection
			h.log.Printf("Skipping message to connection %s (channel full or closed)", connID)
		}
	}

	if sentCount > 0 {
		h.log.Printf("Broadcast message type '%s' sent to %d/%d connections",
			msgType, sentCount, len(h.connections))
	}
}

// BroadcastEntityUpdate sends entity update to all clients
func (h *EventsHandler) BroadcastEntityUpdate(entityID string, entity interface{}) {
	h.BroadcastMessage("entity_updated", map[string]interface{}{
		"entityId": entityID,
		"entity":   entity,
	})
}

// BroadcastConfigUpdate sends config update to all clients
func (h *EventsHandler) BroadcastConfigUpdate(configType string, config interface{}) {
	h.BroadcastMessage("config_update", map[string]interface{}{
		"configType": configType,
		"config":     config,
	})
}

// BroadcastDeviceUpdate sends device update to all clients
func (h *EventsHandler) BroadcastDeviceUpdate(deviceID string, device interface{}) {
	h.BroadcastMessage("device_update", map[string]interface{}{
		"deviceId": deviceID,
		"device":   device,
	})
}

// BroadcastSystemStatus sends system status update to all clients
func (h *EventsHandler) BroadcastSystemStatus(status interface{}) {
	h.BroadcastMessage("system_status", status)
}

// BroadcastError sends error message to all clients
func (h *EventsHandler) BroadcastError(errorMsg string, details interface{}) {
	h.BroadcastMessage("error", map[string]interface{}{
		"message": errorMsg,
		"details": details,
	})
}

// startHeartbeat starts the heartbeat ticker
func (h *EventsHandler) startHeartbeat() {
	h.heartbeatTicker = time.NewTicker(30 * time.Second)

	go func() {
		for range h.heartbeatTicker.C {
			h.BroadcastMessage("heartbeat", map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
				"uptime":    time.Since(time.Now()).String(),
			})
		}
	}()
}

// Shutdown gracefully shuts down the events handler
func (h *EventsHandler) Shutdown() {
	if h.heartbeatTicker != nil {
		h.heartbeatTicker.Stop()
	}

	// Close all connections
	h.connectionsMux.Lock()
	for connID, conn := range h.connections {
		close(conn.Channel)
		delete(h.connections, connID)
	}
	h.connectionsMux.Unlock()

	h.log.Println("Events handler shutdown complete")
}

// GetActiveConnections returns the number of active connections
func (h *EventsHandler) GetActiveConnections() int {
	h.connectionsMux.RLock()
	defer h.connectionsMux.RUnlock()
	return len(h.connections)
}

// GetConnectionInfo returns information about active connections
func (h *EventsHandler) GetConnectionInfo() []map[string]interface{} {
	h.connectionsMux.RLock()
	defer h.connectionsMux.RUnlock()

	info := make([]map[string]interface{}, 0, len(h.connections))
	for _, conn := range h.connections {
		info = append(info, map[string]interface{}{
			"id":          conn.ID,
			"clientIP":    conn.ClientIP,
			"connectedAt": conn.ConnectedAt.Format(time.RFC3339),
			"lastPing":    conn.LastPing.Format(time.RFC3339),
			"duration":    time.Since(conn.ConnectedAt).String(),
		})
	}

	return info
}
