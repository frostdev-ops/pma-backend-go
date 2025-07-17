package bluetooth

import (
	"encoding/json"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
)

// DeviceType represents the type of Bluetooth device
type DeviceType string

const (
	DeviceTypePhone      DeviceType = "phone"
	DeviceTypeComputer   DeviceType = "computer"
	DeviceTypeAudio      DeviceType = "audio"
	DeviceTypeInput      DeviceType = "input"
	DeviceTypeHealth     DeviceType = "health"
	DeviceTypeAutomotive DeviceType = "automotive"
	DeviceTypeWearable   DeviceType = "wearable"
	DeviceTypeNetwork    DeviceType = "network"
	DeviceTypeDevice     DeviceType = "device"
	DeviceTypeUnknown    DeviceType = "unknown"
)

// PairingMethod represents the Bluetooth pairing authentication method
type PairingMethod string

const (
	PairingMethodPIN     PairingMethod = "pin"
	PairingMethodSSP     PairingMethod = "ssp"
	PairingMethodPasskey PairingMethod = "passkey"
	PairingMethodConfirm PairingMethod = "confirm"
	PairingMethodNone    PairingMethod = "none"
)

// PairingStatus represents the status of a pairing session
type PairingStatus string

const (
	PairingStatusPending   PairingStatus = "pending"
	PairingStatusPinReq    PairingStatus = "pin_required"
	PairingStatusConfirm   PairingStatus = "confirmation_required"
	PairingStatusSuccess   PairingStatus = "success"
	PairingStatusFailed    PairingStatus = "failed"
	PairingStatusTimedOut  PairingStatus = "timed_out"
	PairingStatusCancelled PairingStatus = "cancelled"
)

// ScanStatus represents the status of a device scan
type ScanStatus string

const (
	ScanStatusIdle    ScanStatus = "idle"
	ScanStatusActive  ScanStatus = "active"
	ScanStatusStopped ScanStatus = "stopped"
)

// BluetoothAdapter represents the system's Bluetooth adapter
type BluetoothAdapter struct {
	Address      string `json:"address"`
	Name         string `json:"name"`
	Alias        string `json:"alias"`
	Class        string `json:"class"`
	Powered      bool   `json:"powered"`
	Discoverable bool   `json:"discoverable"`
	Pairable     bool   `json:"pairable"`
	Discovering  bool   `json:"discovering"`
}

// Device represents a Bluetooth device with extended information
type Device struct {
	Address        string          `json:"address"`
	Name           string          `json:"name"`
	Alias          string          `json:"alias,omitempty"`
	DeviceClass    string          `json:"device_class,omitempty"`
	DeviceType     DeviceType      `json:"device_type"`
	Connected      bool            `json:"connected"`
	Paired         bool            `json:"paired"`
	Trusted        bool            `json:"trusted"`
	Blocked        bool            `json:"blocked"`
	RSSI           *int            `json:"rssi,omitempty"`
	Services       []string        `json:"services"`
	BatteryLevel   *int            `json:"battery_level,omitempty"`
	LastSeen       time.Time       `json:"last_seen"`
	PairedAt       *time.Time      `json:"paired_at,omitempty"`
	Authentication *Authentication `json:"authentication,omitempty"`
	Pairing        *PairingInfo    `json:"pairing,omitempty"`
}

// Authentication represents the authentication method for a device
type Authentication struct {
	Method            PairingMethod `json:"method"`
	PIN               string        `json:"pin,omitempty"`
	ConfirmationValue *int          `json:"confirmation_value,omitempty"`
}

// PairingInfo represents active pairing session information
type PairingInfo struct {
	InProgress           bool          `json:"in_progress"`
	Method               PairingMethod `json:"method,omitempty"`
	PIN                  string        `json:"pin,omitempty"`
	ConfirmationRequired bool          `json:"confirmation_required"`
	ConfirmationValue    *int          `json:"confirmation_value,omitempty"`
}

// PairingSession represents an active pairing session
type PairingSession struct {
	SessionID         string        `json:"session_id"`
	DeviceAddress     string        `json:"device_address"`
	DeviceName        string        `json:"device_name"`
	Method            PairingMethod `json:"method"`
	Status            PairingStatus `json:"status"`
	PIN               string        `json:"pin,omitempty"`
	ConfirmationValue *int          `json:"confirmation_value,omitempty"`
	RequiresPin       bool          `json:"requires_pin"`
	RequiresConfirm   bool          `json:"requires_confirmation"`
	StartTime         time.Time     `json:"start_time"`
	EndTime           *time.Time    `json:"end_time,omitempty"`
	ErrorMessage      string        `json:"error_message,omitempty"`
	AgentRegistered   bool          `json:"agent_registered"`
	TimeoutTimer      interface{}   `json:"-"` // Internal timer object
}

// ScanSession represents an active device scan session
type ScanSession struct {
	Active            bool               `json:"active"`
	StartTime         time.Time          `json:"start_time"`
	Duration          int                `json:"duration_seconds"`
	DevicesFound      int                `json:"devices_found"`
	Status            ScanStatus         `json:"status"`
	DiscoveredDevices map[string]*Device `json:"-"` // Internal device map
}

// BluetoothAvailability represents the system's Bluetooth availability
type BluetoothAvailability struct {
	Available     bool   `json:"available"`
	Error         string `json:"error,omitempty"`
	ServiceActive bool   `json:"service_active"`
	AdapterFound  bool   `json:"adapter_found"`
	BluezVersion  string `json:"bluez_version,omitempty"`
}

// ScanRequest represents a request to scan for devices
type ScanRequest struct {
	Duration      int  `json:"duration" validate:"min=1,max=60"`
	ClearPrevious bool `json:"clear_previous"`
}

// PairRequest represents a request to pair with a device
type PairRequest struct {
	Address string        `json:"address" validate:"required,bluetooth_address"`
	Method  PairingMethod `json:"method,omitempty"`
	PIN     string        `json:"pin,omitempty"`
	Timeout int           `json:"timeout,omitempty"`
}

// ConnectRequest represents a request to connect to a device
type ConnectRequest struct {
	Address     string `json:"address" validate:"required,bluetooth_address"`
	AutoConnect bool   `json:"auto_connect"`
	Trust       bool   `json:"trust"`
}

// PairResponse represents the response from a pairing operation
type PairResponse struct {
	Success              bool   `json:"success"`
	SessionID            string `json:"session_id,omitempty"`
	RequiresPin          bool   `json:"requires_pin"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
	ConfirmationValue    *int   `json:"confirmation_value,omitempty"`
	Message              string `json:"message"`
}

// DeviceStats represents Bluetooth device statistics
type DeviceStats struct {
	TotalDevices     int `json:"total_devices"`
	PairedDevices    int `json:"paired_devices"`
	ConnectedDevices int `json:"connected_devices"`
	RecentDevices    int `json:"recent_devices"` // Seen in last 24h
}

// ServiceCapabilities represents the Bluetooth service capabilities
type ServiceCapabilities struct {
	Available        bool     `json:"available"`
	ScanSupported    bool     `json:"scan_supported"`
	PairSupported    bool     `json:"pair_supported"`
	ConnectSupported bool     `json:"connect_supported"`
	SupportedMethods []string `json:"supported_methods"`
	MaxScanDuration  int      `json:"max_scan_duration"`
	Platform         string   `json:"platform"`
}

// ToDevice converts a database model to a service Device
func (d *Device) ToModel() *models.BluetoothDevice {
	var servicesJSON json.RawMessage
	if len(d.Services) > 0 {
		servicesData, _ := json.Marshal(d.Services)
		servicesJSON = servicesData
	}

	model := &models.BluetoothDevice{
		Address:     d.Address,
		DeviceClass: toNullString(d.DeviceClass),
		IsPaired:    d.Paired,
		IsConnected: d.Connected,
		Services:    servicesJSON,
		LastSeen:    d.LastSeen,
	}

	if d.Name != "" {
		model.Name.String = d.Name
		model.Name.Valid = true
	}

	if d.PairedAt != nil {
		model.PairedAt.Time = *d.PairedAt
		model.PairedAt.Valid = true
	}

	return model
}

// FromModel converts a database model to a service Device
func DeviceFromModel(model *models.BluetoothDevice) *Device {
	device := &Device{
		Address:     model.Address,
		Name:        model.Name.String,
		Connected:   model.IsConnected,
		Paired:      model.IsPaired,
		DeviceClass: model.DeviceClass.String,
		LastSeen:    model.LastSeen,
		Services:    []string{},
		DeviceType:  DeviceTypeUnknown,
	}

	if model.PairedAt.Valid {
		device.PairedAt = &model.PairedAt.Time
	}

	// Parse services JSON
	if len(model.Services) > 0 {
		var services []string
		if err := json.Unmarshal(model.Services, &services); err == nil {
			device.Services = services
		}
	}

	// Determine device type based on services
	device.DeviceType = determineDeviceType(device.Services)

	return device
}

// Helper function to create sql.NullString
func toNullString(s string) (ns struct {
	String string
	Valid  bool
}) {
	if s != "" {
		ns.String = s
		ns.Valid = true
	}
	return ns
}

// Helper function to determine device type based on services
func determineDeviceType(services []string) DeviceType {
	serviceSet := make(map[string]bool)
	for _, service := range services {
		serviceSet[service] = true
	}

	// Audio devices
	if serviceSet["Audio Sink"] || serviceSet["A2DP Sink"] || serviceSet["Audio Source"] ||
		serviceSet["Advanced Audio Distribution"] || serviceSet["Audio/Video Remote Control"] {
		return DeviceTypeAudio
	}

	// Input devices
	if serviceSet["Human Interface Device"] || serviceSet["HID"] ||
		serviceSet["Keyboard"] || serviceSet["Mouse"] {
		return DeviceTypeInput
	}

	// Phone/mobile devices
	if serviceSet["Dial-up Networking"] || serviceSet["Handsfree"] ||
		serviceSet["SMS"] || serviceSet["Phone Book Access"] {
		return DeviceTypePhone
	}

	// Computer/laptop
	if serviceSet["Network Access Point"] || serviceSet["File Transfer"] ||
		serviceSet["Object Push"] {
		return DeviceTypeComputer
	}

	// Health devices
	if serviceSet["Health Device"] || serviceSet["Heart Rate"] ||
		serviceSet["Fitness Machine"] {
		return DeviceTypeHealth
	}

	// Wearable devices
	if serviceSet["Battery Service"] || serviceSet["Fitness Machine"] ||
		serviceSet["Running Speed and Cadence"] {
		return DeviceTypeWearable
	}

	return DeviceTypeUnknown
}
