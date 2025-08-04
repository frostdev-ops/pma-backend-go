package preferences

import (
	"time"
)

// PreferencesManager defines the interface for managing user preferences
type PreferencesManager interface {
	GetUserPreferences(userID string) (*UserPreferences, error)
	UpdateUserPreferences(userID string, prefs *UserPreferences) error
	GetPreference(userID string, key string) (interface{}, error)
	SetPreference(userID string, key string, value interface{}) error
	ResetToDefaults(userID string) error
	ExportPreferences(userID string) ([]byte, error)
	ImportPreferences(userID string, data []byte) error
}

// UserPreferences represents the complete set of user preferences
type UserPreferences struct {
	UserID         string                   `json:"user_id" db:"user_id"`
	Theme          ThemePreferences         `json:"theme"`
	Notifications  NotificationPreferences  `json:"notifications"`
	Dashboard      DashboardPreferences     `json:"dashboard"`
	Navbar         NavbarPreferences        `json:"navbar"`
	Automation     AutomationPreferences    `json:"automation"`
	Locale         LocalePreferences        `json:"locale"`
	Privacy        PrivacyPreferences       `json:"privacy"`
	Accessibility  AccessibilityPreferences `json:"accessibility"`
	CustomSettings map[string]interface{}   `json:"custom_settings"`
	UpdatedAt      time.Time                `json:"updated_at" db:"updated_at"`
}

// ThemePreferences contains theme-related settings
type ThemePreferences struct {
	ColorScheme   string `json:"color_scheme"` // light, dark, auto
	PrimaryColor  string `json:"primary_color"`
	AccentColor   string `json:"accent_color"`
	FontSize      string `json:"font_size"` // small, medium, large
	FontFamily    string `json:"font_family"`
	CustomCSS     string `json:"custom_css"`
	HighContrast  bool   `json:"high_contrast"`
	ReducedMotion bool   `json:"reduced_motion"`
}

// NotificationPreferences contains notification settings
type NotificationPreferences struct {
	Enabled       bool                            `json:"enabled"`
	Channels      []NotificationChannel           `json:"channels"`
	QuietHours    QuietHoursConfig                `json:"quiet_hours"`
	Priorities    map[string]NotificationPriority `json:"priorities"`
	Subscriptions []NotificationSubscription      `json:"subscriptions"`
}

// NotificationChannel defines a notification delivery method
type NotificationChannel struct {
	Type    string                 `json:"type"` // email, push, sms, webhook
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// NotificationPriority defines priority levels for notifications
type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "low"
	PriorityMedium   NotificationPriority = "medium"
	PriorityHigh     NotificationPriority = "high"
	PriorityCritical NotificationPriority = "critical"
)

// QuietHoursConfig defines when to suppress notifications
type QuietHoursConfig struct {
	Enabled   bool     `json:"enabled"`
	StartTime string   `json:"start_time"` // HH:MM
	EndTime   string   `json:"end_time"`   // HH:MM
	Timezone  string   `json:"timezone"`
	Override  []string `json:"override"` // alert types that override quiet hours
}

// NotificationSubscription defines what events a user wants to be notified about
type NotificationSubscription struct {
	Type     string                 `json:"type"`
	Enabled  bool                   `json:"enabled"`
	Filters  map[string]interface{} `json:"filters"`
	Channels []string               `json:"channels"`
}

// DashboardPreferences contains dashboard layout settings
type DashboardPreferences struct {
	Layout          string                 `json:"layout"` // grid, flex, masonry
	GridSize        int                    `json:"grid_size"`
	RefreshInterval int                    `json:"refresh_interval"` // seconds
	ShowGrid        bool                   `json:"show_grid"`
	CompactMode     bool                   `json:"compact_mode"`
	CustomCSS       string                 `json:"custom_css"`
	WidgetAnimation bool                   `json:"widget_animation"`
	AutoArrange     bool                   `json:"auto_arrange"`
	BackgroundImage string                 `json:"background_image"`
	Settings        map[string]interface{} `json:"settings"`
}

// NavbarPreferences contains navbar configuration settings
type NavbarPreferences struct {
	Configurations        []NavbarConfiguration `json:"configurations"`
	ActiveConfigurationID string                `json:"active_configuration_id"`
	LastModified          time.Time             `json:"last_modified"`
}

// NavbarConfiguration represents a complete navbar configuration
type NavbarConfiguration struct {
	ID                      string          `json:"id"`
	Name                    string          `json:"name"`
	Description             string          `json:"description,omitempty"`
	Location                string          `json:"location"`                           // top, bottom, left, right
	Positioning             string          `json:"positioning,omitempty"`              // overlay, push, static
	Size                    string          `json:"size"`                               // compact, normal, large
	ButtonSize              string          `json:"button_size"`                        // small, medium, large
	Orientation             string          `json:"orientation,omitempty"`              // horizontal, vertical
	HorizontalJustification string          `json:"horizontal_justification,omitempty"` // start, center, between, around, evenly
	Style                   NavbarStyle     `json:"style"`
	Elements                []NavbarElement `json:"elements"`
	AutoHide                bool            `json:"auto_hide,omitempty"`
	AutoHideDelay           int             `json:"auto_hide_delay,omitempty"`
	CollapseOnMobile        bool            `json:"collapse_on_mobile,omitempty"`
	ShowLabels              bool            `json:"show_labels"`
	ShowBadges              bool            `json:"show_badges"`
	ShowCollapseButton      bool            `json:"show_collapse_button,omitempty"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at"`
	IsDefault               bool            `json:"is_default,omitempty"`
	IsActive                bool            `json:"is_active,omitempty"`
}

// NavbarStyle contains styling options for the navbar
type NavbarStyle struct {
	BackgroundColor string  `json:"background_color,omitempty"`
	BorderColor     string  `json:"border_color,omitempty"`
	BorderWidth     int     `json:"border_width,omitempty"`
	BorderRadius    int     `json:"border_radius,omitempty"`
	Padding         int     `json:"padding,omitempty"`
	Gap             int     `json:"gap,omitempty"`
	Opacity         float64 `json:"opacity,omitempty"`
	Blur            bool    `json:"blur,omitempty"`
	Shadow          bool    `json:"shadow,omitempty"`
	CustomCSS       string  `json:"custom_css,omitempty"`
}

// NavbarElement represents an individual navbar element
type NavbarElement struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"` // button, widget, dashboard_link, separator, spacer
	Order        int                    `json:"order"`
	Visible      bool                   `json:"visible"`
	Label        string                 `json:"label,omitempty"`
	Icon         string                 `json:"icon,omitempty"`
	Path         string                 `json:"path,omitempty"`
	Color        string                 `json:"color,omitempty"`
	Badge        string                 `json:"badge,omitempty"`
	Condition    string                 `json:"condition,omitempty"`
	WidgetType   string                 `json:"widget_type,omitempty"`
	WidgetConfig map[string]interface{} `json:"widget_config,omitempty"`
	DashboardID  string                 `json:"dashboard_id,omitempty"`
	DisplayMode  string                 `json:"display_mode,omitempty"` // link, embed
	Style        *NavbarElementStyle    `json:"style,omitempty"`
}

// NavbarElementStyle contains element-specific styling
type NavbarElementStyle struct {
	BackgroundColor string `json:"background_color,omitempty"`
	TextColor       string `json:"text_color,omitempty"`
	BorderColor     string `json:"border_color,omitempty"`
	BorderRadius    int    `json:"border_radius,omitempty"`
	Padding         int    `json:"padding,omitempty"`
	FontSize        string `json:"font_size,omitempty"`
	FontWeight      string `json:"font_weight,omitempty"`
	CustomCSS       string `json:"custom_css,omitempty"`
}

// AutomationPreferences contains automation-related settings
type AutomationPreferences struct {
	SuggestionsEnabled  bool                   `json:"suggestions_enabled"`
	AutomationGroups    []AutomationGroup      `json:"automation_groups"`
	FavoriteAutomations []string               `json:"favorite_automations"`
	ExecutionHistory    bool                   `json:"execution_history"`
	DebugMode           bool                   `json:"debug_mode"`
	CustomVariables     map[string]interface{} `json:"custom_variables"`
	QuickActions        []QuickAction          `json:"quick_actions"`
}

// AutomationGroup organizes automations into logical groups
type AutomationGroup struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Icon        string   `json:"icon"`
	Color       string   `json:"color"`
	Automations []string `json:"automations"`
	Order       int      `json:"order"`
	Collapsed   bool     `json:"collapsed"`
}

// QuickAction defines a quick action button
type QuickAction struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Icon       string                 `json:"icon"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
	Order      int                    `json:"order"`
	Visible    bool                   `json:"visible"`
}

// LocalePreferences contains language and regional settings
type LocalePreferences struct {
	Language     string `json:"language"` // en, es, fr, de, zh, ja
	Region       string `json:"region"`   // US, ES, FR, DE, CN, JP
	DateFormat   string `json:"date_format"`
	TimeFormat   string `json:"time_format"` // 12h, 24h
	Temperature  string `json:"temperature"` // celsius, fahrenheit
	Currency     string `json:"currency"`
	Timezone     string `json:"timezone"`
	FirstDayWeek int    `json:"first_day_week"` // 0=Sunday, 1=Monday
}

// PrivacyPreferences contains privacy and security settings
type PrivacyPreferences struct {
	DataCollection   DataCollectionSettings `json:"data_collection"`
	ActivityTracking bool                   `json:"activity_tracking"`
	LocationSharing  bool                   `json:"location_sharing"`
	DeviceDiscovery  bool                   `json:"device_discovery"`
	APIAccess        []APIAccessRule        `json:"api_access"`
	DataRetention    DataRetentionPolicy    `json:"data_retention"`
	TwoFactorAuth    bool                   `json:"two_factor_auth"`
	SessionTimeout   int                    `json:"session_timeout"` // minutes
}

// DataCollectionSettings controls what data is collected
type DataCollectionSettings struct {
	Analytics       bool `json:"analytics"`
	Diagnostics     bool `json:"diagnostics"`
	UsageStatistics bool `json:"usage_statistics"`
	ErrorReporting  bool `json:"error_reporting"`
	Performance     bool `json:"performance"`
}

// APIAccessRule defines access rules for external applications
type APIAccessRule struct {
	ClientID    string     `json:"client_id"`
	ClientName  string     `json:"client_name"`
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsed    *time.Time `json:"last_used"`
}

// DataRetentionPolicy defines how long data is kept
type DataRetentionPolicy struct {
	LogRetentionDays     int `json:"log_retention_days"`
	MetricsRetentionDays int `json:"metrics_retention_days"`
	BackupRetentionDays  int `json:"backup_retention_days"`
	HistoryRetentionDays int `json:"history_retention_days"`
}

// AccessibilityPreferences contains accessibility settings
type AccessibilityPreferences struct {
	HighContrast   bool   `json:"high_contrast"`
	LargeText      bool   `json:"large_text"`
	ReducedMotion  bool   `json:"reduced_motion"`
	ScreenReader   bool   `json:"screen_reader"`
	KeyboardNav    bool   `json:"keyboard_nav"`
	ColorBlindMode string `json:"color_blind_mode"` // none, protanopia, deuteranopia, tritanopia
	FocusIndicator bool   `json:"focus_indicator"`
	AudioCues      bool   `json:"audio_cues"`
}

// PreferenceSection defines the structure for exporting/importing specific sections
type PreferenceSection struct {
	Section string      `json:"section"`
	Data    interface{} `json:"data"`
	Version string      `json:"version"`
}

// DefaultPreferences returns the default preferences for new users
func DefaultPreferences() *UserPreferences {
	return &UserPreferences{
		Theme: ThemePreferences{
			ColorScheme:   "auto",
			PrimaryColor:  "#1976D2",
			AccentColor:   "#FF5722",
			FontSize:      "medium",
			FontFamily:    "system-ui",
			CustomCSS:     "",
			HighContrast:  false,
			ReducedMotion: false,
		},
		Notifications: NotificationPreferences{
			Enabled: true,
			Channels: []NotificationChannel{
				{Type: "push", Enabled: true, Config: make(map[string]interface{})},
			},
			QuietHours: QuietHoursConfig{
				Enabled:   false,
				StartTime: "22:00",
				EndTime:   "08:00",
				Timezone:  "UTC",
				Override:  []string{"critical", "security"},
			},
			Priorities: map[string]NotificationPriority{
				"device_offline":    PriorityMedium,
				"automation_failed": PriorityHigh,
				"security_alert":    PriorityCritical,
				"system_update":     PriorityLow,
			},
			Subscriptions: []NotificationSubscription{},
		},
		Dashboard: DashboardPreferences{
			Layout:          "masonry",
			GridSize:        4,
			RefreshInterval: 30,
			ShowGrid:        false,
			CompactMode:     false,
			CustomCSS:       "",
			WidgetAnimation: true,
			AutoArrange:     true,
			BackgroundImage: "",
			Settings:        make(map[string]interface{}),
		},
		Navbar: NavbarPreferences{
			Configurations:        []NavbarConfiguration{defaultNavbarConfiguration()},
			ActiveConfigurationID: "default",
			LastModified:          time.Now(),
		},
		Automation: AutomationPreferences{
			SuggestionsEnabled:  true,
			AutomationGroups:    []AutomationGroup{},
			FavoriteAutomations: []string{},
			ExecutionHistory:    true,
			DebugMode:           false,
			CustomVariables:     make(map[string]interface{}),
			QuickActions:        []QuickAction{},
		},
		Locale: LocalePreferences{
			Language:     "en",
			Region:       "US",
			DateFormat:   "MM/DD/YYYY",
			TimeFormat:   "12h",
			Temperature:  "fahrenheit",
			Currency:     "USD",
			Timezone:     "America/New_York",
			FirstDayWeek: 0,
		},
		Privacy: PrivacyPreferences{
			DataCollection: DataCollectionSettings{
				Analytics:       true,
				Diagnostics:     true,
				UsageStatistics: true,
				ErrorReporting:  true,
				Performance:     true,
			},
			ActivityTracking: true,
			LocationSharing:  false,
			DeviceDiscovery:  true,
			APIAccess:        []APIAccessRule{},
			DataRetention: DataRetentionPolicy{
				LogRetentionDays:     30,
				MetricsRetentionDays: 90,
				BackupRetentionDays:  365,
				HistoryRetentionDays: 180,
			},
			TwoFactorAuth:  false,
			SessionTimeout: 720, // 12 hours
		},
		Accessibility: AccessibilityPreferences{
			HighContrast:   false,
			LargeText:      false,
			ReducedMotion:  false,
			ScreenReader:   false,
			KeyboardNav:    false,
			ColorBlindMode: "none",
			FocusIndicator: true,
			AudioCues:      false,
		},
		CustomSettings: make(map[string]interface{}),
		UpdatedAt:      time.Now(),
	}
}

// defaultNavbarConfiguration returns the default navbar configuration
func defaultNavbarConfiguration() NavbarConfiguration {
	return NavbarConfiguration{
		ID:          "default",
		Name:        "Default Navigation",
		Description: "Default PMA navigation bar",
		Location:    "top",
		Size:        "normal",
		ButtonSize:  "medium",
		Style: NavbarStyle{
			BackgroundColor: "var(--surface-primary)",
			BorderColor:     "var(--border-color)",
			BorderWidth:     1,
			BorderRadius:    8,
			Padding:         8,
			Gap:             4,
			Opacity:         1,
			Blur:            false,
			Shadow:          true,
		},
		Elements: []NavbarElement{
			{
				ID:      "home",
				Type:    "button",
				Order:   0,
				Visible: true,
				Label:   "Dashboard",
				Icon:    "Home",
				Path:    "/",
			},
			{
				ID:      "controllers",
				Type:    "button",
				Order:   1,
				Visible: true,
				Label:   "Controllers",
				Icon:    "LayoutGrid",
				Path:    "/controllers",
			},
			{
				ID:      "separator-1",
				Type:    "separator",
				Order:   2,
				Visible: true,
			},
			{
				ID:        "climate",
				Type:      "button",
				Order:     3,
				Visible:   true,
				Label:     "Climate",
				Icon:      "Thermometer",
				Path:      "/climate",
				Condition: "climateDeviceCount > 0",
			},
			{
				ID:        "cameras",
				Type:      "button",
				Order:     4,
				Visible:   true,
				Label:     "Cameras",
				Icon:      "Camera",
				Path:      "/cameras",
				Condition: "cameraCount > 0",
			},
			{
				ID:      "spacer-1",
				Type:    "spacer",
				Order:   5,
				Visible: true,
			},
			{
				ID:      "ai",
				Type:    "button",
				Order:   6,
				Visible: true,
				Label:   "AI Assistant",
				Icon:    "Bot",
				Path:    "/ai",
			},
			{
				ID:      "settings",
				Type:    "button",
				Order:   7,
				Visible: true,
				Label:   "Settings",
				Icon:    "Settings",
				Path:    "/settings",
			},
		},
		AutoHide:         false,
		AutoHideDelay:    3000,
		CollapseOnMobile: true,
		ShowLabels:       true,
		ShowBadges:       true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		IsDefault:        true,
		IsActive:         true,
	}
}
