package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// PMAEntity is the base interface that all PMA entities must implement
type PMAEntity interface {
	// Basic identification
	GetID() string
	GetType() PMAEntityType
	GetFriendlyName() string
	GetIcon() string

	// State management
	GetState() PMAEntityState
	GetAttributes() map[string]interface{}
	GetLastUpdated() time.Time

	// Capabilities
	GetCapabilities() []PMACapability
	HasCapability(capability PMACapability) bool

	// Control
	CanControl() bool
	GetAvailableActions() []string
	ExecuteAction(action PMAControlAction) (*PMAControlResult, error)

	// Relationships
	GetRoomID() *string
	GetAreaID() *string
	GetDeviceID() *string

	// Source tracking
	GetMetadata() *PMAMetadata
	GetSource() PMASourceType

	// Quality and reliability
	IsAvailable() bool
	GetQualityScore() float64

	// Serialization
	ToJSON() ([]byte, error)
}

// PMABaseEntity provides a base implementation of PMAEntity
type PMABaseEntity struct {
	ID           string                 `json:"id"`
	Type         PMAEntityType          `json:"type"`
	FriendlyName string                 `json:"friendly_name"`
	Icon         string                 `json:"icon,omitempty"`
	State        PMAEntityState         `json:"state"`
	Attributes   map[string]interface{} `json:"attributes"`
	LastUpdated  time.Time              `json:"last_updated"`
	Capabilities []PMACapability        `json:"capabilities"`
	RoomID       *string                `json:"room_id,omitempty"`
	AreaID       *string                `json:"area_id,omitempty"`
	DeviceID     *string                `json:"device_id,omitempty"`
	Metadata     *PMAMetadata           `json:"metadata"`
	Available    bool                   `json:"available"`
}

// Implement PMAEntity interface for PMABaseEntity
func (e *PMABaseEntity) GetID() string                         { return e.ID }
func (e *PMABaseEntity) GetType() PMAEntityType                { return e.Type }
func (e *PMABaseEntity) GetFriendlyName() string               { return e.FriendlyName }
func (e *PMABaseEntity) GetIcon() string                       { return e.Icon }
func (e *PMABaseEntity) GetState() PMAEntityState              { return e.State }
func (e *PMABaseEntity) GetAttributes() map[string]interface{} { return e.Attributes }
func (e *PMABaseEntity) GetLastUpdated() time.Time             { return e.LastUpdated }
func (e *PMABaseEntity) GetCapabilities() []PMACapability      { return e.Capabilities }
func (e *PMABaseEntity) GetRoomID() *string                    { return e.RoomID }
func (e *PMABaseEntity) GetAreaID() *string                    { return e.AreaID }
func (e *PMABaseEntity) GetDeviceID() *string                  { return e.DeviceID }
func (e *PMABaseEntity) GetMetadata() *PMAMetadata             { return e.Metadata }
func (e *PMABaseEntity) IsAvailable() bool                     { return e.Available }

func (e *PMABaseEntity) GetSource() PMASourceType {
	if e.Metadata != nil {
		return e.Metadata.Source
	}
	return SourcePMA
}

func (e *PMABaseEntity) GetQualityScore() float64 {
	if e.Metadata != nil {
		return e.Metadata.QualityScore
	}
	return 1.0
}

func (e *PMABaseEntity) HasCapability(capability PMACapability) bool {
	for _, cap := range e.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

func (e *PMABaseEntity) CanControl() bool {
	return e.Available && len(e.GetAvailableActions()) > 0
}

func (e *PMABaseEntity) GetAvailableActions() []string {
	// Base implementation - to be overridden by specific entity types
	actions := []string{}
	if e.HasCapability(CapabilityDimmable) {
		actions = append(actions, "turn_on", "turn_off", "set_brightness")
	} else if e.State == StateOn || e.State == StateOff {
		actions = append(actions, "turn_on", "turn_off", "toggle")
	}
	return actions
}

func (e *PMABaseEntity) ExecuteAction(action PMAControlAction) (*PMAControlResult, error) {
	// Base implementation - to be overridden by specific entity types
	return &PMAControlResult{
		Success:     false,
		EntityID:    e.ID,
		Action:      action.Action,
		ProcessedAt: time.Now(),
		Error: &PMAError{
			Code:    "NOT_IMPLEMENTED",
			Message: "Action execution not implemented for base entity",
			Source:  string(e.GetSource()),
		},
	}, nil
}

func (e *PMABaseEntity) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// Specific entity type interfaces
type PMALight interface {
	PMAEntity
	// Light-specific methods
	GetBrightness() *int
	GetColorMode() string
	GetRGBColor() *[3]int
	GetColorTemp() *int
	SetBrightness(brightness int) error
	SetRGBColor(r, g, b int) error
	SetColorTemp(temp int) error
}

type PMASwitch interface {
	PMAEntity
	// Switch-specific methods
	TurnOn() error
	TurnOff() error
	Toggle() error
}

type PMASensor interface {
	PMAEntity
	// Sensor-specific methods
	GetUnit() string
	GetDeviceClass() string
	GetNumericValue() *float64
	GetStringValue() string
	GetLastMeasurement() time.Time
}

type PMAClimate interface {
	PMAEntity
	// Climate-specific methods
	GetCurrentTemperature() *float64
	GetTargetTemperature() *float64
	GetHumidity() *float64
	GetHVACMode() string
	SetTemperature(temp float64) error
	SetHVACMode(mode string) error
}

type PMACover interface {
	PMAEntity
	// Cover-specific methods
	GetPosition() *int
	Open() error
	Close() error
	SetPosition(position int) error
}

type PMACamera interface {
	PMAEntity
	// Camera-specific methods
	GetStreamURL() string
	GetSnapshotURL() string
	IsRecording() bool
	StartRecording() error
	StopRecording() error
	TakeSnapshot() ([]byte, error)
}

type PMADevice interface {
	PMAEntity
	// Device-specific methods
	GetManufacturer() string
	GetModel() string
	GetSWVersion() string
	GetHWVersion() string
	GetConnections() []string
	GetIdentifiers() []string
	GetConfigurationURL() string
}

// PMALight implementation
type PMALightEntity struct {
	*PMABaseEntity
	Brightness *int    `json:"brightness,omitempty"`
	ColorMode  string  `json:"color_mode,omitempty"`
	RGBColor   *[3]int `json:"rgb_color,omitempty"`
	ColorTemp  *int    `json:"color_temp,omitempty"`
}

func (l *PMALightEntity) GetBrightness() *int  { return l.Brightness }
func (l *PMALightEntity) GetColorMode() string { return l.ColorMode }
func (l *PMALightEntity) GetRGBColor() *[3]int { return l.RGBColor }
func (l *PMALightEntity) GetColorTemp() *int   { return l.ColorTemp }

func (l *PMALightEntity) SetBrightness(brightness int) error {
	action := PMAControlAction{
		Action:   "set_brightness",
		EntityID: l.ID,
		Parameters: map[string]interface{}{
			"brightness": brightness,
		},
	}
	result, err := l.ExecuteAction(action)
	if err != nil {
		return err
	}
	if !result.Success {
		return fmt.Errorf("failed to set brightness: %s", result.Error.Message)
	}
	l.Brightness = &brightness
	return nil
}

func (l *PMALightEntity) SetRGBColor(r, g, b int) error {
	color := [3]int{r, g, b}
	action := PMAControlAction{
		Action:   "set_rgb_color",
		EntityID: l.ID,
		Parameters: map[string]interface{}{
			"rgb_color": color,
		},
	}
	result, err := l.ExecuteAction(action)
	if err != nil {
		return err
	}
	if !result.Success {
		return fmt.Errorf("failed to set RGB color: %s", result.Error.Message)
	}
	l.RGBColor = &color
	return nil
}

func (l *PMALightEntity) SetColorTemp(temp int) error {
	action := PMAControlAction{
		Action:   "set_color_temp",
		EntityID: l.ID,
		Parameters: map[string]interface{}{
			"color_temp": temp,
		},
	}
	result, err := l.ExecuteAction(action)
	if err != nil {
		return err
	}
	if !result.Success {
		return fmt.Errorf("failed to set color temperature: %s", result.Error.Message)
	}
	l.ColorTemp = &temp
	return nil
}

// PMASwitch implementation
type PMASwitchEntity struct {
	*PMABaseEntity
}

func (s *PMASwitchEntity) TurnOn() error {
	action := PMAControlAction{
		Action:   "turn_on",
		EntityID: s.ID,
	}
	result, err := s.ExecuteAction(action)
	if err != nil {
		return err
	}
	if !result.Success {
		return fmt.Errorf("failed to turn on: %s", result.Error.Message)
	}
	s.State = StateOn
	return nil
}

func (s *PMASwitchEntity) TurnOff() error {
	action := PMAControlAction{
		Action:   "turn_off",
		EntityID: s.ID,
	}
	result, err := s.ExecuteAction(action)
	if err != nil {
		return err
	}
	if !result.Success {
		return fmt.Errorf("failed to turn off: %s", result.Error.Message)
	}
	s.State = StateOff
	return nil
}

func (s *PMASwitchEntity) Toggle() error {
	if s.State == StateOn {
		return s.TurnOff()
	}
	return s.TurnOn()
}

// PMASensor implementation
type PMASensorEntity struct {
	*PMABaseEntity
	Unit            string    `json:"unit,omitempty"`
	DeviceClass     string    `json:"device_class,omitempty"`
	NumericValue    *float64  `json:"numeric_value,omitempty"`
	StringValue     string    `json:"string_value,omitempty"`
	LastMeasurement time.Time `json:"last_measurement"`
}

func (s *PMASensorEntity) GetUnit() string               { return s.Unit }
func (s *PMASensorEntity) GetDeviceClass() string        { return s.DeviceClass }
func (s *PMASensorEntity) GetNumericValue() *float64     { return s.NumericValue }
func (s *PMASensorEntity) GetStringValue() string        { return s.StringValue }
func (s *PMASensorEntity) GetLastMeasurement() time.Time { return s.LastMeasurement }
