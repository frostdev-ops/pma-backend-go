package energy

import (
	"time"
)

// EnergySettings represents configuration for energy monitoring
type EnergySettings struct {
	ID               int       `json:"id" db:"id"`
	EnergyRate       float64   `json:"energy_rate" db:"energy_rate"`             // Cost per kWh
	Currency         string    `json:"currency" db:"currency"`                   // Currency code (USD, EUR, etc.)
	TrackingEnabled  bool      `json:"tracking_enabled" db:"tracking_enabled"`   // Enable/disable tracking
	UpdateInterval   int       `json:"update_interval" db:"update_interval"`     // Update interval in seconds
	HistoricalPeriod int       `json:"historical_period" db:"historical_period"` // Days to keep history
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// EnergyHistory represents a historical energy snapshot
type EnergyHistory struct {
	ID               int       `json:"id" db:"id"`
	Timestamp        time.Time `json:"timestamp" db:"timestamp"`
	PowerConsumption float64   `json:"power_consumption" db:"power_consumption"` // Current power in watts
	EnergyUsage      float64   `json:"energy_usage" db:"energy_usage"`           // Energy usage in kWh
	Cost             float64   `json:"cost" db:"cost"`                           // Cost in currency
	DeviceCount      int       `json:"device_count" db:"device_count"`           // Number of devices tracked
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// DeviceEnergy represents energy consumption for a specific device
type DeviceEnergy struct {
	ID               int       `json:"id" db:"id"`
	EntityID         string    `json:"entity_id" db:"entity_id"`
	DeviceName       string    `json:"device_name" db:"device_name"`
	Room             string    `json:"room" db:"room"`
	PowerConsumption float64   `json:"power_consumption" db:"power_consumption"` // Current power in watts
	EnergyUsage      float64   `json:"energy_usage" db:"energy_usage"`           // Energy usage in kWh
	Cost             float64   `json:"cost" db:"cost"`                           // Cost in currency
	State            string    `json:"state" db:"state"`                         // Device state
	IsOn             bool      `json:"is_on" db:"is_on"`                         // Is device currently on
	Percentage       float64   `json:"percentage" db:"percentage"`               // Percentage of total consumption
	Timestamp        time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// DeviceEnergyConsumption represents comprehensive device energy data
type DeviceEnergyConsumption struct {
	EntityID         string  `json:"entity_id"`
	DeviceName       string  `json:"device_name"`
	Room             string  `json:"room"`
	PowerConsumption float64 `json:"power_consumption"` // Current power in watts
	EnergyUsage      float64 `json:"energy_usage"`      // Energy usage in kWh
	Cost             float64 `json:"cost"`              // Cost in currency
	State            string  `json:"state"`             // Device state
	IsOn             bool    `json:"is_on"`             // Is device currently on
	Percentage       float64 `json:"percentage"`        // Percentage of total consumption

	// Comprehensive energy data (mainly for Shelly devices)
	Current        float64  `json:"current,omitempty"`         // Current in amps
	Voltage        float64  `json:"voltage,omitempty"`         // Voltage in volts
	Frequency      float64  `json:"frequency,omitempty"`       // Frequency in Hz
	ReturnedEnergy float64  `json:"returned_energy,omitempty"` // Returned energy (solar/bidirectional)
	HasSensors     bool     `json:"has_sensors"`               // Whether device has energy sensors
	SensorsFound   []string `json:"sensors_found"`             // List of available sensors
}

// EnergyData represents current energy consumption data
type EnergyData struct {
	Timestamp             time.Time                 `json:"timestamp"`
	TotalPowerConsumption float64                   `json:"total_power_consumption"` // Total power in watts
	TotalEnergyUsage      float64                   `json:"total_energy_usage"`      // Total energy in kWh
	TotalCost             float64                   `json:"total_cost"`              // Total cost
	DeviceBreakdown       []DeviceEnergyConsumption `json:"device_breakdown"`        // Per-device breakdown
	UPSPowerConsumption   float64                   `json:"ups_power_consumption"`   // UPS contribution
}

// EnergyStats represents energy consumption statistics
type EnergyStats struct {
	CurrentPower float64                   `json:"current_power"` // Current total power in watts
	PeakPower    float64                   `json:"peak_power"`    // Peak power consumption
	AveragePower float64                   `json:"average_power"` // Average power consumption
	TotalEnergy  float64                   `json:"total_energy"`  // Total energy consumed
	TotalCost    float64                   `json:"total_cost"`    // Total cost
	Savings      EnergySavings             `json:"savings"`       // Calculated savings
	TopConsumers []DeviceEnergyConsumption `json:"top_consumers"` // Top energy consuming devices
	History      []EnergyHistoryEntry      `json:"history"`       // Recent history
}

// EnergySavings represents calculated energy savings
type EnergySavings struct {
	TotalSavings        float64 `json:"total_savings"`        // Total savings in currency
	AutomationSavings   float64 `json:"automation_savings"`   // Savings from automation
	OptimizationSavings float64 `json:"optimization_savings"` // Savings from optimization
	SchedulingSavings   float64 `json:"scheduling_savings"`   // Savings from scheduling
	PeriodDays          int     `json:"period_days"`          // Period over which savings calculated
}

// EnergyHistoryEntry represents a simplified history entry
type EnergyHistoryEntry struct {
	Timestamp        time.Time `json:"timestamp"`
	PowerConsumption float64   `json:"power_consumption"`
	EnergyUsage      float64   `json:"energy_usage"`
	Cost             float64   `json:"cost"`
}

// ComprehensiveEnergyData represents detailed energy sensor data
type ComprehensiveEnergyData struct {
	Power          float64 `json:"power"`           // Power in watts
	Current        float64 `json:"current"`         // Current in amps
	Energy         float64 `json:"energy"`          // Energy in kWh
	Voltage        float64 `json:"voltage"`         // Voltage in volts
	Frequency      float64 `json:"frequency"`       // Frequency in Hz
	ReturnedEnergy float64 `json:"returned_energy"` // Returned energy (solar/bidirectional)
}

// EnergyRequest represents a request to update energy settings
type EnergySettingsRequest struct {
	EnergyRate       *float64 `json:"energy_rate,omitempty"`
	Currency         *string  `json:"currency,omitempty"`
	TrackingEnabled  *bool    `json:"tracking_enabled,omitempty"`
	UpdateInterval   *int     `json:"update_interval,omitempty"`
	HistoricalPeriod *int     `json:"historical_period,omitempty"`
}

// EnergyHistoryFilter represents filters for energy history queries
type EnergyHistoryFilter struct {
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	DeviceID  *string    `json:"device_id,omitempty"`
	Room      *string    `json:"room,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// DeviceEnergyFilter represents filters for device energy queries
type DeviceEnergyFilter struct {
	EntityID  *string    `json:"entity_id,omitempty"`
	Room      *string    `json:"room,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// EnergyMetrics represents energy monitoring metrics
type EnergyMetrics struct {
	TotalDevicesTracked  int       `json:"total_devices_tracked"`
	ActiveDevices        int       `json:"active_devices"`
	PowerDevices         int       `json:"power_devices"`          // Devices with power sensors
	EnergyDevices        int       `json:"energy_devices"`         // Devices with energy sensors
	ShellyDevices        int       `json:"shelly_devices"`         // Shelly devices detected
	UPSDetected          bool      `json:"ups_detected"`           // UPS system detected
	LastUpdateTime       time.Time `json:"last_update_time"`       // Last energy update
	UpdateInterval       int       `json:"update_interval"`        // Current update interval
	TrackingEnabled      bool      `json:"tracking_enabled"`       // Is tracking enabled
	HistoryRetentionDays int       `json:"history_retention_days"` // History retention period
}

// Default values for energy settings
const (
	DefaultEnergyRate       = 0.12 // $0.12 per kWh (US average)
	DefaultCurrency         = "USD"
	DefaultUpdateInterval   = 30 // 30 seconds
	DefaultHistoricalPeriod = 30 // 30 days
	DefaultTrackingEnabled  = true

	// Energy calculation constants
	HoursPerDay      = 24.0
	DaysPerMonth     = 30.0
	WattsToKilowatts = 1000.0
	SecondsPerHour   = 3600.0
)

// PowerSensorDomains defines entity domains that typically have power sensors
var PowerSensorDomains = []string{
	"switch",
	"light",
	"sensor",
	"climate",
	"cover",
	"fan",
	"water_heater",
	"vacuum",
	"media_player",
}

// PowerSensorAttributes defines common attributes that contain power data
var PowerSensorAttributes = []string{
	"current_power_w",
	"power",
	"current_power",
	"power_consumption",
	"active_power",
}

// EnergySensorAttributes defines common attributes that contain energy data
var EnergySensorAttributes = []string{
	"energy",
	"energy_kwh",
	"total_energy",
	"energy_today",
	"energy_total",
}

// ShellyEnergyPatterns defines patterns for finding Shelly energy sensors
var ShellyEnergyPatterns = map[string][]string{
	"power":           {"_power", "_active_power", "_power_w"},
	"current":         {"_current", "_current_a"},
	"energy":          {"_energy", "_energy_kwh", "_total_energy"},
	"voltage":         {"_voltage", "_voltage_v"},
	"frequency":       {"_frequency", "_frequency_hz"},
	"returned_energy": {"_returned_energy", "_returned_energy_kwh"},
}
