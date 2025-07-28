package handlers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/controller"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Helper method to convert string userID to *int for controller service
func (h *Handlers) getControllerUserID(c *gin.Context) *int {
	// If auth is disabled, return nil to skip permission checks
	if authDisabled, exists := c.Get("auth_disabled"); exists && authDisabled.(bool) {
		return nil
	}

	userIDStr := h.getUserIDFromContext(c)
	if userIDStr == "" {
		return nil
	}

	if userID, err := strconv.Atoi(userIDStr); err == nil {
		return &userID
	}

	return nil
}

// Controller Dashboard CRUD Operations

// GetControllerDashboards retrieves all controller dashboards accessible to the user
func (h *Handlers) GetControllerDashboards(c *gin.Context) {
	// Check if controllerService is available
	if h.controllerService == nil {
		utils.SendError(c, http.StatusInternalServerError, "Controller service not available")
		return
	}

	userID := h.getControllerUserID(c)

	// Parse query parameters
	search := c.Query("search")
	category := c.Query("category")
	tagsParam := c.Query("tags")
	favorites := c.Query("favorites") == "true"
	shared := c.Query("shared") == "true"

	var tags []string
	if tagsParam != "" {
		tags = strings.Split(tagsParam, ",")
	}

	filters := controller.DashboardFilters{
		Search:    search,
		Category:  category,
		Tags:      tags,
		Favorites: favorites,
		Shared:    shared,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dashboards, err := h.controllerService.GetDashboards(ctx, userID, filters)
	if err != nil {
		h.log.WithError(err).WithField("user_id", userID).Error("Failed to get controller dashboards")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve dashboards")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      dashboards,
		"count":     len(dashboards),
		"timestamp": time.Now().UTC(),
	})
}

// GetControllerDashboard retrieves a specific controller dashboard by ID
func (h *Handlers) GetControllerDashboard(c *gin.Context) {
	dashboardID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid dashboard ID")
		return
	}

	userID := h.getControllerUserID(c)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dashboard, err := h.controllerService.GetDashboard(ctx, dashboardID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "access denied") || strings.Contains(err.Error(), "not found") {
			utils.SendError(c, http.StatusNotFound, "Dashboard not found or access denied")
			return
		}
		h.log.WithError(err).WithFields(logrus.Fields{
			"dashboard_id": dashboardID,
			"user_id":      userID,
		}).Error("Failed to get controller dashboard")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve dashboard")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      dashboard,
		"timestamp": time.Now().UTC(),
	})
}

// CreateControllerDashboard creates a new controller dashboard
func (h *Handlers) CreateControllerDashboard(c *gin.Context) {
	userID := h.getControllerUserID(c)

	var dashboard models.ControllerDashboard
	if err := c.ShouldBindJSON(&dashboard); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid dashboard data: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	createdDashboard, err := h.controllerService.CreateDashboard(ctx, &dashboard, userID)
	if err != nil {
		h.log.WithError(err).WithField("user_id", userID).Error("Failed to create controller dashboard")
		utils.SendError(c, http.StatusInternalServerError, "Failed to create dashboard")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":   true,
		"data":      createdDashboard,
		"message":   "Dashboard created successfully",
		"timestamp": time.Now().UTC(),
	})
}

// UpdateControllerDashboard updates an existing controller dashboard
func (h *Handlers) UpdateControllerDashboard(c *gin.Context) {
	dashboardID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid dashboard ID")
		return
	}

	userID := h.getControllerUserID(c)

	var dashboard models.ControllerDashboard
	if err := c.ShouldBindJSON(&dashboard); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid dashboard data: "+err.Error())
		return
	}

	dashboard.ID = dashboardID

	ctx := context.WithValue(context.Background(), "is_local_request", isLocal(c.Request))
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = h.controllerService.UpdateDashboard(ctx, &dashboard, userID)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient permissions") {
			utils.SendError(c, http.StatusForbidden, "Insufficient permissions to update dashboard")
			return
		}
		h.log.WithError(err).WithFields(logrus.Fields{
			"dashboard_id": dashboardID,
			"user_id":      userID,
		}).Error("Failed to update controller dashboard")
		utils.SendError(c, http.StatusInternalServerError, "Failed to update dashboard")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Dashboard updated successfully",
		"timestamp": time.Now().UTC(),
	})
}

func isLocal(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, it might be because the port is missing.
		// In that case, RemoteAddr should just be the IP address.
		host = r.RemoteAddr
	}

	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

// DeleteControllerDashboard deletes a controller dashboard
func (h *Handlers) DeleteControllerDashboard(c *gin.Context) {
	dashboardID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid dashboard ID")
		return
	}

	userID := h.getControllerUserID(c)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = h.controllerService.DeleteDashboard(ctx, dashboardID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient permissions") {
			utils.SendError(c, http.StatusForbidden, "Insufficient permissions to delete dashboard")
			return
		}
		h.log.WithError(err).WithFields(logrus.Fields{
			"dashboard_id": dashboardID,
			"user_id":      userID,
		}).Error("Failed to delete controller dashboard")
		utils.SendError(c, http.StatusInternalServerError, "Failed to delete dashboard")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Dashboard deleted successfully",
		"timestamp": time.Now().UTC(),
	})
}

// DuplicateControllerDashboard creates a copy of an existing controller dashboard
func (h *Handlers) DuplicateControllerDashboard(c *gin.Context) {
	dashboardID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid dashboard ID")
		return
	}

	userID := h.getControllerUserID(c)

	var request struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request data: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	duplicatedDashboard, err := h.controllerService.DuplicateDashboard(ctx, dashboardID, request.Name, userID)
	if err != nil {
		if strings.Contains(err.Error(), "access denied") {
			utils.SendError(c, http.StatusForbidden, "Access denied to original dashboard")
			return
		}
		h.log.WithError(err).WithFields(logrus.Fields{
			"dashboard_id": dashboardID,
			"user_id":      userID,
		}).Error("Failed to duplicate controller dashboard")
		utils.SendError(c, http.StatusInternalServerError, "Failed to duplicate dashboard")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":   true,
		"data":      duplicatedDashboard,
		"message":   "Dashboard duplicated successfully",
		"timestamp": time.Now().UTC(),
	})
}

// ToggleControllerDashboardFavorite toggles the favorite status of a controller dashboard
func (h *Handlers) ToggleControllerDashboardFavorite(c *gin.Context) {
	dashboardID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid dashboard ID")
		return
	}

	userID := h.getControllerUserID(c)
	// Default to nil user for unauthenticated access, same as other controller endpoints
	var userIDValue int
	if userID != nil {
		userIDValue = *userID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = h.controllerService.ToggleFavorite(ctx, dashboardID, userIDValue)
	if err != nil {
		if strings.Contains(err.Error(), "access denied") {
			utils.SendError(c, http.StatusForbidden, "Access denied")
			return
		}
		h.log.WithError(err).WithFields(logrus.Fields{
			"dashboard_id": dashboardID,
			"user_id":      userIDValue,
		}).Error("Failed to toggle favorite")
		utils.SendError(c, http.StatusInternalServerError, "Failed to toggle favorite")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Favorite status toggled successfully",
		"timestamp": time.Now().UTC(),
	})
}

// Controller Template Operations

// GetControllerTemplates retrieves dashboard templates
func (h *Handlers) GetControllerTemplates(c *gin.Context) {
	userID := h.getControllerUserID(c)
	includePublic := c.Query("include_public") != "false" // Default to true

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	templates, err := h.controllerService.GetTemplates(ctx, userID, includePublic)
	if err != nil {
		h.log.WithError(err).WithField("user_id", userID).Error("Failed to get controller templates")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve templates")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      templates,
		"count":     len(templates),
		"timestamp": time.Now().UTC(),
	})
}

// CreateControllerTemplate creates a new dashboard template
func (h *Handlers) CreateControllerTemplate(c *gin.Context) {
	userID := h.getControllerUserID(c)

	var template models.ControllerTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid template data: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	createdTemplate, err := h.controllerService.CreateTemplate(ctx, &template, userID)
	if err != nil {
		h.log.WithError(err).WithField("user_id", userID).Error("Failed to create controller template")
		utils.SendError(c, http.StatusInternalServerError, "Failed to create template")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":   true,
		"data":      createdTemplate,
		"message":   "Template created successfully",
		"timestamp": time.Now().UTC(),
	})
}

// ApplyControllerTemplate creates a dashboard from a template
func (h *Handlers) ApplyControllerTemplate(c *gin.Context) {
	templateID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid template ID")
		return
	}

	userID := h.getControllerUserID(c)

	var request struct {
		Name      string                 `json:"name" binding:"required"`
		Variables map[string]interface{} `json:"variables"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request data: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dashboard, err := h.controllerService.ApplyTemplate(ctx, templateID, request.Name, request.Variables, userID)
	if err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"template_id": templateID,
			"user_id":     userID,
		}).Error("Failed to apply controller template")
		utils.SendError(c, http.StatusInternalServerError, "Failed to apply template")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":   true,
		"data":      dashboard,
		"message":   "Dashboard created from template successfully",
		"timestamp": time.Now().UTC(),
	})
}

// Element Action Execution

// ExecuteControllerElementAction executes an action on a dashboard element
func (h *Handlers) ExecuteControllerElementAction(c *gin.Context) {
	dashboardID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid dashboard ID")
		return
	}

	elementID := c.Param("elementId")
	if elementID == "" {
		utils.SendError(c, http.StatusBadRequest, "Element ID is required")
		return
	}

	userID := h.getControllerUserID(c)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = h.controllerService.ExecuteElementAction(ctx, dashboardID, elementID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "access denied") {
			utils.SendError(c, http.StatusForbidden, "Access denied")
			return
		}
		if strings.Contains(err.Error(), "element not found") {
			utils.SendError(c, http.StatusNotFound, "Element not found")
			return
		}
		h.log.WithError(err).WithFields(logrus.Fields{
			"dashboard_id": dashboardID,
			"element_id":   elementID,
			"user_id":      userID,
		}).Error("Failed to execute controller element action")
		utils.SendError(c, http.StatusInternalServerError, "Failed to execute element action")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Element action executed successfully",
		"timestamp": time.Now().UTC(),
	})
}

// Analytics and Statistics

// GetControllerDashboardStats returns dashboard usage statistics
func (h *Handlers) GetControllerDashboardStats(c *gin.Context) {
	dashboardID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid dashboard ID")
		return
	}

	userID := h.getControllerUserID(c)
	timeRange := c.DefaultQuery("time_range", "week")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := h.controllerService.GetDashboardStats(ctx, dashboardID, timeRange, userID)
	if err != nil {
		if strings.Contains(err.Error(), "access denied") {
			utils.SendError(c, http.StatusForbidden, "Access denied")
			return
		}
		h.log.WithError(err).WithFields(logrus.Fields{
			"dashboard_id": dashboardID,
			"user_id":      userID,
		}).Error("Failed to get controller dashboard stats")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve dashboard stats")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      stats,
		"timestamp": time.Now().UTC(),
	})
}

// GetControllerAnalytics returns user analytics
func (h *Handlers) GetControllerAnalytics(c *gin.Context) {
	userID := h.getControllerUserID(c)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	analytics, err := h.controllerService.GetAnalytics(ctx, userID)
	if err != nil {
		h.log.WithError(err).WithField("user_id", userID).Error("Failed to get controller analytics")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve analytics")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      analytics,
		"timestamp": time.Now().UTC(),
	})
}

// Import/Export Operations

// ExportControllerDashboard exports a dashboard to JSON
func (h *Handlers) ExportControllerDashboard(c *gin.Context) {
	dashboardID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid dashboard ID")
		return
	}

	userID := h.getControllerUserID(c)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exportData, err := h.controllerService.ExportDashboard(ctx, dashboardID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "access denied") {
			utils.SendError(c, http.StatusForbidden, "Access denied")
			return
		}
		h.log.WithError(err).WithFields(logrus.Fields{
			"dashboard_id": dashboardID,
			"user_id":      userID,
		}).Error("Failed to export controller dashboard")
		utils.SendError(c, http.StatusInternalServerError, "Failed to export dashboard")
		return
	}

	// Set appropriate headers for file download
	dashboardName := "dashboard"
	if name, ok := exportData["dashboard"].(map[string]interface{})["name"].(string); ok {
		dashboardName = strings.ReplaceAll(name, " ", "_")
	}

	filename := fmt.Sprintf("controller_%s_%d.json", dashboardName, dashboardID)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/json")

	c.JSON(http.StatusOK, exportData)
}

// ImportControllerDashboard imports a dashboard from JSON
func (h *Handlers) ImportControllerDashboard(c *gin.Context) {
	userID := h.getControllerUserID(c)

	var importData map[string]interface{}
	if err := c.ShouldBindJSON(&importData); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid import data: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dashboard, err := h.controllerService.ImportDashboard(ctx, importData, userID)
	if err != nil {
		h.log.WithError(err).WithField("user_id", userID).Error("Failed to import controller dashboard")
		utils.SendError(c, http.StatusInternalServerError, "Failed to import dashboard")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":   true,
		"data":      dashboard,
		"message":   "Dashboard imported successfully",
		"timestamp": time.Now().UTC(),
	})
}

// Sharing Operations

// ShareControllerDashboard shares a dashboard with another user
func (h *Handlers) ShareControllerDashboard(c *gin.Context) {
	dashboardID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid dashboard ID")
		return
	}

	userID := h.getControllerUserID(c)
	if userID == nil {
		utils.SendError(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	var request struct {
		UserID      int    `json:"user_id" binding:"required"`
		Permissions string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request data: "+err.Error())
		return
	}

	// Validate permissions
	validPermissions := map[string]bool{
		"view":  true,
		"edit":  true,
		"admin": true,
	}
	if !validPermissions[request.Permissions] {
		utils.SendError(c, http.StatusBadRequest, "Invalid permissions. Must be: view, edit, or admin")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = h.controllerService.ShareDashboard(ctx, dashboardID, request.UserID, request.Permissions, *userID)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient permissions") {
			utils.SendError(c, http.StatusForbidden, "Insufficient permissions to share dashboard")
			return
		}
		h.log.WithError(err).WithFields(logrus.Fields{
			"dashboard_id":   dashboardID,
			"target_user_id": request.UserID,
			"shared_by":      *userID,
		}).Error("Failed to share controller dashboard")
		utils.SendError(c, http.StatusInternalServerError, "Failed to share dashboard")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Dashboard shared successfully",
		"timestamp": time.Now().UTC(),
	})
}

// Search Operations

// SearchControllerDashboards searches dashboards with advanced filters
func (h *Handlers) SearchControllerDashboards(c *gin.Context) {
	userID := h.getControllerUserID(c)

	query := c.Query("q")
	category := c.Query("category")
	tagsParam := c.Query("tags")

	var tags []string
	if tagsParam != "" {
		tags = strings.Split(tagsParam, ",")
	}

	filters := controller.DashboardFilters{
		Search:   query,
		Category: category,
		Tags:     tags,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dashboards, err := h.controllerService.GetDashboards(ctx, userID, filters)
	if err != nil {
		h.log.WithError(err).WithField("user_id", userID).Error("Failed to search controller dashboards")
		utils.SendError(c, http.StatusInternalServerError, "Failed to search dashboards")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      dashboards,
		"count":     len(dashboards),
		"query":     query,
		"filters":   filters,
		"timestamp": time.Now().UTC(),
	})
}
