package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// GetConfig retrieves configuration values
func (h *Handlers) GetConfig(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		utils.SendError(c, http.StatusBadRequest, "Configuration key required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := h.repos.Config.Get(ctx, key)
	if err != nil {
		// Handle special cases for display_settings and theme_settings
		if key == "display_settings" {
			// Return default display settings structure
			defaultDisplaySettings := models.SystemConfig{
				Key:         "display_settings",
				Value:       `{"brightness":80,"timeout":300,"orientation":"landscape","darkMode":"auto","screensaver":true}`,
				Encrypted:   false,
				Description: "Default display settings configuration",
				UpdatedAt:   time.Now(),
			}
			utils.SendSuccess(c, defaultDisplaySettings)
			return
		}

		if key == "theme_settings" {
			// Return default theme settings structure
			defaultThemeSettings := models.SystemConfig{
				Key:         "theme_settings",
				Value:       `{"colorScheme":"auto","primaryColor":"#007bff","accentColor":"#28a745","fontSize":"medium","fontFamily":"system"}`,
				Encrypted:   false,
				Description: "Default theme settings configuration",
				UpdatedAt:   time.Now(),
			}
			utils.SendSuccess(c, defaultThemeSettings)
			return
		}

		h.log.WithError(err).Errorf("Failed to get config key: %s", key)
		utils.SendError(c, http.StatusNotFound, "Configuration not found")
		return
	}

	// Mask sensitive values
	if config.Encrypted {
		config.Value = "****"
	}

	utils.SendSuccess(c, config)
}

// SetConfig creates or updates configuration values
func (h *Handlers) SetConfig(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		utils.SendError(c, http.StatusBadRequest, "Configuration key required")
		return
	}

	var request struct {
		Value       string `json:"value" binding:"required"`
		Encrypted   bool   `json:"encrypted"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := &models.SystemConfig{
		Key:         key,
		Value:       request.Value,
		Encrypted:   request.Encrypted,
		Description: request.Description,
	}

	err := h.repos.Config.Set(ctx, config)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to set config key: %s", key)
		utils.SendError(c, http.StatusInternalServerError, "Failed to save configuration")
		return
	}

	utils.SendSuccess(c, gin.H{"message": "Configuration saved successfully"})
}

// GetAllConfig retrieves all configuration values
func (h *Handlers) GetAllConfig(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	configs, err := h.repos.Config.GetAll(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get all configs")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve configurations")
		return
	}

	// Mask sensitive values
	for _, config := range configs {
		if config.Encrypted {
			config.Value = "****"
		}
	}

	utils.SendSuccess(c, configs)
}
