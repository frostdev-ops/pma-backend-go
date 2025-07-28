package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/homeassistant"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// DebugHandlers contains debug-related handlers
type DebugHandlers struct {
	logger *logrus.Logger
	config *config.Config
}

// NewDebugHandlers creates new debug handlers
func NewDebugHandlers(logger *logrus.Logger, config *config.Config) *DebugHandlers {
	return &DebugHandlers{
		logger: logger,
		config: config,
	}
}

// TestHAConnection tests direct Home Assistant connection
func (h *DebugHandlers) TestHAConnection(c *gin.Context) {
	h.logger.Info("Testing Home Assistant connection...")

	// Create a new client wrapper for testing
	client := homeassistant.NewHAClientWrapper(h.config, h.logger)

	// Test with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	h.logger.Info("Attempting to connect to Home Assistant...")
	if err := client.Connect(ctx); err != nil {
		h.logger.WithError(err).Error("Home Assistant connection failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
			"step":    "connection",
		})
		return
	}

	h.logger.Info("Home Assistant connection successful, testing entity fetch...")

	// Test entity fetching
	entities, err := client.GetAllEntities(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch entities")
		client.Disconnect()
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
			"step":    "entity_fetch",
		})
		return
	}

	// Clean up
	client.Disconnect()

	h.logger.WithField("entity_count", len(entities)).Info("Home Assistant test completed successfully")

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"entity_count": len(entities),
		"message":      "Home Assistant connection and entity fetch successful",
		"first_entities": func() []string {
			var firstFive []string
			for i, entity := range entities {
				if i >= 5 {
					break
				}
				firstFive = append(firstFive, entity.EntityID)
			}
			return firstFive
		}(),
	})
}

// TestHAConnectionSimple tests just the HTTP API connection
func (h *DebugHandlers) TestHAConnectionSimple(c *gin.Context) {
	h.logger.Info("Testing simple Home Assistant HTTP connection...")

	// Create a new client wrapper for testing
	client := homeassistant.NewHAClientWrapper(h.config, h.logger)

	// Test with a short timeout - only HTTP, no WebSocket
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	h.logger.Info("Testing Home Assistant HTTP API...")

	// Test entity fetching directly without full connection
	entities, err := client.GetAllEntities(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch entities via HTTP")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
			"step":    "http_entity_fetch",
		})
		return
	}

	h.logger.WithField("entity_count", len(entities)).Info("Home Assistant HTTP test completed successfully")

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"entity_count": len(entities),
		"message":      "Home Assistant HTTP API test successful",
		"first_entities": func() []string {
			var firstFive []string
			for i, entity := range entities {
				if i >= 5 {
					break
				}
				firstFive = append(firstFive, entity.EntityID)
			}
			return firstFive
		}(),
	})
}
