package models

import (
	"encoding/json"
	"time"
)

// ControllerDashboard represents a controller dashboard
type ControllerDashboard struct {
	ID           int        `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	Description  string     `json:"description" db:"description"`
	Category     string     `json:"category" db:"category"`
	LayoutConfig string     `json:"layout_config" db:"layout_config"` // JSON
	ElementsJSON string     `json:"elements_json" db:"elements_json"` // JSON
	StyleConfig  string     `json:"style_config" db:"style_config"`   // JSON
	AccessConfig string     `json:"access_config" db:"access_config"` // JSON
	IsFavorite   bool       `json:"is_favorite" db:"is_favorite"`
	Tags         string     `json:"tags" db:"tags"` // JSON array
	ThumbnailURL *string    `json:"thumbnail_url" db:"thumbnail_url"`
	Version      int        `json:"version" db:"version"`
	UserID       *int       `json:"user_id" db:"user_id"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	LastAccessed *time.Time `json:"last_accessed" db:"last_accessed"`
}

// DashboardLayout represents the layout configuration
type DashboardLayout struct {
	Columns    int  `json:"columns"`
	Rows       int  `json:"rows"`
	GridSize   int  `json:"grid_size"`
	Gap        int  `json:"gap"`
	Responsive bool `json:"responsive"`
}

// DashboardElement represents a single dashboard element
type DashboardElement struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"`
	Position       ElementPosition        `json:"position"`
	Config         map[string]interface{} `json:"config"`
	Style          map[string]interface{} `json:"style"`
	Behavior       map[string]interface{} `json:"behavior"`
	EntityBindings []EntityBinding        `json:"entity_bindings"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
}

// ElementPosition represents element position and size
type ElementPosition struct {
	X      int  `json:"x"`
	Y      int  `json:"y"`
	Width  int  `json:"width"`
	Height int  `json:"height"`
	ZIndex *int `json:"z_index,omitempty"`
}

// EntityBinding represents entity data binding
type EntityBinding struct {
	ID         string `json:"id"`
	EntityID   string `json:"entity_id"`
	Property   string `json:"property"`
	UpdateMode string `json:"update_mode"`
}

// DashboardStyle represents style configuration
type DashboardStyle struct {
	Theme        string `json:"theme"`
	BorderRadius int    `json:"border_radius"`
	Padding      int    `json:"padding"`
	Background   string `json:"background,omitempty"`
}

// DashboardAccess represents access control configuration
type DashboardAccess struct {
	Public       bool     `json:"public"`
	SharedWith   []string `json:"shared_with"`
	RequiresAuth bool     `json:"requires_auth"`
}

// ControllerTemplate represents a dashboard template
type ControllerTemplate struct {
	ID            int       `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Description   string    `json:"description" db:"description"`
	Category      string    `json:"category" db:"category"`
	TemplateJSON  string    `json:"template_json" db:"template_json"`   // JSON
	VariablesJSON string    `json:"variables_json" db:"variables_json"` // JSON
	ThumbnailURL  *string   `json:"thumbnail_url" db:"thumbnail_url"`
	UsageCount    int       `json:"usage_count" db:"usage_count"`
	Rating        float64   `json:"rating" db:"rating"`
	IsPublic      bool      `json:"is_public" db:"is_public"`
	UserID        *int      `json:"user_id" db:"user_id"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// TemplateVariable represents a template variable
type TemplateVariable struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Default     interface{} `json:"default"`
	Description string      `json:"description"`
	EntityType  string      `json:"entity_type,omitempty"`
	Required    bool        `json:"required,omitempty"`
}

// ControllerShare represents dashboard sharing
type ControllerShare struct {
	ID          int        `json:"id" db:"id"`
	DashboardID int        `json:"dashboard_id" db:"dashboard_id"`
	UserID      int        `json:"user_id" db:"user_id"`
	Permissions string     `json:"permissions" db:"permissions"`
	SharedBy    int        `json:"shared_by" db:"shared_by"`
	ExpiresAt   *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// ControllerUsageLog represents dashboard usage analytics
type ControllerUsageLog struct {
	ID          int       `json:"id" db:"id"`
	DashboardID int       `json:"dashboard_id" db:"dashboard_id"`
	UserID      *int      `json:"user_id" db:"user_id"`
	Action      string    `json:"action" db:"action"`
	ElementID   *string   `json:"element_id" db:"element_id"`
	ElementType *string   `json:"element_type" db:"element_type"`
	SessionID   *string   `json:"session_id" db:"session_id"`
	IPAddress   *string   `json:"ip_address" db:"ip_address"`
	UserAgent   *string   `json:"user_agent" db:"user_agent"`
	DurationMS  *int      `json:"duration_ms" db:"duration_ms"`
	Metadata    string    `json:"metadata" db:"metadata"` // JSON
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Helper methods for JSON marshaling/unmarshaling

// GetLayout returns the parsed layout configuration
func (d *ControllerDashboard) GetLayout() (*DashboardLayout, error) {
	var layout DashboardLayout
	err := json.Unmarshal([]byte(d.LayoutConfig), &layout)
	return &layout, err
}

// SetLayout sets the layout configuration as JSON
func (d *ControllerDashboard) SetLayout(layout *DashboardLayout) error {
	data, err := json.Marshal(layout)
	if err != nil {
		return err
	}
	d.LayoutConfig = string(data)
	return nil
}

// GetElements returns the parsed elements array
func (d *ControllerDashboard) GetElements() ([]DashboardElement, error) {
	var elements []DashboardElement
	err := json.Unmarshal([]byte(d.ElementsJSON), &elements)
	return elements, err
}

// SetElements sets the elements array as JSON
func (d *ControllerDashboard) SetElements(elements []DashboardElement) error {
	data, err := json.Marshal(elements)
	if err != nil {
		return err
	}
	d.ElementsJSON = string(data)
	return nil
}

// GetStyle returns the parsed style configuration
func (d *ControllerDashboard) GetStyle() (*DashboardStyle, error) {
	var style DashboardStyle
	err := json.Unmarshal([]byte(d.StyleConfig), &style)
	return &style, err
}

// SetStyle sets the style configuration as JSON
func (d *ControllerDashboard) SetStyle(style *DashboardStyle) error {
	data, err := json.Marshal(style)
	if err != nil {
		return err
	}
	d.StyleConfig = string(data)
	return nil
}

// GetAccess returns the parsed access configuration
func (d *ControllerDashboard) GetAccess() (*DashboardAccess, error) {
	var access DashboardAccess
	err := json.Unmarshal([]byte(d.AccessConfig), &access)
	return &access, err
}

// SetAccess sets the access configuration as JSON
func (d *ControllerDashboard) SetAccess(access *DashboardAccess) error {
	data, err := json.Marshal(access)
	if err != nil {
		return err
	}
	d.AccessConfig = string(data)
	return nil
}

// GetTagsList returns the parsed tags array
func (d *ControllerDashboard) GetTagsList() ([]string, error) {
	var tags []string
	err := json.Unmarshal([]byte(d.Tags), &tags)
	return tags, err
}

// SetTagsList sets the tags array as JSON
func (d *ControllerDashboard) SetTagsList(tags []string) error {
	data, err := json.Marshal(tags)
	if err != nil {
		return err
	}
	d.Tags = string(data)
	return nil
}

// GetVariables returns the parsed template variables
func (t *ControllerTemplate) GetVariables() ([]TemplateVariable, error) {
	var variables []TemplateVariable
	err := json.Unmarshal([]byte(t.VariablesJSON), &variables)
	return variables, err
}

// SetVariables sets the template variables as JSON
func (t *ControllerTemplate) SetVariables(variables []TemplateVariable) error {
	data, err := json.Marshal(variables)
	if err != nil {
		return err
	}
	t.VariablesJSON = string(data)
	return nil
}

// GetMetadata returns the parsed usage metadata
func (u *ControllerUsageLog) GetMetadata() (map[string]interface{}, error) {
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(u.Metadata), &metadata)
	return metadata, err
}

// SetMetadata sets the usage metadata as JSON
func (u *ControllerUsageLog) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	u.Metadata = string(data)
	return nil
}
