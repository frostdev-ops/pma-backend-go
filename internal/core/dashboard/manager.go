package dashboard

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Manager implements the DashboardManager interface
type Manager struct {
	db       *sql.DB
	logger   *logrus.Logger
	registry WidgetRegistry
	cache    WidgetCache
}

// NewManager creates a new dashboard manager
func NewManager(db *sql.DB, logger *logrus.Logger) *Manager {
	registry := NewWidgetRegistry(logger)
	cache := NewMemoryCache()

	manager := &Manager{
		db:       db,
		logger:   logger,
		registry: registry,
		cache:    cache,
	}

	// Register built-in widgets
	manager.registerBuiltinWidgets()

	return manager
}

// GetUserDashboard retrieves the dashboard for a user
func (m *Manager) GetUserDashboard(userID string) (*Dashboard, error) {
	query := `SELECT dashboard FROM user_dashboards WHERE user_id = ?`

	var dashboardJSON string
	err := m.db.QueryRow(query, userID).Scan(&dashboardJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create default dashboard for new users
			defaultDash := DefaultDashboard(userID)
			if saveErr := m.SaveDashboard(userID, defaultDash); saveErr != nil {
				m.logger.WithError(saveErr).Error("Failed to save default dashboard")
			}
			return defaultDash, nil
		}
		return nil, fmt.Errorf("failed to get user dashboard: %w", err)
	}

	var dashboard Dashboard
	if err := json.Unmarshal([]byte(dashboardJSON), &dashboard); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dashboard: %w", err)
	}

	dashboard.UserID = userID
	return &dashboard, nil
}

// SaveDashboard saves a user's dashboard
func (m *Manager) SaveDashboard(userID string, dashboard *Dashboard) error {
	dashboard.UserID = userID
	dashboard.UpdatedAt = time.Now()

	dashboardJSON, err := json.Marshal(dashboard)
	if err != nil {
		return fmt.Errorf("failed to marshal dashboard: %w", err)
	}

	query := `
		INSERT INTO user_dashboards (user_id, dashboard, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			dashboard = excluded.dashboard,
			updated_at = excluded.updated_at
	`

	_, err = m.db.Exec(query, userID, string(dashboardJSON), dashboard.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save dashboard: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"widgets": len(dashboard.Widgets),
		"action":  "save_dashboard",
	}).Info("Dashboard saved")

	return nil
}

// GetWidget retrieves a specific widget by ID
func (m *Manager) GetWidget(widgetID string) (*Widget, error) {
	// This would typically search through user dashboards
	// For now, we'll implement a basic version
	query := `
		SELECT dashboard FROM user_dashboards 
		WHERE dashboard LIKE ?
	`

	searchPattern := fmt.Sprintf("%%\"id\":\"%s\"%%", widgetID)
	rows, err := m.db.Query(query, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search for widget: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dashboardJSON string
		if err := rows.Scan(&dashboardJSON); err != nil {
			continue
		}

		var dashboard Dashboard
		if err := json.Unmarshal([]byte(dashboardJSON), &dashboard); err != nil {
			continue
		}

		for _, widget := range dashboard.Widgets {
			if widget.ID == widgetID {
				return &widget, nil
			}
		}
	}

	return nil, fmt.Errorf("widget not found")
}

// AddWidget adds a new widget to a user's dashboard
func (m *Manager) AddWidget(userID string, widget *Widget) error {
	dashboard, err := m.GetUserDashboard(userID)
	if err != nil {
		return err
	}

	// Generate ID if not provided
	if widget.ID == "" {
		widget.ID = uuid.New().String()
	}

	// Set timestamps
	widget.CreatedAt = time.Now()
	widget.UpdatedAt = time.Now()

	// Validate widget
	if err := m.registry.ValidateWidget(widget); err != nil {
		return fmt.Errorf("widget validation failed: %w", err)
	}

	// Add to dashboard
	dashboard.Widgets = append(dashboard.Widgets, *widget)

	return m.SaveDashboard(userID, dashboard)
}

// RemoveWidget removes a widget from a user's dashboard
func (m *Manager) RemoveWidget(userID string, widgetID string) error {
	dashboard, err := m.GetUserDashboard(userID)
	if err != nil {
		return err
	}

	// Find and remove widget
	for i, widget := range dashboard.Widgets {
		if widget.ID == widgetID {
			dashboard.Widgets = append(dashboard.Widgets[:i], dashboard.Widgets[i+1:]...)
			return m.SaveDashboard(userID, dashboard)
		}
	}

	return fmt.Errorf("widget not found")
}

// UpdateWidgetPosition updates a widget's position
func (m *Manager) UpdateWidgetPosition(userID string, widgetID string, position Position) error {
	dashboard, err := m.GetUserDashboard(userID)
	if err != nil {
		return err
	}

	// Find and update widget
	for i, widget := range dashboard.Widgets {
		if widget.ID == widgetID {
			if widget.Locked {
				return fmt.Errorf("widget is locked and cannot be moved")
			}
			dashboard.Widgets[i].Position = position
			dashboard.Widgets[i].UpdatedAt = time.Now()
			return m.SaveDashboard(userID, dashboard)
		}
	}

	return fmt.Errorf("widget not found")
}

// UpdateWidgetSize updates a widget's size
func (m *Manager) UpdateWidgetSize(userID string, widgetID string, size Size) error {
	dashboard, err := m.GetUserDashboard(userID)
	if err != nil {
		return err
	}

	// Find and update widget
	for i, widget := range dashboard.Widgets {
		if widget.ID == widgetID {
			if widget.Locked {
				return fmt.Errorf("widget is locked and cannot be resized")
			}

			// Validate size constraints
			definition, _, err := m.registry.GetWidget(widget.Type)
			if err == nil {
				if size.Width < definition.MinSize.Width || size.Height < definition.MinSize.Height {
					return fmt.Errorf("size below minimum constraints")
				}
				if definition.MaxSize.Width > 0 && size.Width > definition.MaxSize.Width {
					return fmt.Errorf("size above maximum constraints")
				}
				if definition.MaxSize.Height > 0 && size.Height > definition.MaxSize.Height {
					return fmt.Errorf("size above maximum constraints")
				}
			}

			dashboard.Widgets[i].Size = size
			dashboard.Widgets[i].UpdatedAt = time.Now()
			return m.SaveDashboard(userID, dashboard)
		}
	}

	return fmt.Errorf("widget not found")
}

// UpdateWidgetConfig updates a widget's configuration
func (m *Manager) UpdateWidgetConfig(userID string, widgetID string, config map[string]interface{}) error {
	dashboard, err := m.GetUserDashboard(userID)
	if err != nil {
		return err
	}

	// Find and update widget
	for i, widget := range dashboard.Widgets {
		if widget.ID == widgetID {
			// Validate config if renderer is available
			if _, renderer, err := m.registry.GetWidget(widget.Type); err == nil {
				if err := renderer.ValidateConfig(config); err != nil {
					return fmt.Errorf("config validation failed: %w", err)
				}
			}

			dashboard.Widgets[i].Config = config
			dashboard.Widgets[i].UpdatedAt = time.Now()
			return m.SaveDashboard(userID, dashboard)
		}
	}

	return fmt.Errorf("widget not found")
}

// GetAvailableWidgets returns all available widget types
func (m *Manager) GetAvailableWidgets() []WidgetDefinition {
	return m.registry.GetAvailableWidgets()
}

// GetWidgetData retrieves data for a specific widget
func (m *Manager) GetWidgetData(userID string, widgetID string) (*WidgetData, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("widget_data_%s_%s", userID, widgetID)
	if cachedData, found := m.cache.Get(cacheKey); found {
		if data, ok := cachedData.(*WidgetData); ok {
			return data, nil
		}
	}

	widget, err := m.GetWidget(widgetID)
	if err != nil {
		return nil, err
	}

	_, renderer, err := m.registry.GetWidget(widget.Type)
	if err != nil {
		return nil, fmt.Errorf("widget type not found: %w", err)
	}

	// Create render context
	dashboard, _ := m.GetUserDashboard(userID)
	context := RenderContext{
		UserID:      userID,
		Permissions: []string{}, // TODO: Get user permissions
		Preferences: make(map[string]interface{}),
		Dashboard:   dashboard,
		Request:     make(map[string]interface{}),
		Cache:       m.cache,
	}

	// Render widget
	data, err := renderer.RenderWidget(widget, context)
	if err != nil {
		// Return error data
		return &WidgetData{
			WidgetID:    widgetID,
			Type:        widget.Type,
			Title:       widget.Title,
			Data:        nil,
			Metadata:    make(map[string]interface{}),
			LastUpdated: time.Now(),
			Status:      WidgetStatusError,
			Error:       err.Error(),
		}, nil
	}

	// Cache the data if widget supports refresh
	if renderer.SupportsRefresh() {
		cacheDuration := renderer.GetRefreshRate()
		if cacheDuration > 0 {
			m.cache.Set(cacheKey, data, cacheDuration)
		}
	}

	return data, nil
}

// RefreshWidget forces a refresh of widget data
func (m *Manager) RefreshWidget(userID string, widgetID string) error {
	// Clear cache
	cacheKey := fmt.Sprintf("widget_data_%s_%s", userID, widgetID)
	m.cache.Delete(cacheKey)

	// Get fresh data
	_, err := m.GetWidgetData(userID, widgetID)
	return err
}

// registerBuiltinWidgets registers the built-in widget types
func (m *Manager) registerBuiltinWidgets() {
	// Welcome widget
	welcomeDefinition := WidgetDefinition{
		Type:        "welcome",
		Name:        "Welcome Widget",
		Description: "Welcome message and quick actions",
		Icon:        "welcome",
		Category:    "general",
		MinSize:     Size{Width: 2, Height: 1},
		MaxSize:     Size{Width: 6, Height: 3},
		DefaultSize: Size{Width: 4, Height: 2},
		Configurable: []ConfigOption{
			{
				Key:          "message",
				Name:         "Welcome Message",
				Description:  "Custom welcome message",
				Type:         "string",
				DefaultValue: "Welcome to PMA Home Control",
				Required:     false,
			},
		},
		DataSources: []string{},
		Permissions: []string{"view"},
		Tags:        []string{"welcome", "general"},
		Version:     "1.0",
		Author:      "PMA System",
	}
	welcomeRenderer := &WelcomeRenderer{}
	m.registry.RegisterWidget(welcomeDefinition, welcomeRenderer)

	// Device control widget
	deviceDefinition := WidgetDefinition{
		Type:        "device-control",
		Name:        "Device Control",
		Description: "Control smart home devices",
		Icon:        "device",
		Category:    "devices",
		MinSize:     Size{Width: 1, Height: 1},
		MaxSize:     Size{Width: 4, Height: 4},
		DefaultSize: Size{Width: 2, Height: 2},
		Configurable: []ConfigOption{
			{
				Key:         "device_id",
				Name:        "Device ID",
				Description: "ID of the device to control",
				Type:        "string",
				Required:    true,
			},
			{
				Key:          "show_power",
				Name:         "Show Power Status",
				Description:  "Display power consumption",
				Type:         "boolean",
				DefaultValue: true,
			},
		},
		DataSources: []string{"devices"},
		Permissions: []string{"view", "control"},
		Tags:        []string{"device", "control"},
		Version:     "1.0",
		Author:      "PMA System",
	}
	deviceRenderer := &DeviceControlRenderer{}
	m.registry.RegisterWidget(deviceDefinition, deviceRenderer)

	// System status widget
	statusDefinition := WidgetDefinition{
		Type:        "system-status",
		Name:        "System Status",
		Description: "System health and metrics",
		Icon:        "monitor",
		Category:    "system",
		MinSize:     Size{Width: 2, Height: 2},
		MaxSize:     Size{Width: 6, Height: 4},
		DefaultSize: Size{Width: 3, Height: 2},
		Configurable: []ConfigOption{
			{
				Key:          "metrics",
				Name:         "Metrics to Display",
				Description:  "Select which metrics to show",
				Type:         "select",
				DefaultValue: []string{"cpu", "memory", "disk"},
				Options: []Option{
					{Value: "cpu", Label: "CPU Usage"},
					{Value: "memory", Label: "Memory Usage"},
					{Value: "disk", Label: "Disk Usage"},
					{Value: "network", Label: "Network"},
					{Value: "temperature", Label: "Temperature"},
				},
			},
		},
		DataSources: []string{"system"},
		Permissions: []string{"view"},
		Tags:        []string{"system", "monitoring"},
		Version:     "1.0",
		Author:      "PMA System",
	}
	statusRenderer := &SystemStatusRenderer{}
	m.registry.RegisterWidget(statusDefinition, statusRenderer)
}

// ExportDashboard exports a user's dashboard
func (m *Manager) ExportDashboard(userID string) (*DashboardExport, error) {
	dashboard, err := m.GetUserDashboard(userID)
	if err != nil {
		return nil, err
	}

	export := &DashboardExport{
		Version:    "1.0",
		ExportedAt: time.Now(),
		Dashboard:  *dashboard,
		Widgets:    dashboard.Widgets,
		Settings:   dashboard.Settings,
		Metadata: map[string]interface{}{
			"user_id":      userID,
			"widget_count": len(dashboard.Widgets),
		},
	}

	return export, nil
}

// ImportDashboard imports a dashboard from export data
func (m *Manager) ImportDashboard(userID string, exportData *DashboardExport) error {
	dashboard := exportData.Dashboard
	dashboard.UserID = userID
	dashboard.CreatedAt = time.Now()
	dashboard.UpdatedAt = time.Now()

	// Regenerate widget IDs to avoid conflicts
	for i := range dashboard.Widgets {
		dashboard.Widgets[i].ID = uuid.New().String()
		dashboard.Widgets[i].CreatedAt = time.Now()
		dashboard.Widgets[i].UpdatedAt = time.Now()
	}

	return m.SaveDashboard(userID, &dashboard)
}
