package handlers

import (
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/display"
	"github.com/gin-gonic/gin"
)

// GetDisplaySettings gets current display settings and capabilities
// @Summary Get display settings
// @Description Get current display settings and hardware capabilities
// @Tags display
// @Accept json
// @Produce json
// @Success 200 {object} gin.H{success=true,data=DisplaySettingsAndCapabilities}
// @Failure 500 {object} gin.H{success=false,error=string}
// @Router /api/v1/display-settings [get]
func (h *Handlers) GetDisplaySettings(c *gin.Context) {
	ctx := c.Request.Context()

	settings, err := h.displayService.GetSettings(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get display settings")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get display settings",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	capabilities, err := h.displayService.GetCapabilities(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get display capabilities")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get display capabilities",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	response := gin.H{
		"settings":     settings,
		"capabilities": capabilities,
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      response,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// UpdateDisplaySettings updates display settings
// @Summary Update display settings
// @Description Update display settings (partial updates supported)
// @Tags display
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param settings body display.DisplaySettingsRequest true "Display settings to update"
// @Success 200 {object} gin.H{success=true,data=display.DisplaySettings}
// @Failure 400 {object} gin.H{success=false,error=string}
// @Failure 500 {object} gin.H{success=false,error=string}
// @Router /api/v1/display-settings [post]
func (h *Handlers) UpdateDisplaySettings(c *gin.Context) {
	ctx := c.Request.Context()

	var request display.DisplaySettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.log.WithError(err).Error("Invalid display settings request")
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid display settings request",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Validate brightness range
	if request.Brightness != nil && (*request.Brightness < 10 || *request.Brightness > 100) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Brightness must be between 10 and 100",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Validate timeout range
	if request.Timeout != nil && *request.Timeout < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Timeout cannot be negative",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Validate dim level range
	if request.DimLevel != nil && (*request.DimLevel < 5 || *request.DimLevel > 95) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Dim level must be between 5 and 95",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	settings, err := h.displayService.UpdateSettings(ctx, &request)
	if err != nil {
		h.log.WithError(err).Error("Failed to update display settings")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to update display settings",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	h.log.WithField("settings", settings).Info("Display settings updated")

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      settings,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// PutDisplaySettings updates display settings (PUT endpoint for compatibility)
// @Summary Update display settings (PUT)
// @Description Update display settings using PUT method for compatibility
// @Tags display
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param settings body display.DisplaySettingsRequest true "Display settings to update"
// @Success 200 {object} gin.H{success=true,data=display.DisplaySettings}
// @Failure 400 {object} gin.H{success=false,error=string}
// @Failure 500 {object} gin.H{success=false,error=string}
// @Router /api/v1/display-settings [put]
func (h *Handlers) PutDisplaySettings(c *gin.Context) {
	// Delegate to POST handler for consistency
	h.UpdateDisplaySettings(c)
}

// GetDisplayCapabilities gets display hardware capabilities only
// @Summary Get display capabilities
// @Description Get hardware capabilities for display control
// @Tags display
// @Accept json
// @Produce json
// @Success 200 {object} gin.H{success=true,data=display.DisplayCapabilities}
// @Failure 500 {object} gin.H{success=false,error=string}
// @Router /api/v1/display-settings/capabilities [get]
func (h *Handlers) GetDisplayCapabilities(c *gin.Context) {
	ctx := c.Request.Context()

	capabilities, err := h.displayService.GetCapabilities(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get display capabilities")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get display capabilities",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      capabilities,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// WakeScreen wakes up the display
// @Summary Wake screen
// @Description Wake up the display and optionally keep it awake for a specified duration
// @Tags display
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body display.WakeScreenRequest false "Wake screen options"
// @Success 200 {object} gin.H{success=true}
// @Failure 500 {object} gin.H{success=false,error=string}
// @Router /api/v1/display-settings/wake [post]
func (h *Handlers) WakeScreen(c *gin.Context) {
	ctx := c.Request.Context()

	var request display.WakeScreenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		// If no body is provided, use default empty request
		request = display.WakeScreenRequest{}
	}

	if err := h.displayService.WakeScreen(ctx, &request); err != nil {
		h.log.WithError(err).Error("Failed to wake screen")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to wake screen",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	h.log.Info("Screen woken successfully")

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Screen woken successfully",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetDisplayHardwareInfo gets detailed hardware information
// @Summary Get display hardware info
// @Description Get detailed information about display hardware capabilities and status
// @Tags display
// @Accept json
// @Produce json
// @Success 200 {object} gin.H{success=true,data=display.HardwareInfo}
// @Failure 500 {object} gin.H{success=false,error=string}
// @Router /api/v1/display-settings/hardware [get]
func (h *Handlers) GetDisplayHardwareInfo(c *gin.Context) {
	ctx := c.Request.Context()

	info, err := h.displayService.GetHardwareInfo(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get display hardware info")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get display hardware info",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      info,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetDisplaySettingsLegacy provides legacy endpoint compatibility
// @Summary Get display settings (legacy)
// @Description Legacy endpoint for backward compatibility
// @Tags display
// @Accept json
// @Produce json
// @Success 200 {object} gin.H{success=true,data=display.DisplaySettings}
// @Failure 500 {object} gin.H{success=false,error=string}
// @Router /api/display-settings [get]
func (h *Handlers) GetDisplaySettingsLegacy(c *gin.Context) {
	ctx := c.Request.Context()

	settings, err := h.displayService.GetSettings(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get display settings")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get display settings",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	capabilities, err := h.displayService.GetCapabilities(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get display capabilities")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get display capabilities",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	response := gin.H{
		"settings":     settings,
		"capabilities": capabilities,
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      response,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// UpdateDisplaySettingsLegacy provides legacy endpoint compatibility
// @Summary Update display settings (legacy)
// @Description Legacy endpoint for backward compatibility
// @Tags display
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param settings body display.DisplaySettingsRequest true "Display settings to update"
// @Success 200 {object} gin.H{success=true,data=display.DisplaySettings}
// @Failure 400 {object} gin.H{success=false,error=string}
// @Failure 500 {object} gin.H{success=false,error=string}
// @Router /api/display-settings [post]
func (h *Handlers) UpdateDisplaySettingsLegacy(c *gin.Context) {
	// Delegate to main handler
	h.UpdateDisplaySettings(c)
}

// GetDisplayCapabilitiesLegacy provides legacy endpoint compatibility
// @Summary Get display capabilities (legacy)
// @Description Legacy endpoint for backward compatibility
// @Tags display
// @Accept json
// @Produce json
// @Success 200 {object} gin.H{success=true,data=display.DisplayCapabilities}
// @Failure 500 {object} gin.H{success=false,error=string}
// @Router /api/display-settings/capabilities [get]
func (h *Handlers) GetDisplayCapabilitiesLegacy(c *gin.Context) {
	// Delegate to main handler
	h.GetDisplayCapabilities(c)
}

// WakeScreenLegacy provides legacy endpoint compatibility
// @Summary Wake screen (legacy)
// @Description Legacy endpoint for backward compatibility
// @Tags display
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body display.WakeScreenRequest false "Wake screen options"
// @Success 200 {object} gin.H{success=true}
// @Failure 500 {object} gin.H{success=false,error=string}
// @Router /api/display-settings/wake [post]
func (h *Handlers) WakeScreenLegacy(c *gin.Context) {
	// Delegate to main handler
	h.WakeScreen(c)
}

// GetDisplayHardwareInfoLegacy provides legacy endpoint compatibility
// @Summary Get display hardware info (legacy)
// @Description Legacy endpoint for backward compatibility
// @Tags display
// @Accept json
// @Produce json
// @Success 200 {object} gin.H{success=true,data=display.HardwareInfo}
// @Failure 500 {object} gin.H{success=false,error=string}
// @Router /api/display-settings/hardware [get]
func (h *Handlers) GetDisplayHardwareInfoLegacy(c *gin.Context) {
	// Delegate to main handler
	h.GetDisplayHardwareInfo(c)
}
