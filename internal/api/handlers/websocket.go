package handlers

import (
	"net/http"

	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// HASubscriptionRequest represents a request to subscribe/unsubscribe from HA events
type HASubscriptionRequest struct {
	EventTypes []string `json:"event_types,omitempty"`
	RoomIDs    []string `json:"room_ids,omitempty"`
	EntityIDs  []string `json:"entity_ids,omitempty"`
}

// HASubscriptionResponse represents the response to subscription requests
type HASubscriptionResponse struct {
	Success       bool            `json:"success"`
	Subscriptions map[string]bool `json:"subscriptions"`
	Message       string          `json:"message,omitempty"`
}

// WebSocketHandler handles WebSocket connections with authentication
func (h *Handlers) WebSocketHandler(hub *websocket.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		websocket.HandleWebSocketWithAuth(hub, c.Writer, c.Request, h.cfg)
	}
}

// GetWebSocketStats returns WebSocket statistics
func (h *Handlers) GetWebSocketStats(hub *websocket.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats := hub.GetStats()
		utils.SendSuccess(c, stats)
	}
}

// BroadcastMessage broadcasts a message to all WebSocket clients
func (h *Handlers) BroadcastMessage(hub *websocket.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request struct {
			Type string                 `json:"type" binding:"required"`
			Data map[string]interface{} `json:"data" binding:"required"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			utils.SendError(c, http.StatusBadRequest, "Invalid request body")
			return
		}

		message := websocket.Message{
			Type: request.Type,
			Data: request.Data,
		}

		hub.BroadcastToAll(message.Type, message.Data)

		utils.SendSuccess(c, gin.H{
			"message":       "Message broadcasted successfully",
			"clients_count": hub.GetClientCount(),
			"message_type":  request.Type,
		})
	}
}

// SubscribeToHAEvents subscribes a client to Home Assistant events via HTTP API
func (h *Handlers) SubscribeToHAEvents(hub *websocket.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.Query("client_id")
		if clientID == "" {
			utils.SendError(c, http.StatusBadRequest, "client_id query parameter is required")
			return
		}

		var request HASubscriptionRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			utils.SendError(c, http.StatusBadRequest, "Invalid request body")
			return
		}

		client := hub.GetClientByID(clientID)
		if client == nil {
			utils.SendError(c, http.StatusNotFound, "Client not found")
			return
		}

		// Subscribe to event types
		if len(request.EventTypes) > 0 {
			if err := client.SubscribeToHAEvents(request.EventTypes); err != nil {
				utils.SendError(c, http.StatusInternalServerError, "Failed to subscribe to HA events")
				return
			}
		}

		// Subscribe to rooms if specified
		if len(request.RoomIDs) > 0 {
			if err := client.SubscribeToHARooms(request.RoomIDs); err != nil {
				utils.SendError(c, http.StatusInternalServerError, "Failed to subscribe to HA rooms")
				return
			}
		}

		// Subscribe to entities if specified
		if len(request.EntityIDs) > 0 {
			if err := client.SubscribeToHAEntities(request.EntityIDs); err != nil {
				utils.SendError(c, http.StatusInternalServerError, "Failed to subscribe to HA entities")
				return
			}
		}

		response := HASubscriptionResponse{
			Success:       true,
			Subscriptions: client.GetHASubscriptions(),
			Message:       "Successfully subscribed to HA events",
		}

		utils.SendSuccess(c, response)
	}
}

// UnsubscribeFromHAEvents unsubscribes a client from Home Assistant events via HTTP API
func (h *Handlers) UnsubscribeFromHAEvents(hub *websocket.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.Query("client_id")
		if clientID == "" {
			utils.SendError(c, http.StatusBadRequest, "client_id query parameter is required")
			return
		}

		var request HASubscriptionRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			utils.SendError(c, http.StatusBadRequest, "Invalid request body")
			return
		}

		client := hub.GetClientByID(clientID)
		if client == nil {
			utils.SendError(c, http.StatusNotFound, "Client not found")
			return
		}

		// Unsubscribe from event types
		if len(request.EventTypes) > 0 {
			if err := client.UnsubscribeFromHAEvents(request.EventTypes); err != nil {
				utils.SendError(c, http.StatusInternalServerError, "Failed to unsubscribe from HA events")
				return
			}
		}

		// Unsubscribe from rooms if specified
		if len(request.RoomIDs) > 0 {
			if err := client.UnsubscribeFromHARooms(request.RoomIDs); err != nil {
				utils.SendError(c, http.StatusInternalServerError, "Failed to unsubscribe from HA rooms")
				return
			}
		}

		// Unsubscribe from entities if specified
		if len(request.EntityIDs) > 0 {
			if err := client.UnsubscribeFromHAEntities(request.EntityIDs); err != nil {
				utils.SendError(c, http.StatusInternalServerError, "Failed to unsubscribe from HA entities")
				return
			}
		}

		response := HASubscriptionResponse{
			Success:       true,
			Subscriptions: client.GetHASubscriptions(),
			Message:       "Successfully unsubscribed from HA events",
		}

		utils.SendSuccess(c, response)
	}
}

// GetHASubscriptions returns a client's current HA subscriptions
func (h *Handlers) GetHASubscriptions(hub *websocket.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.Query("client_id")
		if clientID == "" {
			utils.SendError(c, http.StatusBadRequest, "client_id query parameter is required")
			return
		}

		client := hub.GetClientByID(clientID)
		if client == nil {
			utils.SendError(c, http.StatusNotFound, "Client not found")
			return
		}

		subscriptions := map[string]interface{}{
			"event_subscriptions": client.GetHASubscriptions(),
			"entity_filters":      client.GetHAEntityFilters(),
			"room_filters":        client.GetHARoomFilters(),
		}

		utils.SendSuccess(c, subscriptions)
	}
}

// Legacy HA event forwarding methods removed - functionality moved to unified service
