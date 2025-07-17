package dashboard

import (
	"time"
)

// DashboardManager defines the interface for managing user dashboards
type DashboardManager interface {
	GetUserDashboard(userID string) (*Dashboard, error)
	SaveDashboard(userID string, dashboard *Dashboard) error
	GetWidget(widgetID string) (*Widget, error)
	AddWidget(userID string, widget *Widget) error
	RemoveWidget(userID string, widgetID string) error
	UpdateWidgetPosition(userID string, widgetID string, position Position) error
	UpdateWidgetSize(userID string, widgetID string, size Size) error
	UpdateWidgetConfig(userID string, widgetID string, config map[string]interface{}) error
	GetAvailableWidgets() []WidgetDefinition
	GetWidgetData(userID string, widgetID string) (*WidgetData, error)
	RefreshWidget(userID string, widgetID string) error
}

// Dashboard represents a user's dashboard configuration
type Dashboard struct {
	UserID    string            `json:"user_id" db:"user_id"`
	Layout    string            `json:"layout"` // grid, flex, masonry
	Widgets   []Widget          `json:"widgets"`
	Settings  DashboardSettings `json:"settings"`
	CreatedAt time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" db:"updated_at"`
}

// Widget represents a dashboard widget
type Widget struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // device, chart, weather, calendar, etc.
	Title       string                 `json:"title"`
	Position    Position               `json:"position"`
	Size        Size                   `json:"size"`
	Config      map[string]interface{} `json:"config"`
	RefreshRate int                    `json:"refresh_rate"` // seconds
	Visible     bool                   `json:"visible"`
	Locked      bool                   `json:"locked"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Position defines widget position in the dashboard
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"` // z-index for layering
}

// Size defines widget dimensions
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DashboardSettings contains dashboard-wide settings
type DashboardSettings struct {
	GridSize        int                    `json:"grid_size"`
	ShowGrid        bool                   `json:"show_grid"`
	SnapToGrid      bool                   `json:"snap_to_grid"`
	CompactMode     bool                   `json:"compact_mode"`
	Animation       bool                   `json:"animation"`
	AutoRefresh     bool                   `json:"auto_refresh"`
	RefreshInterval int                    `json:"refresh_interval"` // seconds
	BackgroundImage string                 `json:"background_image"`
	BackgroundColor string                 `json:"background_color"`
	Padding         int                    `json:"padding"`
	Gap             int                    `json:"gap"`
	CustomCSS       string                 `json:"custom_css"`
	Responsive      bool                   `json:"responsive"`
	Breakpoints     map[string]Breakpoint  `json:"breakpoints"`
	Custom          map[string]interface{} `json:"custom"`
}

// Breakpoint defines responsive breakpoints
type Breakpoint struct {
	Width   int `json:"width"`
	Columns int `json:"columns"`
	Gap     int `json:"gap"`
}

// WidgetDefinition defines available widget types
type WidgetDefinition struct {
	Type         string         `json:"type"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Icon         string         `json:"icon"`
	Category     string         `json:"category"`
	MinSize      Size           `json:"min_size"`
	MaxSize      Size           `json:"max_size"`
	DefaultSize  Size           `json:"default_size"`
	Configurable []ConfigOption `json:"configurable"`
	DataSources  []string       `json:"data_sources"`
	Permissions  []string       `json:"permissions"`
	Tags         []string       `json:"tags"`
	PreviewImage string         `json:"preview_image"`
	Version      string         `json:"version"`
	Author       string         `json:"author"`
}

// ConfigOption defines a configurable widget option
type ConfigOption struct {
	Key          string      `json:"key"`
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Type         string      `json:"type"` // string, number, boolean, select, color, etc.
	DefaultValue interface{} `json:"default_value"`
	Required     bool        `json:"required"`
	Options      []Option    `json:"options,omitempty"` // for select type
	Min          *float64    `json:"min,omitempty"`     // for number type
	Max          *float64    `json:"max,omitempty"`     // for number type
	Pattern      string      `json:"pattern,omitempty"` // for string validation
	Group        string      `json:"group,omitempty"`   // for organizing options
}

// Option defines a select option
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Icon  string `json:"icon,omitempty"`
}

// WidgetData represents the data returned by a widget
type WidgetData struct {
	WidgetID    string                 `json:"widget_id"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Data        interface{}            `json:"data"`
	Metadata    map[string]interface{} `json:"metadata"`
	LastUpdated time.Time              `json:"last_updated"`
	Status      WidgetStatus           `json:"status"`
	Error       string                 `json:"error,omitempty"`
}

// WidgetStatus represents the status of a widget
type WidgetStatus string

const (
	WidgetStatusLoading WidgetStatus = "loading"
	WidgetStatusReady   WidgetStatus = "ready"
	WidgetStatusError   WidgetStatus = "error"
	WidgetStatusRefresh WidgetStatus = "refreshing"
)

// WidgetRenderer defines the interface for rendering widgets
type WidgetRenderer interface {
	RenderWidget(widget *Widget, context RenderContext) (*WidgetData, error)
	ValidateConfig(config map[string]interface{}) error
	GetDefaultConfig() map[string]interface{}
	SupportsRefresh() bool
	GetRefreshRate() time.Duration
}

// RenderContext provides context for widget rendering
type RenderContext struct {
	UserID      string                 `json:"user_id"`
	Permissions []string               `json:"permissions"`
	Preferences map[string]interface{} `json:"preferences"`
	Dashboard   *Dashboard             `json:"dashboard"`
	Request     map[string]interface{} `json:"request"`
	Cache       WidgetCache            `json:"-"`
}

// WidgetCache defines caching interface for widgets
type WidgetCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, duration time.Duration)
	Delete(key string)
	Clear()
}

// DashboardTemplate represents a predefined dashboard template
type DashboardTemplate struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Icon        string                 `json:"icon"`
	Preview     string                 `json:"preview"`
	Dashboard   Dashboard              `json:"dashboard"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
}

// WidgetEvent represents events that can be triggered by widgets
type WidgetEvent struct {
	WidgetID  string                 `json:"widget_id"`
	Type      string                 `json:"type"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// LayoutEngine defines interface for different layout engines
type LayoutEngine interface {
	ArrangeWidgets(widgets []Widget, settings DashboardSettings) ([]Widget, error)
	ValidateLayout(widgets []Widget, settings DashboardSettings) error
	GetLayoutType() string
	OptimizeLayout(widgets []Widget, settings DashboardSettings) ([]Widget, error)
}

// WidgetRegistry manages widget types and renderers
type WidgetRegistry interface {
	RegisterWidget(definition WidgetDefinition, renderer WidgetRenderer) error
	UnregisterWidget(widgetType string) error
	GetWidget(widgetType string) (WidgetDefinition, WidgetRenderer, error)
	GetAvailableWidgets() []WidgetDefinition
	ValidateWidget(widget *Widget) error
}

// DashboardExport represents exported dashboard data
type DashboardExport struct {
	Version    string                 `json:"version"`
	ExportedAt time.Time              `json:"exported_at"`
	Dashboard  Dashboard              `json:"dashboard"`
	Widgets    []Widget               `json:"widgets"`
	Settings   DashboardSettings      `json:"settings"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// WidgetAnalytics represents widget usage analytics
type WidgetAnalytics struct {
	WidgetID      string    `json:"widget_id"`
	WidgetType    string    `json:"widget_type"`
	UserID        string    `json:"user_id"`
	ViewCount     int       `json:"view_count"`
	InteractCount int       `json:"interact_count"`
	ErrorCount    int       `json:"error_count"`
	AvgLoadTime   float64   `json:"avg_load_time"`
	LastAccessed  time.Time `json:"last_accessed"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DashboardPermission defines permission levels for dashboard operations
type DashboardPermission string

const (
	PermissionView   DashboardPermission = "view"
	PermissionEdit   DashboardPermission = "edit"
	PermissionManage DashboardPermission = "manage"
	PermissionAdmin  DashboardPermission = "admin"
)

// DefaultDashboard returns a default dashboard configuration
func DefaultDashboard(userID string) *Dashboard {
	return &Dashboard{
		UserID: userID,
		Layout: "masonry",
		Widgets: []Widget{
			{
				ID:          "welcome-widget",
				Type:        "welcome",
				Title:       "Welcome to PMA",
				Position:    Position{X: 0, Y: 0, Z: 1},
				Size:        Size{Width: 4, Height: 2},
				Config:      map[string]interface{}{},
				RefreshRate: 0, // Static widget
				Visible:     true,
				Locked:      false,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		},
		Settings: DashboardSettings{
			GridSize:        12,
			ShowGrid:        false,
			SnapToGrid:      true,
			CompactMode:     false,
			Animation:       true,
			AutoRefresh:     true,
			RefreshInterval: 30,
			BackgroundImage: "",
			BackgroundColor: "",
			Padding:         16,
			Gap:             16,
			CustomCSS:       "",
			Responsive:      true,
			Breakpoints: map[string]Breakpoint{
				"xs": {Width: 576, Columns: 1, Gap: 8},
				"sm": {Width: 768, Columns: 2, Gap: 12},
				"md": {Width: 992, Columns: 3, Gap: 16},
				"lg": {Width: 1200, Columns: 4, Gap: 16},
				"xl": {Width: 1400, Columns: 6, Gap: 20},
			},
			Custom: make(map[string]interface{}),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
