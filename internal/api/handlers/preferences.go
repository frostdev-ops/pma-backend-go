package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/frostdev-ops/pma-backend-go/internal/core/dashboard"
	"github.com/frostdev-ops/pma-backend-go/internal/core/i18n"
	"github.com/frostdev-ops/pma-backend-go/internal/core/preferences"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PreferencesHandler handles preference-related requests
type PreferencesHandler struct {
	prefsManager     preferences.PreferencesManager
	themeManager     preferences.ThemeManager
	dashboardManager dashboard.DashboardManager
	localeManager    i18n.LocaleManager
	logger           *logrus.Logger
}

// NewPreferencesHandler creates a new preferences handler
func NewPreferencesHandler(
	prefsManager preferences.PreferencesManager,
	themeManager preferences.ThemeManager,
	dashboardManager dashboard.DashboardManager,
	localeManager i18n.LocaleManager,
	logger *logrus.Logger,
) *PreferencesHandler {
	return &PreferencesHandler{
		prefsManager:     prefsManager,
		themeManager:     themeManager,
		dashboardManager: dashboardManager,
		localeManager:    localeManager,
		logger:           logger,
	}
}

// GetUserPreferences retrieves current user preferences
func (h *PreferencesHandler) GetUserPreferences(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		// Default to user ID "1" for unauthenticated requests (localhost/API access)
		userID = "1"
	}

	prefs, err := h.prefsManager.GetUserPreferences(userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"preferences": prefs,
		"user_id":     userID,
	})
}

// UpdateUserPreferences updates user preferences
func (h *PreferencesHandler) UpdateUserPreferences(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		// Default to user ID "1" for unauthenticated requests (localhost/API access)
		userID = "1"
	}

	var prefs preferences.UserPreferences
	if err := c.ShouldBindJSON(&prefs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.prefsManager.UpdateUserPreferences(userID, &prefs); err != nil {
		h.logger.WithError(err).Error("Failed to update user preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Preferences updated successfully",
		"preferences": prefs,
	})
}

// GetPreferenceSection retrieves a specific preference section
func (h *PreferencesHandler) GetPreferenceSection(c *gin.Context) {
	userID := c.GetString("user_id")
	section := c.Param("section")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	prefs, err := h.prefsManager.GetUserPreferences(userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get preferences"})
		return
	}

	var sectionData interface{}
	switch section {
	case "theme":
		sectionData = prefs.Theme
	case "notifications":
		sectionData = prefs.Notifications
	case "dashboard":
		sectionData = prefs.Dashboard
	case "automation":
		sectionData = prefs.Automation
	case "locale":
		sectionData = prefs.Locale
	case "privacy":
		sectionData = prefs.Privacy
	case "accessibility":
		sectionData = prefs.Accessibility
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid preference section"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"section": section,
		"data":    sectionData,
	})
}

// UpdatePreferenceSection updates a specific preference section
func (h *PreferencesHandler) UpdatePreferenceSection(c *gin.Context) {
	userID := c.GetString("user_id")
	section := c.Param("section")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	prefs, err := h.prefsManager.GetUserPreferences(userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get preferences"})
		return
	}

	// Update the specific section
	dataBytes, _ := json.Marshal(updateData)
	switch section {
	case "theme":
		json.Unmarshal(dataBytes, &prefs.Theme)
	case "notifications":
		json.Unmarshal(dataBytes, &prefs.Notifications)
	case "dashboard":
		json.Unmarshal(dataBytes, &prefs.Dashboard)
	case "automation":
		json.Unmarshal(dataBytes, &prefs.Automation)
	case "locale":
		json.Unmarshal(dataBytes, &prefs.Locale)
	case "privacy":
		json.Unmarshal(dataBytes, &prefs.Privacy)
	case "accessibility":
		json.Unmarshal(dataBytes, &prefs.Accessibility)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid preference section"})
		return
	}

	if err := h.prefsManager.UpdateUserPreferences(userID, prefs); err != nil {
		h.logger.WithError(err).Error("Failed to update preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Section updated successfully",
		"section": section,
	})
}

// ResetToDefaults resets user preferences to default values
func (h *PreferencesHandler) ResetToDefaults(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	if err := h.prefsManager.ResetToDefaults(userID); err != nil {
		h.logger.WithError(err).Error("Failed to reset preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences reset to defaults"})
}

// ExportPreferences exports user preferences
func (h *PreferencesHandler) ExportPreferences(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	data, err := h.prefsManager.ExportPreferences(userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to export preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export preferences"})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=pma-preferences.json")
	c.Data(http.StatusOK, "application/json", data)
}

// ImportPreferences imports user preferences
func (h *PreferencesHandler) ImportPreferences(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var importData json.RawMessage
	if err := c.ShouldBindJSON(&importData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid import data"})
		return
	}

	if err := h.prefsManager.ImportPreferences(userID, importData); err != nil {
		h.logger.WithError(err).Error("Failed to import preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preferences imported successfully"})
}

// GetUserDashboard retrieves user dashboard configuration
func (h *PreferencesHandler) GetUserDashboard(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	dashboard, err := h.dashboardManager.GetUserDashboard(userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user dashboard")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get dashboard"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"dashboard": dashboard})
}

// SaveDashboard saves user dashboard configuration
func (h *PreferencesHandler) SaveDashboard(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var dashboard dashboard.Dashboard
	if err := c.ShouldBindJSON(&dashboard); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid dashboard data"})
		return
	}

	if err := h.dashboardManager.SaveDashboard(userID, &dashboard); err != nil {
		h.logger.WithError(err).Error("Failed to save dashboard")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save dashboard"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Dashboard saved successfully"})
}

// AddWidget adds a widget to user's dashboard
func (h *PreferencesHandler) AddWidget(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var widget dashboard.Widget
	if err := c.ShouldBindJSON(&widget); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid widget data"})
		return
	}

	if err := h.dashboardManager.AddWidget(userID, &widget); err != nil {
		h.logger.WithError(err).Error("Failed to add widget")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add widget"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Widget added successfully", "widget": widget})
}

// RemoveWidget removes a widget from user's dashboard
func (h *PreferencesHandler) RemoveWidget(c *gin.Context) {
	userID := c.GetString("user_id")
	widgetID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	if err := h.dashboardManager.RemoveWidget(userID, widgetID); err != nil {
		h.logger.WithError(err).Error("Failed to remove widget")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove widget"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Widget removed successfully"})
}

// UpdateWidget updates a widget configuration
func (h *PreferencesHandler) UpdateWidget(c *gin.Context) {
	userID := c.GetString("user_id")
	widgetID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid update data"})
		return
	}

	// Update position if provided
	if position, exists := updateData["position"]; exists {
		var pos dashboard.Position
		posBytes, _ := json.Marshal(position)
		if err := json.Unmarshal(posBytes, &pos); err == nil {
			if err := h.dashboardManager.UpdateWidgetPosition(userID, widgetID, pos); err != nil {
				h.logger.WithError(err).Error("Failed to update widget position")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update position"})
				return
			}
		}
	}

	// Update size if provided
	if size, exists := updateData["size"]; exists {
		var s dashboard.Size
		sizeBytes, _ := json.Marshal(size)
		if err := json.Unmarshal(sizeBytes, &s); err == nil {
			if err := h.dashboardManager.UpdateWidgetSize(userID, widgetID, s); err != nil {
				h.logger.WithError(err).Error("Failed to update widget size")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update size"})
				return
			}
		}
	}

	// Update config if provided
	if config, exists := updateData["config"]; exists {
		if configMap, ok := config.(map[string]interface{}); ok {
			if err := h.dashboardManager.UpdateWidgetConfig(userID, widgetID, configMap); err != nil {
				h.logger.WithError(err).Error("Failed to update widget config")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Widget updated successfully"})
}

// GetAvailableWidgets returns available widget types
func (h *PreferencesHandler) GetAvailableWidgets(c *gin.Context) {
	widgets := h.dashboardManager.GetAvailableWidgets()
	c.JSON(http.StatusOK, gin.H{"widgets": widgets})
}

// GetWidgetData retrieves data for a specific widget
func (h *PreferencesHandler) GetWidgetData(c *gin.Context) {
	userID := c.GetString("user_id")
	widgetID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	data, err := h.dashboardManager.GetWidgetData(userID, widgetID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get widget data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get widget data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// RefreshWidget forces a refresh of widget data
func (h *PreferencesHandler) RefreshWidget(c *gin.Context) {
	userID := c.GetString("user_id")
	widgetID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	if err := h.dashboardManager.RefreshWidget(userID, widgetID); err != nil {
		h.logger.WithError(err).Error("Failed to refresh widget")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh widget"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Widget refreshed successfully"})
}

// GetSupportedLocales returns supported locales
func (h *PreferencesHandler) GetSupportedLocales(c *gin.Context) {
	locales := h.localeManager.GetSupportedLocales()
	c.JSON(http.StatusOK, gin.H{"locales": locales})
}

// GetUserLocale retrieves user's current locale
func (h *PreferencesHandler) GetUserLocale(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	locale, err := h.localeManager.GetUserLocale(userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user locale")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get locale"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"locale": locale})
}

// SetUserLocale sets user's locale
func (h *PreferencesHandler) SetUserLocale(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request struct {
		Locale string `json:"locale" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.localeManager.SetUserLocale(userID, request.Locale); err != nil {
		h.logger.WithError(err).Error("Failed to set user locale")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set locale"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Locale updated successfully"})
}

// GetTranslations retrieves translations for a locale
func (h *PreferencesHandler) GetTranslations(c *gin.Context) {
	locale := c.Param("locale")
	if locale == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Locale is required"})
		return
	}

	translations, err := h.localeManager.GetTranslations(locale)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get translations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get translations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"locale":       locale,
		"translations": translations,
	})
}

// GetAvailableThemes returns available themes
func (h *PreferencesHandler) GetAvailableThemes(c *gin.Context) {
	themes := h.themeManager.GetAvailableThemes()
	c.JSON(http.StatusOK, gin.H{"themes": themes})
}

// GetTheme retrieves a specific theme
func (h *PreferencesHandler) GetTheme(c *gin.Context) {
	themeID := c.Param("id")
	if themeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Theme ID is required"})
		return
	}

	theme, err := h.themeManager.GetTheme(themeID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get theme")
		c.JSON(http.StatusNotFound, gin.H{"error": "Theme not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"theme": theme})
}

// ApplyTheme applies a theme to user's preferences
func (h *PreferencesHandler) ApplyTheme(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request struct {
		ThemeID string `json:"theme_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.themeManager.ApplyTheme(userID, request.ThemeID); err != nil {
		h.logger.WithError(err).Error("Failed to apply theme")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply theme"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Theme applied successfully"})
}

// CreateCustomTheme creates a new custom theme
func (h *PreferencesHandler) CreateCustomTheme(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var theme preferences.ThemeDefinition
	if err := c.ShouldBindJSON(&theme); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme data"})
		return
	}

	if err := h.themeManager.CreateCustomTheme(userID, theme); err != nil {
		h.logger.WithError(err).Error("Failed to create custom theme")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create theme"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Custom theme created successfully"})
}

// DeleteCustomTheme deletes a user's custom theme
func (h *PreferencesHandler) DeleteCustomTheme(c *gin.Context) {
	userID := c.GetString("user_id")
	themeID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	if err := h.themeManager.DeleteCustomTheme(userID, themeID); err != nil {
		h.logger.WithError(err).Error("Failed to delete custom theme")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete theme"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Custom theme deleted successfully"})
}

// GetPreferenceStatistics returns statistics about preferences usage
func (h *PreferencesHandler) GetPreferenceStatistics(c *gin.Context) {
	// This would typically require admin permissions
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// For now, return basic stats - in real implementation, check admin permissions
	stats := map[string]interface{}{
		"total_users":    1,
		"avg_size_bytes": 2048.0,
		"oldest_update":  "2024-01-01T00:00:00Z",
		"newest_update":  "2024-01-15T12:00:00Z",
	}

	c.JSON(http.StatusOK, gin.H{"statistics": stats})
}
