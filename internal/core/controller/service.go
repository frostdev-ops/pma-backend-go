package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/sirupsen/logrus"
)

// Service provides controller dashboard business logic
type Service struct {
	controllerRepo repositories.ControllerRepository
	userRepo       repositories.UserRepository
	entityRepo     repositories.EntityRepository
	unifiedService *unified.UnifiedEntityService
	wsHub          *websocket.Hub
	logger         *logrus.Logger
	config         *config.Config
}

// NewService creates a new controller service
func NewService(
	controllerRepo repositories.ControllerRepository,
	userRepo repositories.UserRepository,
	entityRepo repositories.EntityRepository,
	unifiedService *unified.UnifiedEntityService,
	wsHub *websocket.Hub,
	logger *logrus.Logger,
	config *config.Config,
) *Service {
	return &Service{
		controllerRepo: controllerRepo,
		userRepo:       userRepo,
		entityRepo:     entityRepo,
		unifiedService: unifiedService,
		wsHub:          wsHub,
		logger:         logger,
		config:         config,
	}
}

// Dashboard Management

// CreateDashboard creates a new dashboard with validation
func (s *Service) CreateDashboard(ctx context.Context, dashboard *models.ControllerDashboard, userID *int) (*models.ControllerDashboard, error) {
	// Validate dashboard data
	if err := s.validateDashboard(dashboard); err != nil {
		return nil, fmt.Errorf("dashboard validation failed: %w", err)
	}

	// Set default values
	if dashboard.Category == "" {
		dashboard.Category = "custom"
	}
	if dashboard.Version == 0 {
		dashboard.Version = 1
	}
	dashboard.UserID = userID

	// Ensure default configurations are valid JSON
	if err := s.setDefaultConfigurations(dashboard); err != nil {
		return nil, fmt.Errorf("failed to set default configurations: %w", err)
	}

	// Create dashboard
	err := s.controllerRepo.CreateDashboard(ctx, dashboard)
	if err != nil {
		return nil, fmt.Errorf("failed to create dashboard: %w", err)
	}

	// Log usage
	s.logUsage(ctx, dashboard.ID, userID, "create", nil, nil, nil)

	// Broadcast update via WebSocket
	s.broadcastDashboardUpdate("dashboard_created", dashboard.ID, userID)

	s.logger.WithFields(logrus.Fields{
		"dashboard_id":   dashboard.ID,
		"dashboard_name": dashboard.Name,
		"user_id":        userID,
	}).Info("Dashboard created successfully")

	return dashboard, nil
}

// GetDashboard retrieves a dashboard by ID with access control
func (s *Service) GetDashboard(ctx context.Context, dashboardID int, userID *int) (*models.ControllerDashboard, error) {
	// Check access permissions
	if userID != nil {
		_, err := s.controllerRepo.CheckUserAccess(ctx, dashboardID, *userID)
		if err != nil {
			return nil, fmt.Errorf("access denied: %w", err)
		}
	}

	dashboard, err := s.controllerRepo.GetDashboardByID(ctx, dashboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	// Update last accessed timestamp
	go func() {
		s.controllerRepo.UpdateLastAccessed(context.Background(), dashboardID)
		if userID != nil {
			s.logUsage(context.Background(), dashboardID, userID, "view", nil, nil, nil)
		}
	}()

	return dashboard, nil
}

// GetDashboards retrieves dashboards for a user with filtering
func (s *Service) GetDashboards(ctx context.Context, userID *int, filters DashboardFilters) ([]*models.ControllerDashboard, error) {
	var dashboards []*models.ControllerDashboard
	var err error

	if filters.Search != "" || filters.Category != "" || len(filters.Tags) > 0 {
		// Use search functionality
		dashboards, err = s.controllerRepo.SearchDashboards(ctx, userID, filters.Search, filters.Category, filters.Tags)
	} else if filters.Favorites && userID != nil {
		// Get favorite dashboards
		dashboards, err = s.controllerRepo.GetFavoriteDashboards(ctx, *userID)
	} else if filters.Shared && userID != nil {
		// Get user dashboards including shared ones
		dashboards, err = s.controllerRepo.GetDashboardsByUserID(ctx, userID, true)
	} else {
		// Get all accessible dashboards
		dashboards, err = s.controllerRepo.GetAllDashboards(ctx, userID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get dashboards: %w", err)
	}

	return dashboards, nil
}

// UpdateDashboard updates an existing dashboard with validation
func (s *Service) UpdateDashboard(ctx context.Context, dashboard *models.ControllerDashboard, userID *int) error {
	// For local connections, bypass permission checks
	if isLocalRequest(ctx) {
		s.logger.WithFields(logrus.Fields{
			"dashboard_id": dashboard.ID,
			"user_id":      userID,
			"reason":       "local request",
		}).Info("Bypassing permission checks for local dashboard update")
	} else if userID != nil {
		permission, err := s.controllerRepo.CheckUserAccess(ctx, dashboard.ID, *userID)
		if err != nil || (permission != "admin" && permission != "edit") {
			return fmt.Errorf("insufficient permissions to update dashboard")
		}
	}

	// Validate dashboard data
	if err := s.validateDashboard(dashboard); err != nil {
		return fmt.Errorf("dashboard validation failed: %w", err)
	}

	// Update dashboard
	err := s.controllerRepo.UpdateDashboard(ctx, dashboard)
	if err != nil {
		return fmt.Errorf("failed to update dashboard: %w", err)
	}

	// Log usage
	s.logUsage(ctx, dashboard.ID, userID, "update", nil, nil, nil)

	// Broadcast update via WebSocket
	s.broadcastDashboardUpdate("dashboard_updated", dashboard.ID, userID)

	s.logger.WithFields(logrus.Fields{
		"dashboard_id": dashboard.ID,
		"user_id":      userID,
	}).Info("Dashboard updated successfully")

	return nil
}

// DeleteDashboard deletes a dashboard with permission checks
func (s *Service) DeleteDashboard(ctx context.Context, dashboardID int, userID *int) error {
	// Check permissions (only admin/owner can delete)
	if userID != nil {
		permission, err := s.controllerRepo.CheckUserAccess(ctx, dashboardID, *userID)
		if err != nil || permission != "admin" {
			return fmt.Errorf("insufficient permissions to delete dashboard")
		}
	}

	// Delete dashboard
	err := s.controllerRepo.DeleteDashboard(ctx, dashboardID)
	if err != nil {
		return fmt.Errorf("failed to delete dashboard: %w", err)
	}

	// Log usage
	s.logUsage(ctx, dashboardID, userID, "delete", nil, nil, nil)

	// Broadcast update via WebSocket
	s.broadcastDashboardUpdate("dashboard_deleted", dashboardID, userID)

	s.logger.WithFields(logrus.Fields{
		"dashboard_id": dashboardID,
		"user_id":      userID,
	}).Info("Dashboard deleted successfully")

	return nil
}

// DuplicateDashboard creates a copy of an existing dashboard
func (s *Service) DuplicateDashboard(ctx context.Context, dashboardID int, newName string, userID *int) (*models.ControllerDashboard, error) {
	// Check access to original dashboard
	if userID != nil {
		_, err := s.controllerRepo.CheckUserAccess(ctx, dashboardID, *userID)
		if err != nil {
			return nil, fmt.Errorf("access denied to original dashboard: %w", err)
		}
	}

	// Duplicate dashboard
	dashboard, err := s.controllerRepo.DuplicateDashboard(ctx, dashboardID, userID, newName)
	if err != nil {
		return nil, fmt.Errorf("failed to duplicate dashboard: %w", err)
	}

	// Log usage
	s.logUsage(ctx, dashboard.ID, userID, "create", nil, nil, map[string]interface{}{
		"source_dashboard_id": dashboardID,
		"action_type":         "duplicate",
	})

	// Broadcast update via WebSocket
	s.broadcastDashboardUpdate("dashboard_created", dashboard.ID, userID)

	s.logger.WithFields(logrus.Fields{
		"new_dashboard_id": dashboard.ID,
		"source_id":        dashboardID,
		"user_id":          userID,
	}).Info("Dashboard duplicated successfully")

	return dashboard, nil
}

// ToggleFavorite toggles the favorite status of a dashboard
func (s *Service) ToggleFavorite(ctx context.Context, dashboardID int, userID int) error {
	// Check access
	_, err := s.controllerRepo.CheckUserAccess(ctx, dashboardID, userID)
	if err != nil {
		return fmt.Errorf("access denied: %w", err)
	}

	err = s.controllerRepo.ToggleFavorite(ctx, dashboardID, userID)
	if err != nil {
		return fmt.Errorf("failed to toggle favorite: %w", err)
	}

	// Log usage
	s.logUsage(ctx, dashboardID, &userID, "toggle_favorite", nil, nil, nil)

	return nil
}

// Template Management

// CreateTemplate creates a new dashboard template
func (s *Service) CreateTemplate(ctx context.Context, template *models.ControllerTemplate, userID *int) (*models.ControllerTemplate, error) {
	// Validate template
	if err := s.validateTemplate(template); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	template.UserID = userID
	template.UsageCount = 0
	template.Rating = 0.0

	err := s.controllerRepo.CreateTemplate(ctx, template)
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"template_id":   template.ID,
		"template_name": template.Name,
		"user_id":       userID,
	}).Info("Template created successfully")

	return template, nil
}

// GetTemplates retrieves templates with filtering
func (s *Service) GetTemplates(ctx context.Context, userID *int, includePublic bool) ([]*models.ControllerTemplate, error) {
	var templates []*models.ControllerTemplate
	var err error

	if userID != nil {
		templates, err = s.controllerRepo.GetTemplatesByUserID(ctx, userID, includePublic)
	} else {
		templates, err = s.controllerRepo.GetPublicTemplates(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get templates: %w", err)
	}

	return templates, nil
}

// ApplyTemplate creates a dashboard from a template
func (s *Service) ApplyTemplate(ctx context.Context, templateID int, dashboardName string, variables map[string]interface{}, userID *int) (*models.ControllerDashboard, error) {
	// Get template
	template, err := s.controllerRepo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Parse template data
	var templateData map[string]interface{}
	err = json.Unmarshal([]byte(template.TemplateJSON), &templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Apply variables to template
	processedData, err := s.applyTemplateVariables(templateData, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to apply template variables: %w", err)
	}

	// Create dashboard from processed template
	dashboard := &models.ControllerDashboard{
		Name:        dashboardName,
		Description: template.Description,
		Category:    template.Category,
		UserID:      userID,
	}

	// Set configurations from template
	if layout, ok := processedData["layout"]; ok {
		layoutJSON, _ := json.Marshal(layout)
		dashboard.LayoutConfig = string(layoutJSON)
	}
	if elements, ok := processedData["elements"]; ok {
		elementsJSON, _ := json.Marshal(elements)
		dashboard.ElementsJSON = string(elementsJSON)
	}
	if style, ok := processedData["style"]; ok {
		styleJSON, _ := json.Marshal(style)
		dashboard.StyleConfig = string(styleJSON)
	}

	// Set default configurations
	if err := s.setDefaultConfigurations(dashboard); err != nil {
		return nil, fmt.Errorf("failed to set default configurations: %w", err)
	}

	// Create dashboard
	err = s.controllerRepo.CreateDashboard(ctx, dashboard)
	if err != nil {
		return nil, fmt.Errorf("failed to create dashboard from template: %w", err)
	}

	// Increment template usage
	go s.controllerRepo.IncrementTemplateUsage(context.Background(), templateID)

	// Log usage
	s.logUsage(ctx, dashboard.ID, userID, "create", nil, nil, map[string]interface{}{
		"template_id": templateID,
		"action_type": "apply_template",
	})

	s.logger.WithFields(logrus.Fields{
		"dashboard_id": dashboard.ID,
		"template_id":  templateID,
		"user_id":      userID,
	}).Info("Dashboard created from template")

	return dashboard, nil
}

// Element Action Execution

// ExecuteElementAction executes an action on a dashboard element
func (s *Service) ExecuteElementAction(ctx context.Context, dashboardID int, elementID string, userID *int) error {
	// Check access
	if userID != nil {
		permission, err := s.controllerRepo.CheckUserAccess(ctx, dashboardID, *userID)
		if err != nil || permission == "" {
			return fmt.Errorf("access denied: %w", err)
		}
	}

	// Get dashboard to find element
	dashboard, err := s.controllerRepo.GetDashboardByID(ctx, dashboardID)
	if err != nil {
		return fmt.Errorf("failed to get dashboard: %w", err)
	}

	// Parse elements to find the target element
	elements, err := dashboard.GetElements()
	if err != nil {
		return fmt.Errorf("failed to parse dashboard elements: %w", err)
	}

	var targetElement *models.DashboardElement
	for _, element := range elements {
		if element.ID == elementID {
			targetElement = &element
			break
		}
	}

	if targetElement == nil {
		return fmt.Errorf("element not found: %s", elementID)
	}

	// Execute the element's action through unified service
	err = s.executeElementActions(ctx, targetElement)
	if err != nil {
		return fmt.Errorf("failed to execute element action: %w", err)
	}

	// Log usage
	s.logUsage(ctx, dashboardID, userID, "element_action", &elementID, &targetElement.Type, map[string]interface{}{
		"element_config": targetElement.Config,
	})

	return nil
}

// Analytics and Statistics

// GetDashboardStats returns comprehensive dashboard statistics
func (s *Service) GetDashboardStats(ctx context.Context, dashboardID int, timeRange string, userID *int) (map[string]interface{}, error) {
	// Check access
	if userID != nil {
		_, err := s.controllerRepo.CheckUserAccess(ctx, dashboardID, *userID)
		if err != nil {
			return nil, fmt.Errorf("access denied: %w", err)
		}
	}

	stats, err := s.controllerRepo.GetUsageStats(ctx, dashboardID, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard stats: %w", err)
	}

	return stats, nil
}

// GetAnalytics returns user analytics
func (s *Service) GetAnalytics(ctx context.Context, userID *int) (map[string]interface{}, error) {
	analytics, err := s.controllerRepo.GetDashboardAnalytics(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	}

	return analytics, nil
}

// Import/Export

// ExportDashboard exports a dashboard to JSON
func (s *Service) ExportDashboard(ctx context.Context, dashboardID int, userID *int) (map[string]interface{}, error) {
	// Check access
	if userID != nil {
		_, err := s.controllerRepo.CheckUserAccess(ctx, dashboardID, *userID)
		if err != nil {
			return nil, fmt.Errorf("access denied: %w", err)
		}
	}

	export, err := s.controllerRepo.ExportDashboard(ctx, dashboardID)
	if err != nil {
		return nil, fmt.Errorf("failed to export dashboard: %w", err)
	}

	// Log usage
	s.logUsage(ctx, dashboardID, userID, "export", nil, nil, nil)

	return export, nil
}

// ImportDashboard imports a dashboard from JSON
func (s *Service) ImportDashboard(ctx context.Context, data map[string]interface{}, userID *int) (*models.ControllerDashboard, error) {
	dashboard, err := s.controllerRepo.ImportDashboard(ctx, data, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to import dashboard: %w", err)
	}

	// Log usage
	s.logUsage(ctx, dashboard.ID, userID, "import", nil, nil, nil)

	// Broadcast update
	s.broadcastDashboardUpdate("dashboard_created", dashboard.ID, userID)

	s.logger.WithFields(logrus.Fields{
		"dashboard_id": dashboard.ID,
		"user_id":      userID,
	}).Info("Dashboard imported successfully")

	return dashboard, nil
}

// Sharing Management

// ShareDashboard shares a dashboard with another user
func (s *Service) ShareDashboard(ctx context.Context, dashboardID int, targetUserID int, permissions string, sharedBy int) error {
	// Check if the user can share this dashboard
	permission, err := s.controllerRepo.CheckUserAccess(ctx, dashboardID, sharedBy)
	if err != nil || (permission != "admin" && permission != "edit") {
		return fmt.Errorf("insufficient permissions to share dashboard")
	}

	share := &models.ControllerShare{
		DashboardID: dashboardID,
		UserID:      targetUserID,
		Permissions: permissions,
		SharedBy:    sharedBy,
	}

	err = s.controllerRepo.CreateShare(ctx, share)
	if err != nil {
		return fmt.Errorf("failed to share dashboard: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"dashboard_id":   dashboardID,
		"target_user_id": targetUserID,
		"shared_by":      sharedBy,
		"permissions":    permissions,
	}).Info("Dashboard shared successfully")

	return nil
}

// Helper Types

// DashboardFilters represents filters for dashboard queries
type DashboardFilters struct {
	Search    string
	Category  string
	Tags      []string
	Favorites bool
	Shared    bool
}

// Helper Methods

// validateDashboard validates dashboard data
func (s *Service) validateDashboard(dashboard *models.ControllerDashboard) error {
	if dashboard.Name == "" {
		return fmt.Errorf("dashboard name is required")
	}
	if len(dashboard.Name) > 255 {
		return fmt.Errorf("dashboard name too long")
	}
	return nil
}

// validateTemplate validates template data
func (s *Service) validateTemplate(template *models.ControllerTemplate) error {
	if template.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if template.TemplateJSON == "" {
		return fmt.Errorf("template JSON is required")
	}
	// Validate JSON
	var data map[string]interface{}
	err := json.Unmarshal([]byte(template.TemplateJSON), &data)
	if err != nil {
		return fmt.Errorf("invalid template JSON: %w", err)
	}
	return nil
}

// setDefaultConfigurations ensures dashboard has valid default configurations
func (s *Service) setDefaultConfigurations(dashboard *models.ControllerDashboard) error {
	// Set default layout if empty
	if dashboard.LayoutConfig == "" {
		defaultLayout := models.DashboardLayout{
			Columns:    12,
			Rows:       8,
			GridSize:   64,
			Gap:        8,
			Responsive: true,
		}
		err := dashboard.SetLayout(&defaultLayout)
		if err != nil {
			return fmt.Errorf("failed to set default layout: %w", err)
		}
	}

	// Set default style if empty
	if dashboard.StyleConfig == "" {
		defaultStyle := models.DashboardStyle{
			Theme:        "auto",
			BorderRadius: 8,
			Padding:      16,
		}
		err := dashboard.SetStyle(&defaultStyle)
		if err != nil {
			return fmt.Errorf("failed to set default style: %w", err)
		}
	}

	// Set default access if empty
	if dashboard.AccessConfig == "" {
		defaultAccess := models.DashboardAccess{
			Public:       false,
			SharedWith:   []string{},
			RequiresAuth: false,
		}
		err := dashboard.SetAccess(&defaultAccess)
		if err != nil {
			return fmt.Errorf("failed to set default access: %w", err)
		}
	}

	// Set default elements if empty
	if dashboard.ElementsJSON == "" {
		err := dashboard.SetElements([]models.DashboardElement{})
		if err != nil {
			return fmt.Errorf("failed to set default elements: %w", err)
		}
	}

	// Set default tags if empty
	if dashboard.Tags == "" {
		err := dashboard.SetTagsList([]string{})
		if err != nil {
			return fmt.Errorf("failed to set default tags: %w", err)
		}
	}

	return nil
}

// applyTemplateVariables applies variable substitutions to template data
func (s *Service) applyTemplateVariables(templateData map[string]interface{}, variables map[string]interface{}) (map[string]interface{}, error) {
	// Create a deep copy of the template data to avoid modifying the original
	result := make(map[string]interface{})

	for key, value := range templateData {
		processedValue, err := s.processTemplateValue(value, variables)
		if err != nil {
			return nil, fmt.Errorf("failed to process template variable for key %s: %w", key, err)
		}
		result[key] = processedValue
	}

	return result, nil
}

// processTemplateValue recursively processes a value and substitutes variables
func (s *Service) processTemplateValue(value interface{}, variables map[string]interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		// Replace variable placeholders like {{variable_name}}
		return s.substituteStringVariables(v, variables), nil

	case map[string]interface{}:
		// Recursively process nested maps
		result := make(map[string]interface{})
		for key, val := range v {
			processedVal, err := s.processTemplateValue(val, variables)
			if err != nil {
				return nil, err
			}
			result[key] = processedVal
		}
		return result, nil

	case []interface{}:
		// Recursively process arrays
		result := make([]interface{}, len(v))
		for i, val := range v {
			processedVal, err := s.processTemplateValue(val, variables)
			if err != nil {
				return nil, err
			}
			result[i] = processedVal
		}
		return result, nil

	default:
		// Return primitive values as-is
		return value, nil
	}
}

// substituteStringVariables replaces variable placeholders in strings
func (s *Service) substituteStringVariables(str string, variables map[string]interface{}) string {
	// Use regular expression to find and replace {{variable_name}} patterns
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	return re.ReplaceAllStringFunc(str, func(match string) string {
		// Extract variable name (remove {{ and }})
		varName := strings.TrimSpace(match[2 : len(match)-2])

		// Look up variable value
		if value, exists := variables[varName]; exists {
			// Convert value to string
			switch v := value.(type) {
			case string:
				return v
			case int, int64, float64:
				return fmt.Sprintf("%v", v)
			case bool:
				if v {
					return "true"
				}
				return "false"
			default:
				// For complex types, try JSON marshaling
				if data, err := json.Marshal(v); err == nil {
					return string(data)
				}
				return fmt.Sprintf("%v", v)
			}
		}

		// Variable not found, return original placeholder
		return match
	})
}

// executeElementActions executes actions for a dashboard element
func (s *Service) executeElementActions(ctx context.Context, element *models.DashboardElement) error {
	// Handle different element types and their actions
	switch element.Type {
	case "button":
		return s.executeButtonAction(ctx, element)
	case "slider":
		return s.executeSliderAction(ctx, element)
	case "switch":
		return s.executeSwitchAction(ctx, element)
	default:
		return fmt.Errorf("unsupported element type: %s", element.Type)
	}
}

// executeButtonAction executes a button element action
func (s *Service) executeButtonAction(ctx context.Context, element *models.DashboardElement) error {
	// Extract action configuration from element config
	config := element.Config

	// Handle entity bindings
	for _, binding := range element.EntityBindings {
		if binding.EntityID != "" {
			// Execute entity action through unified service
			action := "toggle" // Default action
			if actionValue, ok := config["action"]; ok {
				if actionStr, ok := actionValue.(string); ok {
					action = actionStr
				}
			}

			// Create control action for unified service
			controlAction := types.PMAControlAction{
				EntityID: binding.EntityID,
				Action:   action,
			}
			_, err := s.unifiedService.ExecuteAction(ctx, controlAction)
			if err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"element_id": element.ID,
					"entity_id":  binding.EntityID,
					"action":     action,
				}).Error("Failed to execute entity action")
				return fmt.Errorf("failed to execute entity action: %w", err)
			}
		}
	}

	return nil
}

// executeSliderAction executes a slider element action
func (s *Service) executeSliderAction(ctx context.Context, element *models.DashboardElement) error {
	// Extract slider configuration from element config
	config := element.Config

	// Handle entity bindings for slider elements
	for _, binding := range element.EntityBindings {
		if binding.EntityID != "" {
			// Get slider value from config
			var value interface{} = 50 // Default value
			if sliderConfig, exists := config["slider"]; exists {
				if sliderMap, ok := sliderConfig.(map[string]interface{}); ok {
					if sliderValue, exists := sliderMap["value"]; exists {
						value = sliderValue
					}
				}
			}

			// Create control action for setting entity value
			controlAction := types.PMAControlAction{
				EntityID:   binding.EntityID,
				Action:     "set_value",
				Parameters: map[string]interface{}{"value": value},
			}

			_, err := s.unifiedService.ExecuteAction(ctx, controlAction)
			if err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"element_id": element.ID,
					"entity_id":  binding.EntityID,
					"action":     "set_value",
					"value":      value,
				}).Error("Failed to execute slider action")
				return fmt.Errorf("failed to execute slider action: %w", err)
			}
		}
	}

	return nil
}

// executeSwitchAction executes a switch element action
func (s *Service) executeSwitchAction(ctx context.Context, element *models.DashboardElement) error {
	// Extract switch configuration from element config
	config := element.Config

	// Handle entity bindings for switch elements
	for _, binding := range element.EntityBindings {
		if binding.EntityID != "" {
			// Determine action based on switch state
			action := "toggle" // Default action

			// Check if switch config specifies a particular state
			if switchConfig, exists := config["switch"]; exists {
				if switchMap, ok := switchConfig.(map[string]interface{}); ok {
					if toggled, exists := switchMap["toggled"]; exists {
						if isToggled, ok := toggled.(bool); ok {
							if isToggled {
								action = "turn_on"
							} else {
								action = "turn_off"
							}
						}
					}
				}
			}

			// Create control action for unified service
			controlAction := types.PMAControlAction{
				EntityID: binding.EntityID,
				Action:   action,
			}

			_, err := s.unifiedService.ExecuteAction(ctx, controlAction)
			if err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"element_id": element.ID,
					"entity_id":  binding.EntityID,
					"action":     action,
				}).Error("Failed to execute switch action")
				return fmt.Errorf("failed to execute switch action: %w", err)
			}
		}
	}

	return nil
}

// logUsage logs dashboard usage for analytics
func (s *Service) logUsage(ctx context.Context, dashboardID int, userID *int, action string, elementID *string, elementType *string, metadata map[string]interface{}) {
	metadataJSON := "{}"
	if metadata != nil {
		if data, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	log := &models.ControllerUsageLog{
		DashboardID: dashboardID,
		UserID:      userID,
		Action:      action,
		ElementID:   elementID,
		ElementType: elementType,
		Metadata:    metadataJSON,
	}

	// Log asynchronously to avoid blocking
	go func() {
		err := s.controllerRepo.LogUsage(context.Background(), log)
		if err != nil {
			s.logger.WithError(err).Error("Failed to log dashboard usage")
		}
	}()
}

// broadcastDashboardUpdate broadcasts dashboard updates via WebSocket
func (s *Service) broadcastDashboardUpdate(eventType string, dashboardID int, userID *int) {
	if s.wsHub == nil {
		return
	}

	message := map[string]interface{}{
		"type":         eventType,
		"dashboard_id": dashboardID,
		"user_id":      userID,
		"timestamp":    time.Now().UTC(),
	}

	// Broadcast to all connected clients
	// In a real implementation, you might want to filter by user permissions
	go s.wsHub.BroadcastToTopic("dashboards", eventType, message)
}

func isLocalRequest(ctx context.Context) bool {
	// This is a simplified check. In a real-world scenario, you would
	// need a more robust way to identify local requests, such as checking
	// the request's remote address.
	// For this example, we'll assume a context value is set for local requests.
	if local, ok := ctx.Value("is_local_request").(bool); ok && local {
		return true
	}
	return false
}
