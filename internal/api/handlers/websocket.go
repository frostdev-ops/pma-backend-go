package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/frostdev-ops/pma-backend-go/internal/websocket"
    "github.com/frostdev-ops/pma-backend-go/pkg/utils"
)

// WebSocketHandler handles WebSocket connections
func (h *Handlers) WebSocketHandler(hub *websocket.Hub) gin.HandlerFunc {
    return websocket.HandleWebSocketGin(hub)
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

        hub.BroadcastToAll(message)

        utils.SendSuccess(c, gin.H{
            "message":        "Message broadcasted successfully",
            "clients_count":  hub.GetClientCount(),
            "message_type":   request.Type,
        })
    }
} 